package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	epub "github.com/bmaupin/go-epub"
	readability "github.com/go-shiori/go-readability"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobCaptureURL is the registered name for the URL capture job.
const JobCaptureURL = "capture:url"

// captureURLStagingDir is the subdirectory within the library root used for
// temporarily staging captured EPUB files until the background job processes them.
// It matches the upload handler's staging directory so both upload and capture
// jobs land in the same staging area.
const captureURLStagingDir = ".uploads"

// captureURLHTTPTimeout is the HTTP client timeout used when fetching the page.
const captureURLHTTPTimeout = 30 * time.Second

// captureURLUserAgent is the User-Agent sent when fetching web pages for capture.
const captureURLUserAgent = "Biblioteka/1.0 (+https://github.com/amalgamated-tools/biblioteka)"

// CaptureURLPayload is the JSON payload for the capture:url job.
type CaptureURLPayload struct {
	URL         string `json:"url"`
	LibraryID   string `json:"library_id,omitempty"`
	LibraryRoot string `json:"library_root,omitempty"`
	UserID      string `json:"user_id,omitempty"`

	// Optional metadata overrides supplied by the caller. Non-empty values
	// take precedence over anything parsed from the captured page.
	OverrideTitle       string `json:"override_title,omitempty"`
	OverrideAuthor      string `json:"override_author,omitempty"`
	OverrideDescription string `json:"override_description,omitempty"`
	OverrideLanguage    string `json:"override_language,omitempty"`
	OverridePublisher   string `json:"override_publisher,omitempty"`
}

// fetcher is the interface used to fetch a URL for capture; it exists to allow
// substituting a test HTTP server in unit tests.
type fetcher interface {
	Fetch(ctx context.Context, rawURL string) (title, byline, content string, err error)
}

// httpFetcher is the production fetcher that makes real HTTP requests.
type httpFetcher struct {
	client *http.Client
}

// Fetch retrieves rawURL and extracts the main article content using Readability.
func (f *httpFetcher) Fetch(ctx context.Context, rawURL string) (string, string, string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("create HTTP request: %w", err)
	}
	req.Header.Set("User-Agent", captureURLUserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("fetch URL %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", fmt.Errorf("fetch URL %q: HTTP %d", rawURL, resp.StatusCode)
	}

	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		return "", "", "", fmt.Errorf("parse article from %q: %w", rawURL, err)
	}

	return article.Title, article.Byline, article.Content, nil
}

// defaultFetcher is the production fetcher instance.
var defaultFetcher fetcher = &httpFetcher{
	client: &http.Client{Timeout: captureURLHTTPTimeout},
}

// NewCaptureURLHandler returns a worker.Func that fetches the target URL,
// extracts its article content with go-readability, generates an EPUB file,
// stages it in the library's .uploads directory, and enqueues a process:file
// job so the existing metadata-extraction and organisation pipeline processes it.
func NewCaptureURLHandler(database *db.DB, enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return newCaptureURLHandler(database, enqueuer, defaultFetcher)
}

func newCaptureURLHandler(database *db.DB, enqueuer Enqueuer, f fetcher) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p CaptureURLPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal capture:url payload", slog.Any(otelkeys.Error, err))
			return fmt.Errorf("unmarshal capture:url payload: %w", err)
		}

		slog.DebugContext(ctx, "capture:url job received",
			slog.String(otelkeys.URL, p.URL),
			slog.String(otelkeys.LibraryID, p.LibraryID),
		)

		return captureURL(ctx, database, enqueuer, f, p)
	}
}

// captureURL implements the full URL-capture pipeline:
//  1. Fetch the page and extract article text with go-readability.
//  2. Build an EPUB from the article content with go-epub.
//  3. Stage the EPUB in the library's .uploads directory.
//  4. Enqueue a process:file job so the standard pipeline picks it up.
func captureURL(ctx context.Context, database *db.DB, enqueuer Enqueuer, f fetcher, p CaptureURLPayload) error {
	if p.URL == "" {
		return fmt.Errorf("capture:url payload url is required")
	}
	if p.LibraryRoot == "" {
		return fmt.Errorf("capture:url payload library_root is required")
	}

	// Fetch and extract article content.
	pageTitle, byline, content, err := f.Fetch(ctx, p.URL)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch and parse URL",
			slog.String(otelkeys.URL, p.URL),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("fetch URL %q: %w", p.URL, err)
	}

	// Apply caller-supplied overrides.
	title := pageTitle
	if p.OverrideTitle != "" {
		title = p.OverrideTitle
	}
	if title == "" {
		title = p.URL
	}

	author := byline
	if p.OverrideAuthor != "" {
		author = p.OverrideAuthor
	}

	// Build the EPUB.
	e := epub.NewEpub(title)
	if author != "" {
		e.SetAuthor(author)
	}
	if p.OverrideDescription != "" {
		e.SetDescription(p.OverrideDescription)
	}
	if p.OverrideLanguage != "" {
		e.SetLang(p.OverrideLanguage)
	}

	// Wrap the raw article HTML in a minimal section body.
	sectionBody := content
	if sectionBody == "" {
		sectionBody = "<p>No content could be extracted from the page.</p>"
	}
	if _, err := e.AddSection(sectionBody, title, "", ""); err != nil {
		slog.ErrorContext(ctx, "failed to add EPUB section",
			slog.String(otelkeys.URL, p.URL),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("add EPUB section for %q: %w", p.URL, err)
	}

	// Stage the EPUB in the library's .uploads directory.
	stagingDir := filepath.Join(p.LibraryRoot, captureURLStagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		slog.ErrorContext(ctx, "failed to create capture staging directory",
			slog.String(otelkeys.Path, stagingDir),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create capture staging directory %q: %w", stagingDir, err)
	}

	prefix, err := generatePrefix()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate staging filename prefix",
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("generate staging prefix: %w", err)
	}

	safeTitle := sanitizeFilename(title)
	if safeTitle == "" {
		safeTitle = "captured"
	}
	fileName := safeTitle + ".epub"
	stagingPath := filepath.Join(stagingDir, prefix+"_"+fileName)

	if err := e.Write(stagingPath); err != nil {
		slog.ErrorContext(ctx, "failed to write EPUB to staging path",
			slog.String(otelkeys.Path, stagingPath),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("write EPUB to %q: %w", stagingPath, err)
	}

	fileInfo, err := os.Stat(stagingPath)
	if err != nil {
		return fmt.Errorf("stat staged EPUB: %w", err)
	}

	// Enqueue a process:file job so the existing pipeline handles the rest.
	enqueueCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	processPayload := ProcessFilePayload{
		Path:                stagingPath,
		FileName:            fileName,
		FileType:            "epub",
		FileSize:            fileInfo.Size(),
		LibraryID:           p.LibraryID,
		LibraryRoot:         p.LibraryRoot,
		UserID:              p.UserID,
		OverrideTitle:       title,
		OverrideAuthor:      author,
		OverrideDescription: p.OverrideDescription,
		OverrideLanguage:    p.OverrideLanguage,
		OverridePublisher:   p.OverridePublisher,
	}

	if _, err := enqueuer.Enqueue(enqueueCtx, JobProcessFile, processPayload); err != nil {
		// Best-effort cleanup to avoid orphaned staged files.
		if rmErr := os.Remove(stagingPath); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.WarnContext(ctx, "failed to remove staged EPUB after enqueue failure",
				slog.String(otelkeys.Path, stagingPath),
				slog.Any(otelkeys.Error, rmErr),
			)
		}
		slog.ErrorContext(ctx, "failed to enqueue process:file job after URL capture",
			slog.String(otelkeys.URL, p.URL),
			slog.String(otelkeys.Path, stagingPath),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("enqueue process:file for captured URL %q: %w", p.URL, err)
	}

	// Best-effort audit log — log warning but do not fail the job.
	if database != nil {
		if auditErr := database.CreateAuditLog(ctx, p.UserID, db.AuditActionBookCaptured, "book_capture", p.URL, map[string]any{
			"url":       p.URL,
			"title":     title,
			"file_name": fileName,
			"file_size": fileInfo.Size(),
		}); auditErr != nil {
			slog.WarnContext(ctx, "failed to write audit log for captured URL",
				slog.String(otelkeys.URL, p.URL),
				slog.Any(otelkeys.Error, auditErr),
			)
		}
	}

	slog.InfoContext(ctx, "URL captured and EPUB staged",
		slog.String(otelkeys.URL, p.URL),
		slog.String(otelkeys.Title, title),
		slog.String(otelkeys.Path, stagingPath),
		slog.Int64(otelkeys.FileSize, fileInfo.Size()),
	)

	return nil
}

// generatePrefix returns a random hex string suitable for use as a filename
// prefix to avoid collisions.
func generatePrefix() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// sanitizeFilename removes characters that are invalid in file names from s
// and truncates the result to 100 bytes.
func sanitizeFilename(s string) string {
	var b []byte
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' || r == '\x00':
			b = append(b, '_')
		case r < 0x20:
			// skip control characters
		default:
			b = utf8.AppendRune(b, r)
		}
	}
	if len(b) > 100 {
		b = b[:100]
	}
	return string(b)
}
