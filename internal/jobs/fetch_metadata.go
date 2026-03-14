package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// JobFetchMetadata is the registered name for the metadata-fetching job.
const JobFetchMetadata = "fetch:metadata"

// FetchMetadataPayload is the JSON payload for the fetch:metadata job.
type FetchMetadataPayload struct {
	BookID string `json:"book_id"`
}

// BookMetadata holds metadata fields returned by an external source.
// All fields are pointers so that only provided values are applied.
type BookMetadata struct {
	Title           *string
	Description     *string
	GoodreadsID     *string
	HardcoverID     *string
	PublicationDate *string
	Publisher       *string
	Language        *string
	NumPages        *int
	CoverImageURL   *string
}

// MetadataFetcher is the interface implemented by external metadata sources.
type MetadataFetcher interface {
	// Name returns a human-readable identifier used in log messages.
	Name() string
	// Fetch retrieves metadata for the given book.
	// Returns nil, nil when the source is not configured or no result is found.
	Fetch(ctx context.Context, book *db.Book) (*BookMetadata, error)
}

// BookGetterUpdater is the subset of db.DB used by the metadata job.
type BookGetterUpdater interface {
	GetBook(id string) (*db.Book, error)
	UpdateBook(id, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*db.Book, error)
}

// NewFetchMetadataHandler returns a worker.Func that fetches metadata for a book
// from each configured fetcher and persists the merged result.
func NewFetchMetadataHandler(database BookGetterUpdater, fetchers ...MetadataFetcher) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p FetchMetadataPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal fetch metadata payload: %w", err)
		}

		if p.BookID == "" {
			return fmt.Errorf("fetch metadata payload: book_id is required")
		}

		slog.InfoContext(ctx, "fetching metadata for book", slog.String("book_id", p.BookID))

		book, err := database.GetBook(p.BookID)
		if err != nil {
			return fmt.Errorf("get book %s: %w", p.BookID, err)
		}

		merged := &BookMetadata{}
		for _, f := range fetchers {
			meta, fetchErr := f.Fetch(ctx, book)
			if fetchErr != nil {
				slog.WarnContext(ctx, "metadata fetch failed",
					slog.String("source", f.Name()),
					slog.String("book_id", p.BookID),
					slog.Any("error", fetchErr),
				)
				continue
			}
			if meta == nil {
				slog.DebugContext(ctx, "no metadata returned",
					slog.String("source", f.Name()),
					slog.String("book_id", p.BookID),
				)
				continue
			}
			slog.InfoContext(ctx, "metadata fetched",
				slog.String("source", f.Name()),
				slog.String("book_id", p.BookID),
			)
			applyMetadata(merged, meta)
		}

		// Resolve final values: prefer freshly fetched data, fall back to what the
		// book already has so that UpdateBook never wipes existing information.
		title := book.Title
		if merged.Title != nil {
			title = *merged.Title
		}

		_, err = database.UpdateBook(
			book.ID,
			title,
			firstNonNilStr(merged.Description, book.Description),
			book.ASIN,
			book.ISBN10,
			book.ISBN13,
			firstNonNilStr(merged.GoodreadsID, book.GoodreadsID),
			firstNonNilStr(merged.HardcoverID, book.HardcoverID),
			book.GoogleBooksID,
			firstNonNilStr(merged.PublicationDate, book.PublicationDate),
			firstNonNilStr(merged.Publisher, book.Publisher),
			firstNonNilStr(merged.Language, book.Language),
			firstNonNilInt(merged.NumPages, book.NumPages),
			firstNonNilStr(merged.CoverImageURL, book.CoverImageURL),
		)
		if err != nil {
			return fmt.Errorf("update book %s with metadata: %w", p.BookID, err)
		}

		slog.InfoContext(ctx, "book metadata updated", slog.String("book_id", p.BookID))
		return nil
	}
}

// applyMetadata copies non-nil fields from src into dst.
func applyMetadata(dst, src *BookMetadata) {
	if src.Title != nil {
		dst.Title = src.Title
	}
	if src.Description != nil {
		dst.Description = src.Description
	}
	if src.GoodreadsID != nil {
		dst.GoodreadsID = src.GoodreadsID
	}
	if src.HardcoverID != nil {
		dst.HardcoverID = src.HardcoverID
	}
	if src.PublicationDate != nil {
		dst.PublicationDate = src.PublicationDate
	}
	if src.Publisher != nil {
		dst.Publisher = src.Publisher
	}
	if src.Language != nil {
		dst.Language = src.Language
	}
	if src.NumPages != nil {
		dst.NumPages = src.NumPages
	}
	if src.CoverImageURL != nil {
		dst.CoverImageURL = src.CoverImageURL
	}
}

// firstNonNilStr returns the first non-nil pointer from the list.
func firstNonNilStr(ptrs ...*string) *string {
	for _, p := range ptrs {
		if p != nil {
			return p
		}
	}
	return nil
}

// firstNonNilInt returns the first non-nil int pointer from the list.
func firstNonNilInt(ptrs ...*int) *int {
	for _, p := range ptrs {
		if p != nil {
			return p
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Goodreads client
// ---------------------------------------------------------------------------

// GoodreadsClient fetches book metadata from the Goodreads search XML API.
// Configure by setting the GOODREADS_API_KEY environment variable.
type GoodreadsClient struct {
	APIKey     string
	HTTPClient *http.Client
	baseURL    string // overridable for tests; defaults to https://www.goodreads.com
}

// NewGoodreadsClient creates a GoodreadsClient configured from the environment.
func NewGoodreadsClient() *GoodreadsClient {
	return &GoodreadsClient{
		APIKey:     os.Getenv("GOODREADS_API_KEY"),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://www.goodreads.com",
	}
}

func (c *GoodreadsClient) Name() string { return "goodreads" }

// goodreadsSearchResponse is the XML envelope returned by the Goodreads
// search endpoint (https://www.goodreads.com/search/index.xml).
type goodreadsSearchResponse struct {
	XMLName xml.Name `xml:"GoodreadsResponse"`
	Search  struct {
		Results struct {
			Works []struct {
				BestBook struct {
					ID       int    `xml:"id"`
					Title    string `xml:"title"`
					ImageURL string `xml:"image_url"`
					Author   struct {
						Name string `xml:"name"`
					} `xml:"author"`
				} `xml:"best_book"`
				OriginalPublicationYear string `xml:"original_publication_year"`
			} `xml:"work"`
		} `xml:"results"`
	} `xml:"search"`
}

// Fetch queries the Goodreads search API and returns the top result.
// Returns nil, nil when the API key is not set or no result is found.
func (c *GoodreadsClient) Fetch(ctx context.Context, book *db.Book) (*BookMetadata, error) {
	if c.APIKey == "" {
		return nil, nil
	}

	query := buildSearchQuery(book)
	if query == "" {
		return nil, nil
	}

	base := c.baseURL
	if base == "" {
		base = "https://www.goodreads.com"
	}

	reqURL := base + "/search/index.xml?key=" + url.QueryEscape(c.APIKey) + "&q=" + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build goodreads request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("goodreads request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("goodreads returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read goodreads response: %w", err)
	}

	return parseGoodreadsBody(body)
}

// parseGoodreadsBody parses the XML body returned by the Goodreads search API.
func parseGoodreadsBody(body []byte) (*BookMetadata, error) {
	var result goodreadsSearchResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse goodreads response: %w", err)
	}

	if len(result.Search.Results.Works) == 0 {
		return nil, nil
	}

	best := result.Search.Results.Works[0].BestBook
	grID := fmt.Sprintf("%d", best.ID)

	meta := &BookMetadata{
		GoodreadsID: &grID,
	}

	if best.Title != "" {
		meta.Title = &best.Title
	}
	if best.ImageURL != "" && !strings.Contains(best.ImageURL, "nophoto") {
		meta.CoverImageURL = &best.ImageURL
	}
	if pubYear := result.Search.Results.Works[0].OriginalPublicationYear; pubYear != "" {
		meta.PublicationDate = &pubYear
	}

	return meta, nil
}

// ---------------------------------------------------------------------------
// Hardcover client
// ---------------------------------------------------------------------------

// HardcoverClient fetches book metadata from the Hardcover GraphQL API
// (https://api.hardcover.app/v1/graphql).
// Configure by setting the HARDCOVER_API_TOKEN environment variable.
type HardcoverClient struct {
	APIToken   string
	HTTPClient *http.Client
	baseURL    string // overridable for tests; defaults to https://api.hardcover.app
}

// NewHardcoverClient creates a HardcoverClient configured from the environment.
func NewHardcoverClient() *HardcoverClient {
	return &HardcoverClient{
		APIToken:   os.Getenv("HARDCOVER_API_TOKEN"),
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://api.hardcover.app",
	}
}

func (c *HardcoverClient) Name() string { return "hardcover" }

const hardcoverSearchQuery = `
query SearchBooks($query: String!) {
  search(query: $query, query_type: "Book", per_page: 1) {
    results
  }
}`

// hardcoverSearchRequest is the JSON body for a Hardcover GraphQL request.
type hardcoverSearchRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// hardcoverSearchResponse is the top-level GraphQL response envelope.
type hardcoverSearchResponse struct {
	Data struct {
		Search struct {
			Results json.RawMessage `json:"results"`
		} `json:"search"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// hardcoverBook represents a single book entry inside the Hardcover search
// results JSON.  The API returns results as a nested JSON array/object, so we
// parse only the fields we care about.
type hardcoverBook struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Pages       int    `json:"pages"`
	ReleaseDate string `json:"release_date"`
	Language    struct {
		Language string `json:"language"`
	} `json:"language"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
}

// Fetch queries the Hardcover GraphQL API and returns the top result.
// Returns nil, nil when the API token is not set or no result is found.
func (c *HardcoverClient) Fetch(ctx context.Context, book *db.Book) (*BookMetadata, error) {
	if c.APIToken == "" {
		return nil, nil
	}

	query := buildSearchQuery(book)
	if query == "" {
		return nil, nil
	}

	reqBody := hardcoverSearchRequest{
		Query:     hardcoverSearchQuery,
		Variables: map[string]any{"query": query},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal hardcover request: %w", err)
	}

	base := c.baseURL
	if base == "" {
		base = "https://api.hardcover.app"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build hardcover request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hardcover request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hardcover returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read hardcover response: %w", err)
	}

	var result hardcoverSearchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse hardcover response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("hardcover API error: %s", result.Errors[0].Message)
	}

	// results is returned as a JSON array at the top level
	var books []hardcoverBook
	if err := json.Unmarshal(result.Data.Search.Results, &books); err != nil {
		return nil, fmt.Errorf("parse hardcover results: %w", err)
	}

	if len(books) == 0 {
		return nil, nil
	}

	hc := books[0]
	hcID := fmt.Sprintf("%d", hc.ID)
	meta := &BookMetadata{
		HardcoverID: &hcID,
	}

	if hc.Title != "" {
		meta.Title = &hc.Title
	}
	if hc.Description != "" {
		meta.Description = &hc.Description
	}
	if hc.Pages > 0 {
		pages := hc.Pages
		meta.NumPages = &pages
	}
	if hc.ReleaseDate != "" {
		meta.PublicationDate = &hc.ReleaseDate
	}
	if hc.Language.Language != "" {
		meta.Language = &hc.Language.Language
	}
	if hc.Image.URL != "" {
		meta.CoverImageURL = &hc.Image.URL
	}

	return meta, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildSearchQuery constructs a search string for a book using its ISBN or title.
func buildSearchQuery(book *db.Book) string {
	if book.ISBN13 != nil && *book.ISBN13 != "" {
		return *book.ISBN13
	}
	if book.ISBN10 != nil && *book.ISBN10 != "" {
		return *book.ISBN10
	}
	return strings.TrimSpace(book.Title)
}
