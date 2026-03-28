package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
)

func setupOPDSHandler(t *testing.T) *OPDSHandler {
	t.Helper()
	d := newTestDB(t)
	return &OPDSHandler{DB: d}
}

// parseOPDSFeed parses the response body as an OPDS Atom feed.
func parseOPDSFeed(t *testing.T, body []byte) opdsFeed {
	t.Helper()
	var feed opdsFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("unmarshal feed: %v\nbody: %s", err, body)
	}
	return feed
}

// findLink returns the first link with the given rel, or nil if not found.
func findLink(links []opdsLink, rel string) *opdsLink {
	for _, l := range links {
		if l.Rel == rel {
			return &l
		}
	}
	return nil
}

// --- Routing / method dispatch ---

func TestHandleOPDS_MethodNotAllowed(t *testing.T) {
	h := setupOPDSHandler(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		r := httptest.NewRequest(method, "/opds", nil)
		w := httptest.NewRecorder()
		h.HandleOPDS(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestHandleOPDS_UnknownPath(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Root feed ---

func TestRootFeed(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Biblioteka OPDS Catalog" {
		t.Errorf("title = %q, want %q", feed.Title, "Biblioteka OPDS Catalog")
	}

	// Root feed has 4 navigation entries.
	if len(feed.Entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(feed.Entries))
	}
	titles := []string{"All Books", "Recent Books", "Authors", "Series"}
	for i, want := range titles {
		if feed.Entries[i].Title != want {
			t.Errorf("entry[%d].title = %q, want %q", i, feed.Entries[i].Title, want)
		}
	}

	// Must have self, start, and search links.
	if l := findLink(feed.Links, relSelf); l == nil {
		t.Error("missing self link")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		t.Error("missing start link")
	}
	if l := findLink(feed.Links, relSearch); l == nil {
		t.Error("missing search link")
	}
}

func TestRootFeed_TrailingSlash(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRootFeed_HEAD(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodHead, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- All books feed ---

func TestAllBooks_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsAcqContentType {
		t.Errorf("content-type = %q, want %q", ct, opdsAcqContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "All Books" {
		t.Errorf("title = %q, want %q", feed.Title, "All Books")
	}
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
}

func TestAllBooks_WithBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Beta", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}
	// Books should be sorted by title.
	if feed.Entries[0].Title != "Alpha" {
		t.Errorf("entries[0].title = %q, want %q", feed.Entries[0].Title, "Alpha")
	}
}

func TestAllBooks_WithDescription(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	desc := "A great book"
	h.DB.CreateBook(ctx, "Alpha", &desc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Content == nil {
		t.Fatal("expected content, got nil")
	}
	if feed.Entries[0].Content.Value != "A great book" {
		t.Errorf("content = %q, want %q", feed.Entries[0].Content.Value, "A great book")
	}
}

func TestAllBooks_WithAuthorsAndFiles(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, err := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	author, err := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}
	h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID})
	_, err = h.DB.CreateBookFile(ctx, book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}

	entry := feed.Entries[0]
	if len(entry.Authors) != 1 || entry.Authors[0].Name != "Stephen King" {
		t.Errorf("authors = %v, want [Stephen King]", entry.Authors)
	}

	acqLink := findLink(entry.Links, relAcquisition)
	if acqLink == nil {
		t.Fatal("missing acquisition link")
	}
	if acqLink.Type != "application/epub+zip" {
		t.Errorf("acquisition type = %q, want %q", acqLink.Type, "application/epub+zip")
	}
}

// --- Recent books feed ---

func TestRecentBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "First", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Second", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Recent Books" {
		t.Errorf("title = %q, want %q", feed.Title, "Recent Books")
	}
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}
}

// --- Authors feed ---

func TestAuthorsFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
	if l := findLink(feed.Links, relStart); l == nil {
		t.Error("missing start link")
	}
}

func TestAuthorsFeed_WithAuthors(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateAuthor(ctx, "Brandon Sanderson", nil, nil, nil, nil)
	h.DB.CreateAuthor(ctx, "Anne McCaffrey", nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}

	// Each entry should have a subsection link.
	for i, e := range feed.Entries {
		if l := findLink(e.Links, relSubsection); l == nil {
			t.Errorf("entry[%d]: missing subsection link", i)
		}
	}
}

// --- Author books feed ---

func TestAuthorBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	author, err := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create author: %v", err)
	}
	book, err := h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID})

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Books by Stephen King" {
		t.Errorf("title = %q, want %q", feed.Title, "Books by Stephen King")
	}
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "The Shining" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "The Shining")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		t.Error("missing start link")
	}
}

func TestAuthorBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Series feed ---

func TestSeriesFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
}

func TestSeriesFeed_WithSeries(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)
	h.DB.CreateSeries(ctx, "Discworld", nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}
}

// --- Series books feed ---

func TestSeriesBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	series, err := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	book, err := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	pos := 1.0
	h.DB.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}})

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "The Dark Tower" {
		t.Errorf("title = %q, want %q", feed.Title, "The Dark Tower")
	}
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "The Gunslinger" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "The Gunslinger")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		t.Error("missing start link")
	}
}

func TestSeriesBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Search ---

func TestSearch_WithResults(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=The", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}
	if !strings.Contains(feed.Title, "The") {
		t.Errorf("title = %q, should contain search query", feed.Title)
	}
}

func TestSearch_NoResults(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
}

func TestSearch_SpecialCharsInQuery(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "100% Pure", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Other Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Search for "%" should not match everything due to LIKE wildcard escaping.
	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=%25", nil) // %25 = URL-encoded "%"
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Errorf("entries = %d, want 1 (only '100%% Pure' should match)", len(feed.Entries))
	}
}

func TestSearch_URLEncodesQueryInLinks(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=foo+bar", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	// The query should be URL-encoded in the self link.
	if strings.Contains(selfLink.Href, "q=foo bar") {
		t.Errorf("self link has unencoded query: %q", selfLink.Href)
	}
}

// --- OpenSearch description ---

func TestOpenSearchDescription(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsSearchType {
		t.Errorf("content-type = %q, want %q", ct, opdsSearchType)
	}
	body := w.Body.String()
	if !strings.Contains(body, "OpenSearchDescription") {
		t.Error("response should contain OpenSearchDescription element")
	}
	if !strings.Contains(body, "{searchTerms}") {
		t.Error("response should contain {searchTerms} template")
	}
}

// --- Download ---

func TestDownload_Success(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	// Create a temp file to serve.
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.epub")
	if err := os.WriteFile(filePath, []byte("fake epub content"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 17, nil, filePath)
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/epub+zip" {
		t.Errorf("content-type = %q, want %q", ct, "application/epub+zip")
	}
	if disp := w.Header().Get("Content-Disposition"); !strings.Contains(disp, "test.epub") {
		t.Errorf("content-disposition = %q, should contain filename", disp)
	}
	if w.Body.String() != "fake epub content" {
		t.Errorf("body = %q, want %q", w.Body.String(), "fake epub content")
	}
}

func TestDownload_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/download/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload_FileMissing(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 100, nil, "/nonexistent/path.epub")
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload_UnknownFileType(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.xyz")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "xyz", "test.xyz", 4, nil, filePath)
	if err != nil {
		t.Fatalf("create book file: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("content-type = %q, want %q", ct, "application/octet-stream")
	}
}

// --- Pagination ---

func TestAllBooks_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	// Create enough books to have a second page (opdsPageSize is 50).
	for i := range 55 {
		h.DB.CreateBook(ctx, "Book "+padInt(i), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}

	// Page 1: should have "next" link but no "previous" link.
	r := httptest.NewRequest(http.MethodGet, "/opds/all?page=1", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 50 {
		t.Errorf("page 1: entries = %d, want 50", len(feed.Entries))
	}
	if findLink(feed.Links, relNext) == nil {
		t.Error("page 1: missing next link")
	}
	if findLink(feed.Links, relPrevious) != nil {
		t.Error("page 1: should not have previous link")
	}

	// Page 2: should have "previous" link but no "next" link.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/all?page=2", nil)
	w2 := httptest.NewRecorder()
	h.HandleOPDS(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: status = %d, want %d", w2.Code, http.StatusOK)
	}

	feed2 := parseOPDSFeed(t, w2.Body.Bytes())
	if len(feed2.Entries) != 5 {
		t.Errorf("page 2: entries = %d, want 5", len(feed2.Entries))
	}
	if findLink(feed2.Links, relPrevious) == nil {
		t.Error("page 2: missing previous link")
	}
	if findLink(feed2.Links, relNext) != nil {
		t.Error("page 2: should not have next link")
	}
}

// --- X-Forwarded-Proto ---

func TestBaseURL_XForwardedProto(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	if !strings.HasPrefix(selfLink.Href, "https://") {
		t.Errorf("self link = %q, want https:// prefix", selfLink.Href)
	}
}

func TestBaseURL_InvalidXForwardedProto(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.Header.Set("X-Forwarded-Proto", "javascript:")
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	// Should fallback to http, not use the injected value.
	if strings.HasPrefix(selfLink.Href, "javascript:") {
		t.Errorf("self link = %q, should not use injected proto", selfLink.Href)
	}
}

// --- Cover image MIME type ---

func TestCoverMIMEType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/cover.jpg", "image/jpeg"},
		{"https://example.com/cover.jpeg", "image/jpeg"},
		{"https://example.com/cover.png", "image/png"},
		{"https://example.com/cover.PNG", "image/png"},
		{"https://example.com/cover.webp", "image/webp"},
		{"https://example.com/cover.gif", "image/gif"},
		{"https://example.com/cover.svg", "image/svg+xml"},
		{"https://example.com/cover.avif", "image/avif"},
		{"data:image/png;base64,AAAA", "image/png"},
		{"https://example.com/cover", "image/jpeg"}, // no extension defaults to jpeg
	}

	for _, tt := range tests {
		got := coverMIMEType(tt.url)
		if got != tt.want {
			t.Errorf("coverMIMEType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestCoverImageInFeed(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	coverURL := "https://example.com/cover.png"
	h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	imgLink := findLink(feed.Entries[0].Links, relImage)
	if imgLink == nil {
		t.Fatal("missing image link")
	}
	if imgLink.Type != "image/png" {
		t.Errorf("image type = %q, want %q", imgLink.Type, "image/png")
	}
}

func TestCoverImageInFeed_DataURLRewritten(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	imgLink := findLink(feed.Entries[0].Links, relImage)
	if imgLink == nil {
		t.Fatal("missing image link")
	}
	wantHref := "http://example.com/opds/covers/" + book.ID
	if imgLink.Href != wantHref {
		t.Errorf("image href = %q, want %q", imgLink.Href, wantHref)
	}
	if strings.HasPrefix(imgLink.Href, "data:") {
		t.Error("image href should not be a data URL")
	}
}

func TestServeCover_DataURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		t.Errorf("content-type = %q, want image/*", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), pngBytes) {
		t.Errorf("body length = %d, want %d", w.Body.Len(), len(pngBytes))
	}
}

// --- Helper unit tests ---

func TestParsePage(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{"", 1},
		{"?page=1", 1},
		{"?page=3", 3},
		{"?page=0", 1},
		{"?page=-1", 1},
		{"?page=abc", 1},
	}

	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodGet, "/opds/all"+tt.query, nil)
		got := parsePage(r)
		if got != tt.want {
			t.Errorf("parsePage(%q) = %d, want %d", tt.query, got, tt.want)
		}
	}
}

func TestPaginationLinks(t *testing.T) {
	// Single page: no next or previous.
	links := paginationLinks("/opds/all", 1, 10, 50, opdsAcqContentType)
	if findLink(links, relNext) != nil {
		t.Error("single page: should not have next link")
	}
	if findLink(links, relPrevious) != nil {
		t.Error("single page: should not have previous link")
	}

	// First of multiple pages: next but no previous.
	links = paginationLinks("/opds/all", 1, 100, 50, opdsAcqContentType)
	if findLink(links, relNext) == nil {
		t.Error("first page: should have next link")
	}
	if findLink(links, relPrevious) != nil {
		t.Error("first page: should not have previous link")
	}

	// Middle page: both next and previous.
	links = paginationLinks("/opds/all", 2, 150, 50, opdsAcqContentType)
	if findLink(links, relNext) == nil {
		t.Error("middle page: should have next link")
	}
	if findLink(links, relPrevious) == nil {
		t.Error("middle page: should have previous link")
	}

	// Last page: previous but no next.
	links = paginationLinks("/opds/all", 2, 100, 50, opdsAcqContentType)
	if findLink(links, relNext) != nil {
		t.Error("last page: should not have next link")
	}
	if findLink(links, relPrevious) == nil {
		t.Error("last page: should have previous link")
	}
}

func TestPaginationLinks_SearchURL(t *testing.T) {
	// URLs with existing query params should use "&" not "?" for page param.
	links := paginationLinks("/opds/search?q=test", 1, 100, 50, opdsAcqContentType)
	selfLink := findLink(links, relSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	if strings.Contains(selfLink.Href, "?q=test?page=") {
		t.Errorf("self link has double '?': %q", selfLink.Href)
	}
	if !strings.Contains(selfLink.Href, "&page=") {
		t.Errorf("self link should use '&' for page param: %q", selfLink.Href)
	}
}

// padInt zero-pads an integer to 3 digits for consistent sorting.
func padInt(n int) string {
	return fmt.Sprintf("%03d", n)
}
