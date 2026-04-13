package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureHTTPFetcher wraps a test HTTP server as a fetcher.
type captureHTTPFetcher struct {
	title   string
	byline  string
	content string
	err     error
}

func (f *captureHTTPFetcher) Fetch(_ context.Context, _ string) (string, string, string, error) {
	return f.title, f.byline, f.content, f.err
}

// TestCaptureURL_Success verifies the happy path: EPUB staged, process:file enqueued.
func TestCaptureURL_Success(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{}

	f := &captureHTTPFetcher{
		title:   "Test Article",
		byline:  "Jane Doe",
		content: "<p>Hello world</p>",
	}

	p := CaptureURLPayload{
		URL:         "https://example.com/article",
		LibraryID:   "lib-1",
		LibraryRoot: dir,
		UserID:      "user-1",
	}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.NoError(t, err)

	// A process:file job should have been enqueued.
	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)
	require.Equal(t, JobProcessFile, enq.jobs[0].Name)

	var payload ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.Equal(t, "epub", payload.FileType)
	require.Equal(t, "lib-1", payload.LibraryID)
	require.Equal(t, "user-1", payload.UserID)
	require.Equal(t, "Test Article", payload.OverrideTitle)
	require.Equal(t, "Jane Doe", payload.OverrideAuthor)
	require.NotEmpty(t, payload.Path, "staged path must be set")
	require.True(t, strings.HasSuffix(payload.FileName, ".epub"), "staged file must be an epub")

	// The staged EPUB should exist on disk.
	_, statErr := os.Stat(payload.Path)
	require.NoError(t, statErr, "staged EPUB should exist")
}

// TestCaptureURL_MetadataOverrides verifies caller-supplied overrides are applied.
func TestCaptureURL_MetadataOverrides(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{}

	f := &captureHTTPFetcher{
		title:   "Page Title",
		byline:  "Page Author",
		content: "<p>Content</p>",
	}

	p := CaptureURLPayload{
		URL:                 "https://example.com/article",
		LibraryID:           "lib-1",
		LibraryRoot:         dir,
		OverrideTitle:       "My Override Title",
		OverrideAuthor:      "My Override Author",
		OverrideDescription: "My description",
		OverrideLanguage:    "fr",
		OverridePublisher:   "My Press",
	}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.NoError(t, err)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)

	var payload ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.Equal(t, "My Override Title", payload.OverrideTitle)
	require.Equal(t, "My Override Author", payload.OverrideAuthor)
	require.Equal(t, "My description", payload.OverrideDescription)
	require.Equal(t, "fr", payload.OverrideLanguage)
	require.Equal(t, "My Press", payload.OverridePublisher)
}

// TestCaptureURL_FetchError verifies that fetch failures are propagated.
func TestCaptureURL_FetchError(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{}

	f := &captureHTTPFetcher{err: errors.New("connection refused")}

	p := CaptureURLPayload{
		URL:         "https://example.com/article",
		LibraryRoot: dir,
	}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch URL")

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Empty(t, enq.jobs, "no job should be enqueued on fetch failure")
}

// TestCaptureURL_EnqueueFailureCleansStagedFile verifies staged files are
// removed when the downstream process:file job cannot be enqueued.
func TestCaptureURL_EnqueueFailureCleansStagedFile(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{err: errors.New("redis unavailable")}

	f := &captureHTTPFetcher{
		title:   "Article",
		content: "<p>Body</p>",
	}

	p := CaptureURLPayload{
		URL:         "https://example.com/article",
		LibraryRoot: dir,
	}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.Error(t, err)

	// No EPUB should remain in the staging directory.
	entries, readErr := os.ReadDir(filepath.Join(dir, captureURLStagingDir))
	if readErr == nil {
		for _, e := range entries {
			require.False(t, strings.HasSuffix(e.Name(), ".epub"), "staged epub should be removed after enqueue failure")
		}
	}
}

// TestCaptureURL_MissingURL verifies validation rejects an empty URL.
func TestCaptureURL_MissingURL(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{}
	f := &captureHTTPFetcher{}

	p := CaptureURLPayload{LibraryRoot: dir}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "url is required")
}

// TestCaptureURL_MissingLibraryRoot verifies validation rejects an empty library root.
func TestCaptureURL_MissingLibraryRoot(t *testing.T) {
	enq := &genericMockEnqueuer{}
	f := &captureHTTPFetcher{}

	p := CaptureURLPayload{URL: "https://example.com/article"}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "library_root is required")
}

// TestCaptureURL_EmptyContent verifies that pages with no extractable content
// still produce a valid EPUB.
func TestCaptureURL_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	enq := &genericMockEnqueuer{}

	f := &captureHTTPFetcher{title: "Empty Page", content: ""}

	p := CaptureURLPayload{
		URL:         "https://example.com/empty",
		LibraryRoot: dir,
	}

	err := captureURL(t.Context(), nil, enq, f, p)
	require.NoError(t, err)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)
}

// TestCaptureURL_RealHTTPServer tests the real HTTP fetcher against a local
// test server to validate that the integration path works end-to-end without
// external network access.
func TestCaptureURL_RealHTTPServer(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>Real Article</title></head>
<body><article><p>This is the article body.</p></article></body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintln(w, html)
	}))
	defer srv.Close()

	dir := t.TempDir()
	enq := &genericMockEnqueuer{}

	handler := newCaptureURLHandler(nil, enq, &httpFetcher{client: srv.Client()})

	payload := CaptureURLPayload{
		URL:         srv.URL + "/article",
		LibraryRoot: dir,
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	err = handler(t.Context(), data)
	require.NoError(t, err)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)
	require.Equal(t, JobProcessFile, enq.jobs[0].Name)
}

// TestSanitizeFilename verifies path-traversal and reserved-character stripping.
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal title", "normal title"},
		{"with/slash", "with_slash"},
		{"with\\backslash", "with_backslash"},
		{"with:colon", "with_colon"},
		{"with*star", "with_star"},
		{"with?question", "with_question"},
		{`with"quote`, "with_quote"},
		{"with<less", "with_less"},
		{"with>greater", "with_greater"},
		{"with|pipe", "with_pipe"},
		{"with\x00null", "with_null"},
		{strings.Repeat("a", 120), strings.Repeat("a", 100)},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			require.Equal(t, tc.want, sanitizeFilename(tc.input))
		})
	}
}

// TestNewCaptureURLHandler_MissingPayload verifies the handler returns an error
// for an invalid JSON payload.
func TestNewCaptureURLHandler_MissingPayload(t *testing.T) {
	handler := NewCaptureURLHandler(nil, &genericMockEnqueuer{})
	err := handler(t.Context(), []byte("not json"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal")
}
