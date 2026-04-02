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
	if err := h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID}); err != nil {
		t.Fatalf("set book authors: %v", err)
	}
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
	if err := h.DB.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}}); err != nil {
		t.Fatalf("set book series: %v", err)
	}

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

// --- XML marshaling (opds_types.go) ---

func TestOPDSFeed_XMLMarshal(t *testing.T) {
	feed := &opdsFeed{
		XMLNS:     xmlnsAtom,
		XMLNSOPDS: xmlnsOPDS,
		ID:        "urn:test",
		Title:     "Test Feed",
		Updated:   "2024-01-01T00:00:00Z",
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// Element name must be "feed"
	if !strings.Contains(s, "<feed ") && !strings.Contains(s, "<feed>") {
		t.Errorf("XML does not contain <feed> element: %s", s)
	}
	// Atom xmlns must be present as attribute
	if !strings.Contains(s, xmlnsAtom) {
		t.Errorf("XML missing Atom xmlns attribute: %s", s)
	}
	// OPDS xmlns must be present
	if !strings.Contains(s, xmlnsOPDS) {
		t.Errorf("XML missing OPDS xmlns attribute: %s", s)
	}
	// ID and Title must be child elements
	if !strings.Contains(s, "<id>urn:test</id>") {
		t.Errorf("XML missing <id> element: %s", s)
	}
	if !strings.Contains(s, "<title>Test Feed</title>") {
		t.Errorf("XML missing <title> element: %s", s)
	}
}

func TestOPDSFeed_XMLMarshal_OmitEmptyOPDSNS(t *testing.T) {
	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      "urn:test",
		Title:   "Nav Feed",
		Updated: "2024-01-01T00:00:00Z",
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// When XMLNSOPDS is empty it should be omitted (omitempty)
	if strings.Contains(s, "xmlns:opds") {
		t.Errorf("xmlns:opds attribute should be absent when empty, got: %s", s)
	}
}

func TestOPDSLink_XMLMarshal(t *testing.T) {
	link := opdsLink{
		Rel:  relSelf,
		Href: "http://example.com/opds",
		Type: opdsNavContentType,
	}

	data, err := xml.Marshal(link)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `rel="self"`) {
		t.Errorf("XML missing rel attribute: %s", s)
	}
	if !strings.Contains(s, `href="http://example.com/opds"`) {
		t.Errorf("XML missing href attribute: %s", s)
	}
	if !strings.Contains(s, `type=`) {
		t.Errorf("XML missing type attribute: %s", s)
	}
}

func TestOPDSContent_XMLMarshal(t *testing.T) {
	content := opdsContent{
		Type:  "text",
		Value: "Some description text",
	}

	data, err := xml.Marshal(content)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `type="text"`) {
		t.Errorf("XML missing type attribute: %s", s)
	}
	// Value must be encoded as character data (not a child element)
	if !strings.Contains(s, "Some description text") {
		t.Errorf("XML missing chardata value: %s", s)
	}
	if strings.Contains(s, "<Value>") {
		t.Errorf("Value should be chardata, not a child element: %s", s)
	}
}

func TestOPDSEntry_XMLMarshal_Full(t *testing.T) {
	entry := opdsEntry{
		Title:   "My Book",
		ID:      "urn:book:1",
		Updated: "2024-01-01T00:00:00Z",
		Content: &opdsContent{Type: "text", Value: "A description"},
		Authors: []opdsAuthor{{Name: "Jane Doe"}},
		Links: []opdsLink{
			{Rel: relAcquisition, Href: "http://example.com/dl/1", Type: "application/epub+zip"},
		},
	}

	data, err := xml.Marshal(entry)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, "<title>My Book</title>") {
		t.Errorf("XML missing title: %s", s)
	}
	if !strings.Contains(s, "<id>urn:book:1</id>") {
		t.Errorf("XML missing id: %s", s)
	}
	if !strings.Contains(s, "A description") {
		t.Errorf("XML missing content: %s", s)
	}
	if !strings.Contains(s, "<name>Jane Doe</name>") {
		t.Errorf("XML missing author name: %s", s)
	}
}

func TestOPDSEntry_XMLMarshal_NoContent(t *testing.T) {
	entry := opdsEntry{
		Title:   "No Desc",
		ID:      "urn:book:2",
		Updated: "2024-01-01T00:00:00Z",
		Links:   []opdsLink{{Rel: relSelf, Href: "/x", Type: opdsAcqContentType}},
	}

	data, err := xml.Marshal(entry)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// Content should be absent when nil (omitempty)
	if strings.Contains(s, "<content") {
		t.Errorf("XML should have no <content> when nil, got: %s", s)
	}
}

func TestOPDSFeed_XMLMarshal_LinksAndEntries(t *testing.T) {
	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      "urn:test:2",
		Title:   "Books",
		Updated: "2024-01-01T00:00:00Z",
		Links: []opdsLink{
			{Rel: relSelf, Href: "/opds/all?page=1", Type: opdsAcqContentType},
			{Rel: relNext, Href: "/opds/all?page=2", Type: opdsAcqContentType},
		},
		Entries: []opdsEntry{
			{Title: "Book One", ID: "urn:b:1", Updated: "2024-01-01T00:00:00Z"},
		},
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `rel="self"`) {
		t.Errorf("XML missing self link: %s", s)
	}
	if !strings.Contains(s, `rel="next"`) {
		t.Errorf("XML missing next link: %s", s)
	}
	if !strings.Contains(s, "<title>Book One</title>") {
		t.Errorf("XML missing entry: %s", s)
	}
}

// --- writeOPDSError (opds_helpers.go) ---

func TestWriteOPDSError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdsAcqContentType, "urn:biblioteka:opds:error", "Something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsAcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdsAcqContentType)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "<?xml") {
		t.Errorf("body should start with XML declaration, got: %s", body[:min(len(body), 50)])
	}

	// Must be a valid feed with the provided title
	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Something went wrong" {
		t.Errorf("title = %q, want %q", feed.Title, "Something went wrong")
	}
	if feed.ID != "urn:biblioteka:opds:error" {
		t.Errorf("id = %q, want %q", feed.ID, "urn:biblioteka:opds:error")
	}
}

func TestWriteOPDSError_NavContentType(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdsNavContentType, "urn:test", "Nav error")

	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdsNavContentType)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- writeOPDSFeed (opds_helpers.go) ---

func TestWriteOPDSFeed_Direct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()

	feed := &opdsFeed{
		XMLNS:   xmlnsAtom,
		ID:      "urn:test",
		Title:   "Direct Feed",
		Updated: "2024-01-01T00:00:00Z",
	}
	writeOPDSFeed(r, w, opdsAcqContentType, feed)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsAcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdsAcqContentType)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "<?xml") {
		t.Errorf("body should start with XML declaration, got: %s", body[:min(len(body), 50)])
	}
	parsed := parseOPDSFeed(t, w.Body.Bytes())
	if parsed.Title != "Direct Feed" {
		t.Errorf("title = %q, want %q", parsed.Title, "Direct Feed")
	}
}

// --- DB error paths (opds_feeds.go) ---

func TestAllBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsAcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdsAcqContentType)
	}
	// Response must still be valid XML (OPDS error feed)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestRecentBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestAuthorsFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSeriesFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSearch_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=test", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

// --- bookEntries error paths ---

func TestBookEntries_AuthorLoadError(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	// Close DB so batch author/file loads fail.
	h.DB.Close()

	books := []db.Book{*book}
	entries := h.bookEntries(ctx, books, "http://example.com/opds")

	// Should still return entries, just without authors or download links.
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Title != "Test Book" {
		t.Errorf("title = %q, want %q", entries[0].Title, "Test Book")
	}
	if len(entries[0].Authors) != 0 {
		t.Errorf("authors = %v, want empty (batch load failed)", entries[0].Authors)
	}
}

// --- serveCover missing paths ---

func TestServeCover_NilCover(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	// Book with no cover (nil CoverImageURL).
	book, err := h.DB.CreateBook(ctx, "No Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServeCover_EmptyCover(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	empty := ""
	book, err := h.DB.CreateBook(ctx, "Empty Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &empty)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServeCover_ExternalURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	coverURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(ctx, "External Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}
	location := w.Header().Get("Location")
	if location != coverURL {
		t.Errorf("Location = %q, want %q", location, coverURL)
	}
}

func TestServeCover_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	h.DB.Close()

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/someid", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Authors/Series pagination ---

func TestAuthorsFeed_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	for i := range 55 {
		if _, err := h.DB.CreateAuthor(ctx, fmt.Sprintf("Author %03d", i), nil, nil, nil, nil); err != nil {
			t.Fatalf("create author %d: %v", i, err)
		}
	}

	// Page 1: should have next link but no previous.
	r := httptest.NewRequest(http.MethodGet, "/opds/authors?page=1", nil)
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

	// Page 2: should have previous link but no next.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/authors?page=2", nil)
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

func TestSeriesFeed_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	for i := range 55 {
		if _, err := h.DB.CreateSeries(ctx, fmt.Sprintf("Series %03d", i), nil, nil, nil); err != nil {
			t.Fatalf("create series %d: %v", i, err)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series?page=1", nil)
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
}
