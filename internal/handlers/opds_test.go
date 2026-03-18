package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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
		failNowf(t, "unmarshal feed: %v\nbody: %s", err, body)
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
			failf(t, "%s: status = %d, want %d", method, w.Code, http.StatusMethodNotAllowed)
		}
	}
}

func TestHandleOPDS_UnknownPath(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Root feed ---

func TestRootFeed(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		failf(t, "content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Biblioteka OPDS Catalog" {
		failf(t, "title = %q, want %q", feed.Title, "Biblioteka OPDS Catalog")
	}

	// Root feed has 4 navigation entries.
	if len(feed.Entries) != 4 {
		failNowf(t, "entries = %d, want 4", len(feed.Entries))
	}
	titles := []string{"All Books", "Recent Books", "Authors", "Series"}
	for i, want := range titles {
		if feed.Entries[i].Title != want {
			failf(t, "entry[%d].title = %q, want %q", i, feed.Entries[i].Title, want)
		}
	}

	// Must have self, start, and search links.
	if l := findLink(feed.Links, relSelf); l == nil {
		fail(t, "missing self link")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		fail(t, "missing start link")
	}
	if l := findLink(feed.Links, relSearch); l == nil {
		fail(t, "missing search link")
	}
}

func TestRootFeed_TrailingSlash(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRootFeed_HEAD(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodHead, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- All books feed ---

func TestAllBooks_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsAcqContentType {
		failf(t, "content-type = %q, want %q", ct, opdsAcqContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "All Books" {
		failf(t, "title = %q, want %q", feed.Title, "All Books")
	}
	if len(feed.Entries) != 0 {
		failf(t, "entries = %d, want 0", len(feed.Entries))
	}
}

func TestAllBooks_WithBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Beta", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		failf(t, "entries = %d, want 2", len(feed.Entries))
	}
	// Books should be sorted by title.
	if feed.Entries[0].Title != "Alpha" {
		failf(t, "entries[0].title = %q, want %q", feed.Entries[0].Title, "Alpha")
	}
}

func TestAllBooks_WithDescription(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	desc := "A great book"
	h.DB.CreateBook(ctx, "Alpha", &desc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		failNowf(t, "entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Content == nil {
		failNow(t, "expected content, got nil")
	}
	if feed.Entries[0].Content.Value != "A great book" {
		failf(t, "content = %q, want %q", feed.Entries[0].Content.Value, "A great book")
	}
}

func TestAllBooks_WithAuthorsAndFiles(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, _ := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	author, _ := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID})
	h.DB.CreateBookFile(ctx, book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		failNowf(t, "entries = %d, want 1", len(feed.Entries))
	}

	entry := feed.Entries[0]
	if len(entry.Authors) != 1 || entry.Authors[0].Name != "Stephen King" {
		failf(t, "authors = %v, want [Stephen King]", entry.Authors)
	}

	acqLink := findLink(entry.Links, relAcquisition)
	if acqLink == nil {
		failNow(t, "missing acquisition link")
	}
	if acqLink.Type != "application/epub+zip" {
		failf(t, "acquisition type = %q, want %q", acqLink.Type, "application/epub+zip")
	}
}

// --- Recent books feed ---

func TestRecentBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "First", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Second", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Recent Books" {
		failf(t, "title = %q, want %q", feed.Title, "Recent Books")
	}
	if len(feed.Entries) != 2 {
		failf(t, "entries = %d, want 2", len(feed.Entries))
	}
}

// --- Authors feed ---

func TestAuthorsFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		failf(t, "content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		failf(t, "entries = %d, want 0", len(feed.Entries))
	}
	if l := findLink(feed.Links, relStart); l == nil {
		fail(t, "missing start link")
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
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		failf(t, "entries = %d, want 2", len(feed.Entries))
	}

	// Each entry should have a subsection link.
	for i, e := range feed.Entries {
		if l := findLink(e.Links, relSubsection); l == nil {
			failf(t, "entry[%d]: missing subsection link", i)
		}
	}
}

// --- Author books feed ---

func TestAuthorBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	author, _ := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	book, _ := h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID})

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Books by Stephen King" {
		failf(t, "title = %q, want %q", feed.Title, "Books by Stephen King")
	}
	if len(feed.Entries) != 1 {
		failNowf(t, "entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "The Shining" {
		failf(t, "entry title = %q, want %q", feed.Entries[0].Title, "The Shining")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		fail(t, "missing start link")
	}
}

func TestAuthorBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Series feed ---

func TestSeriesFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsNavContentType {
		failf(t, "content-type = %q, want %q", ct, opdsNavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		failf(t, "entries = %d, want 0", len(feed.Entries))
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
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		failf(t, "entries = %d, want 2", len(feed.Entries))
	}
}

// --- Series books feed ---

func TestSeriesBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	series, _ := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)
	book, _ := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	pos := 1.0
	h.DB.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}})

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "The Dark Tower" {
		failf(t, "title = %q, want %q", feed.Title, "The Dark Tower")
	}
	if len(feed.Entries) != 1 {
		failNowf(t, "entries = %d, want 1", len(feed.Entries))
	}
	if feed.Entries[0].Title != "The Gunslinger" {
		failf(t, "entry title = %q, want %q", feed.Entries[0].Title, "The Gunslinger")
	}
	if l := findLink(feed.Links, relStart); l == nil {
		fail(t, "missing start link")
	}
}

func TestSeriesBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Search ---

func TestSearch_WithResults(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=The", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		failf(t, "entries = %d, want 2", len(feed.Entries))
	}
	if !strings.Contains(feed.Title, "The") {
		failf(t, "title = %q, should contain search query", feed.Title)
	}
}

func TestSearch_NoResults(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		failf(t, "entries = %d, want 0", len(feed.Entries))
	}
}

func TestSearch_SpecialCharsInQuery(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	h.DB.CreateBook(ctx, "100% Pure", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.DB.CreateBook(ctx, "Other Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Search for "%" should not match everything due to LIKE wildcard escaping.
	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=%25", nil) // %25 = URL-encoded "%"
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		failf(t, "entries = %d, want 1 (only '100%% Pure' should match)", len(feed.Entries))
	}
}

func TestSearch_URLEncodesQueryInLinks(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=foo+bar", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		failNow(t, "missing self link")
	}
	// The query should be URL-encoded in the self link.
	if strings.Contains(selfLink.Href, "q=foo bar") {
		failf(t, "self link has unencoded query: %q", selfLink.Href)
	}
}

// --- OpenSearch description ---

func TestOpenSearchDescription(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdsSearchType {
		failf(t, "content-type = %q, want %q", ct, opdsSearchType)
	}
	body := w.Body.String()
	if !strings.Contains(body, "OpenSearchDescription") {
		fail(t, "response should contain OpenSearchDescription element")
	}
	if !strings.Contains(body, "{searchTerms}") {
		fail(t, "response should contain {searchTerms} template")
	}
}

// --- Download ---

func TestDownload_Success(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, _ := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// Create a temp file to serve.
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.epub")
	if err := os.WriteFile(filePath, []byte("fake epub content"), 0644); err != nil {
		failNowf(t, "write temp file: %v", err)
	}

	bf, _ := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 17, nil, filePath)

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/epub+zip" {
		failf(t, "content-type = %q, want %q", ct, "application/epub+zip")
	}
	if disp := w.Header().Get("Content-Disposition"); !strings.Contains(disp, "test.epub") {
		failf(t, "content-disposition = %q, should contain filename", disp)
	}
	if w.Body.String() != "fake epub content" {
		failf(t, "body = %q, want %q", w.Body.String(), "fake epub content")
	}
}

func TestDownload_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/download/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload_FileMissing(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, _ := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 100, nil, "/nonexistent/path.epub")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload_UnknownFileType(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	book, _ := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.xyz")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		failNowf(t, "write temp file: %v", err)
	}

	bf, _ := h.DB.CreateBookFile(ctx, book.ID, "xyz", "test.xyz", 4, nil, filePath)

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/octet-stream" {
		failf(t, "content-type = %q, want %q", ct, "application/octet-stream")
	}
}

// --- Pagination ---

func TestAllBooks_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	// Create enough books to have a second page (opdsPageSize is 50).
	for i := range 55 {
		h.DB.CreateBook(ctx, "Book "+padInt(i), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	}

	// Page 1: should have "next" link but no "previous" link.
	r := httptest.NewRequest(http.MethodGet, "/opds/all?page=1", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "page 1: status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 50 {
		failf(t, "page 1: entries = %d, want 50", len(feed.Entries))
	}
	if findLink(feed.Links, relNext) == nil {
		fail(t, "page 1: missing next link")
	}
	if findLink(feed.Links, relPrevious) != nil {
		fail(t, "page 1: should not have previous link")
	}

	// Page 2: should have "previous" link but no "next" link.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/all?page=2", nil)
	w2 := httptest.NewRecorder()
	h.HandleOPDS(w2, r2)

	if w2.Code != http.StatusOK {
		failNowf(t, "page 2: status = %d, want %d", w2.Code, http.StatusOK)
	}

	feed2 := parseOPDSFeed(t, w2.Body.Bytes())
	if len(feed2.Entries) != 5 {
		failf(t, "page 2: entries = %d, want 5", len(feed2.Entries))
	}
	if findLink(feed2.Links, relPrevious) == nil {
		fail(t, "page 2: missing previous link")
	}
	if findLink(feed2.Links, relNext) != nil {
		fail(t, "page 2: should not have next link")
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
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		failNow(t, "missing self link")
	}
	if !strings.HasPrefix(selfLink.Href, "https://") {
		failf(t, "self link = %q, want https:// prefix", selfLink.Href)
	}
}

func TestBaseURL_InvalidXForwardedProto(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.Header.Set("X-Forwarded-Proto", "javascript:")
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, relSelf)
	if selfLink == nil {
		failNow(t, "missing self link")
	}
	// Should fallback to http, not use the injected value.
	if strings.HasPrefix(selfLink.Href, "javascript:") {
		failf(t, "self link = %q, should not use injected proto", selfLink.Href)
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
		{"https://example.com/cover", "image/jpeg"}, // no extension defaults to jpeg
	}

	for _, tt := range tests {
		got := coverMIMEType(tt.url)
		if got != tt.want {
			failf(t, "coverMIMEType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestCoverImageInFeed(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := context.Background()

	coverURL := "https://example.com/cover.png"
	h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		failNowf(t, "entries = %d, want 1", len(feed.Entries))
	}
	imgLink := findLink(feed.Entries[0].Links, relImage)
	if imgLink == nil {
		failNow(t, "missing image link")
	}
	if imgLink.Type != "image/png" {
		failf(t, "image type = %q, want %q", imgLink.Type, "image/png")
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
			failf(t, "parsePage(%q) = %d, want %d", tt.query, got, tt.want)
		}
	}
}

func TestPaginationLinks(t *testing.T) {
	// Single page: no next or previous.
	links := paginationLinks("/opds/all", 1, 10, 50, opdsAcqContentType)
	if findLink(links, relNext) != nil {
		fail(t, "single page: should not have next link")
	}
	if findLink(links, relPrevious) != nil {
		fail(t, "single page: should not have previous link")
	}

	// First of multiple pages: next but no previous.
	links = paginationLinks("/opds/all", 1, 100, 50, opdsAcqContentType)
	if findLink(links, relNext) == nil {
		fail(t, "first page: should have next link")
	}
	if findLink(links, relPrevious) != nil {
		fail(t, "first page: should not have previous link")
	}

	// Middle page: both next and previous.
	links = paginationLinks("/opds/all", 2, 150, 50, opdsAcqContentType)
	if findLink(links, relNext) == nil {
		fail(t, "middle page: should have next link")
	}
	if findLink(links, relPrevious) == nil {
		fail(t, "middle page: should have previous link")
	}

	// Last page: previous but no next.
	links = paginationLinks("/opds/all", 2, 100, 50, opdsAcqContentType)
	if findLink(links, relNext) != nil {
		fail(t, "last page: should not have next link")
	}
	if findLink(links, relPrevious) == nil {
		fail(t, "last page: should have previous link")
	}
}

func TestPaginationLinks_SearchURL(t *testing.T) {
	// URLs with existing query params should use "&" not "?" for page param.
	links := paginationLinks("/opds/search?q=test", 1, 100, 50, opdsAcqContentType)
	selfLink := findLink(links, relSelf)
	if selfLink == nil {
		failNow(t, "missing self link")
	}
	if strings.Contains(selfLink.Href, "?q=test?page=") {
		failf(t, "self link has double '?': %q", selfLink.Href)
	}
	if !strings.Contains(selfLink.Href, "&page=") {
		failf(t, "self link should use '&' for page param: %q", selfLink.Href)
	}
}

// padInt zero-pads an integer to 3 digits for consistent sorting.
func padInt(n int) string {
	return fmt.Sprintf("%03d", n)
}
