package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mockBookGetterUpdater is an in-memory implementation of BookGetterUpdater for tests.
type mockBookGetterUpdater struct {
	books   map[string]*db.Book
	updated *db.Book
}

func (m *mockBookGetterUpdater) GetBook(id string) (*db.Book, error) {
	b, ok := m.books[id]
	if !ok {
		return nil, fmt.Errorf("book not found: %s", id)
	}
	return b, nil
}

func (m *mockBookGetterUpdater) UpdateBook(id, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*db.Book, error) {
	b := *m.books[id]
	b.Title = title
	b.Description = description
	b.ASIN = asin
	b.ISBN10 = isbn10
	b.ISBN13 = isbn13
	b.GoodreadsID = goodreadsID
	b.HardcoverID = hardcoverID
	b.GoogleBooksID = googleBooksID
	b.PublicationDate = publicationDate
	b.Publisher = publisher
	b.Language = language
	b.NumPages = numPages
	b.CoverImageURL = coverImageURL
	m.updated = &b
	return &b, nil
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func newMockStore(books ...*db.Book) *mockBookGetterUpdater {
	m := &mockBookGetterUpdater{books: make(map[string]*db.Book)}
	for _, b := range books {
		m.books[b.ID] = b
	}
	return m
}

// ---------------------------------------------------------------------------
// NewFetchMetadataHandler tests
// ---------------------------------------------------------------------------

func TestFetchMetadataHandler_MissingPayload(t *testing.T) {
	store := newMockStore()
	handler := NewFetchMetadataHandler(store)

	err := handler(context.Background(), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for missing book_id")
	}
}

func TestFetchMetadataHandler_UnknownBook(t *testing.T) {
	store := newMockStore()
	handler := NewFetchMetadataHandler(store)

	payload, _ := json.Marshal(FetchMetadataPayload{BookID: "nonexistent"})
	err := handler(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error for unknown book")
	}
}

func TestFetchMetadataHandler_NoFetchers(t *testing.T) {
	book := &db.Book{ID: "book-1", Title: "My Book"}
	store := newMockStore(book)
	handler := NewFetchMetadataHandler(store)

	payload, _ := json.Marshal(FetchMetadataPayload{BookID: "book-1"})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	// No fetchers → book is updated with its original values unchanged.
	if store.updated == nil {
		t.Fatal("expected UpdateBook to be called")
	}
	if store.updated.Title != "My Book" {
		t.Errorf("expected title %q, got %q", "My Book", store.updated.Title)
	}
}

func TestFetchMetadataHandler_MergesMetadata(t *testing.T) {
	book := &db.Book{ID: "book-1", Title: "My Book"}
	store := newMockStore(book)

	// A fetcher that returns a description and page count.
	staticFetcher := &staticMetadataFetcher{
		meta: &BookMetadata{
			Description: strPtr("A great story"),
			NumPages:    intPtr(300),
		},
	}

	handler := NewFetchMetadataHandler(store, staticFetcher)

	payload, _ := json.Marshal(FetchMetadataPayload{BookID: "book-1"})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if store.updated == nil {
		t.Fatal("expected UpdateBook to be called")
	}
	if store.updated.Description == nil || *store.updated.Description != "A great story" {
		t.Errorf("expected description %q, got %v", "A great story", store.updated.Description)
	}
	if store.updated.NumPages == nil || *store.updated.NumPages != 300 {
		t.Errorf("expected num_pages 300, got %v", store.updated.NumPages)
	}
}

func TestFetchMetadataHandler_PreservesExistingData(t *testing.T) {
	existingDesc := "Existing description"
	book := &db.Book{ID: "book-1", Title: "My Book", Description: &existingDesc}
	store := newMockStore(book)

	// Fetcher returns nil metadata (nothing to update).
	nopFetcher := &staticMetadataFetcher{meta: nil}

	handler := NewFetchMetadataHandler(store, nopFetcher)

	payload, _ := json.Marshal(FetchMetadataPayload{BookID: "book-1"})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler: %v", err)
	}

	if store.updated.Description == nil || *store.updated.Description != existingDesc {
		t.Errorf("expected description to be preserved as %q, got %v", existingDesc, store.updated.Description)
	}
}

func TestFetchMetadataHandler_FetcherError(t *testing.T) {
	book := &db.Book{ID: "book-1", Title: "My Book"}
	store := newMockStore(book)

	// An erroring fetcher should not abort the job.
	errFetcher := &errorMetadataFetcher{}
	handler := NewFetchMetadataHandler(store, errFetcher)

	payload, _ := json.Marshal(FetchMetadataPayload{BookID: "book-1"})
	if err := handler(context.Background(), payload); err != nil {
		t.Fatalf("handler should not fail on a fetcher error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GoodreadsClient tests
// ---------------------------------------------------------------------------

func TestGoodreadsClient_NotConfigured(t *testing.T) {
	c := &GoodreadsClient{APIKey: "", HTTPClient: http.DefaultClient}
	book := &db.Book{ID: "b1", Title: "Some Book"}
	meta, err := c.Fetch(context.Background(), book)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil metadata when not configured, got %+v", meta)
	}
}

func TestGoodreadsClient_ParsesResponse(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<GoodreadsResponse>
  <search>
    <results>
      <work>
        <best_book>
          <id>12345</id>
          <title>My Great Book</title>
          <image_url>https://example.com/cover.jpg</image_url>
          <author><name>Jane Doe</name></author>
        </best_book>
        <original_publication_year>2020</original_publication_year>
      </work>
    </results>
  </search>
</GoodreadsResponse>`

	// Parse the XML directly via the unexported helper.
	meta, err := parseGoodreadsBody([]byte(xmlBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}
	if meta.GoodreadsID == nil || *meta.GoodreadsID != "12345" {
		t.Errorf("expected goodreads_id %q, got %v", "12345", meta.GoodreadsID)
	}
	if meta.Title == nil || *meta.Title != "My Great Book" {
		t.Errorf("expected title %q, got %v", "My Great Book", meta.Title)
	}
	if meta.CoverImageURL == nil || *meta.CoverImageURL != "https://example.com/cover.jpg" {
		t.Errorf("expected cover_image_url, got %v", meta.CoverImageURL)
	}
	if meta.PublicationDate == nil || *meta.PublicationDate != "2020" {
		t.Errorf("expected publication_date %q, got %v", "2020", meta.PublicationDate)
	}
}

func TestGoodreadsClient_EmptyResults(t *testing.T) {
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<GoodreadsResponse>
  <search><results></results></search>
</GoodreadsResponse>`

	meta, err := parseGoodreadsBody([]byte(xmlBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil metadata for empty results, got %+v", meta)
	}
}

// ---------------------------------------------------------------------------
// HardcoverClient tests
// ---------------------------------------------------------------------------

func TestHardcoverClient_NotConfigured(t *testing.T) {
	c := &HardcoverClient{APIToken: "", HTTPClient: http.DefaultClient}
	book := &db.Book{ID: "b1", Title: "Some Book"}
	meta, err := c.Fetch(context.Background(), book)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil metadata when not configured, got %+v", meta)
	}
}

func TestHardcoverClient_ParsesResponse(t *testing.T) {
	books := []hardcoverBook{
		{
			ID:          42,
			Title:       "Hardcover Book",
			Description: "A fine read",
			Pages:       250,
			ReleaseDate: "2021-06-01",
			Language: struct {
				Language string `json:"language"`
			}{Language: "English"},
			Image: struct {
				URL string `json:"url"`
			}{URL: "https://example.com/hc.jpg"},
		},
	}
	resultsJSON, _ := json.Marshal(books)
	resp := hardcoverSearchResponse{}
	resp.Data.Search.Results = resultsJSON

	respBytes, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(respBytes)
	}))
	defer srv.Close()

	c := &HardcoverClient{APIToken: "token", HTTPClient: srv.Client()}
	c.baseURL = srv.URL

	book := &db.Book{ID: "b1", Title: "Hardcover Book"}
	meta, err := c.Fetch(context.Background(), book)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}
	if meta.HardcoverID == nil || *meta.HardcoverID != "42" {
		t.Errorf("expected hardcover_id %q, got %v", "42", meta.HardcoverID)
	}
	if meta.Title == nil || *meta.Title != "Hardcover Book" {
		t.Errorf("expected title %q, got %v", "Hardcover Book", meta.Title)
	}
	if meta.Description == nil || *meta.Description != "A fine read" {
		t.Errorf("expected description, got %v", meta.Description)
	}
	if meta.NumPages == nil || *meta.NumPages != 250 {
		t.Errorf("expected num_pages 250, got %v", meta.NumPages)
	}
	if meta.Language == nil || *meta.Language != "English" {
		t.Errorf("expected language %q, got %v", "English", meta.Language)
	}
	if meta.CoverImageURL == nil || *meta.CoverImageURL != "https://example.com/hc.jpg" {
		t.Errorf("expected cover_image_url, got %v", meta.CoverImageURL)
	}
}

func TestHardcoverClient_EmptyResults(t *testing.T) {
	resp := hardcoverSearchResponse{}
	resp.Data.Search.Results = json.RawMessage(`[]`)
	respBytes, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(respBytes)
	}))
	defer srv.Close()

	c := &HardcoverClient{APIToken: "token", HTTPClient: srv.Client()}
	c.baseURL = srv.URL

	book := &db.Book{ID: "b1", Title: "No Match"}
	meta, err := c.Fetch(context.Background(), book)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil metadata for empty results, got %+v", meta)
	}
}

// ---------------------------------------------------------------------------
// buildSearchQuery tests
// ---------------------------------------------------------------------------

func TestBuildSearchQuery(t *testing.T) {
	isbn13 := "9780000000000"
	isbn10 := "0000000000"

	tests := []struct {
		name string
		book *db.Book
		want string
	}{
		{"title only", &db.Book{Title: "My Book"}, "My Book"},
		{"isbn13 preferred", &db.Book{Title: "My Book", ISBN13: &isbn13}, isbn13},
		{"isbn10 used when no isbn13", &db.Book{Title: "My Book", ISBN10: &isbn10}, isbn10},
		{"empty title", &db.Book{Title: "  "}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSearchQuery(tc.book)
			if got != tc.want {
				t.Errorf("buildSearchQuery() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Stub fetchers used by tests
// ---------------------------------------------------------------------------

type staticMetadataFetcher struct{ meta *BookMetadata }

func (f *staticMetadataFetcher) Name() string { return "static" }
func (f *staticMetadataFetcher) Fetch(_ context.Context, _ *db.Book) (*BookMetadata, error) {
	return f.meta, nil
}

type errorMetadataFetcher struct{}

func (f *errorMetadataFetcher) Name() string { return "error" }
func (f *errorMetadataFetcher) Fetch(_ context.Context, _ *db.Book) (*BookMetadata, error) {
	return nil, fmt.Errorf("simulated fetch error")
}
