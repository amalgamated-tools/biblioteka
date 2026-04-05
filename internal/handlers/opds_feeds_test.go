package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"

	"github.com/stretchr/testify/require"
)

// --- Root feed ---

func TestRootFeed(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdspkg.NavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Biblioteka OPDS Catalog" {
		t.Errorf("title = %q, want %q", feed.Title, "Biblioteka OPDS Catalog")
	}

	// Root feed has 4 navigation entries.
	require.Len(t, feed.Entries, 4)
	titles := []string{"All Books", "Recent Books", "Authors", "Series"}
	for i, want := range titles {
		if feed.Entries[i].Title != want {
			t.Errorf("entry[%d].title = %q, want %q", i, feed.Entries[i].Title, want)
		}
	}

	// Must have self, start, and search links.
	if l := findLink(feed.Links, opdspkg.RelSelf); l == nil {
		t.Error("missing self link")
	}
	if l := findLink(feed.Links, opdspkg.RelStart); l == nil {
		t.Error("missing start link")
	}
	if l := findLink(feed.Links, opdspkg.RelSearch); l == nil {
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

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("content-type = %q, want %q", ct, opdspkg.AcqContentType)
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
	ctx := t.Context()

	if _, err := h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book Alpha")
	}
	if _, err := h.DB.CreateBook(ctx, "Beta", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book Beta")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

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
	ctx := t.Context()

	desc := "A great book"
	if _, err := h.DB.CreateBook(ctx, "Alpha", &desc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)
	require.NotNil(t, feed.Entries[0].Content)
	if feed.Entries[0].Content.Value != "A great book" {
		t.Errorf("content = %q, want %q", feed.Entries[0].Content.Value, "A great book")
	}
}

func TestAllBooks_WithAuthorsAndFiles(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	author, err := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	require.NoError(t, h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID}), "set book authors")
	_, err = h.DB.CreateBookFile(ctx, book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)

	entry := feed.Entries[0]
	if len(entry.Authors) != 1 || entry.Authors[0].Name != "Stephen King" {
		t.Errorf("authors = %v, want [Stephen King]", entry.Authors)
	}

	acqLink := findLink(entry.Links, opdspkg.RelAcquisition)
	require.NotNil(t, acqLink)
	if acqLink.Type != "application/epub+zip" {
		t.Errorf("acquisition type = %q, want %q", acqLink.Type, "application/epub+zip")
	}
}

// --- Recent books feed ---

func TestRecentBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	if _, err := h.DB.CreateBook(ctx, "First", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book First")
	}
	if _, err := h.DB.CreateBook(ctx, "Second", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book Second")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

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

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdspkg.NavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
	if l := findLink(feed.Links, opdspkg.RelStart); l == nil {
		t.Error("missing start link")
	}
}

func TestAuthorsFeed_WithAuthors(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	if _, err := h.DB.CreateAuthor(ctx, "Brandon Sanderson", nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create author")
	}
	if _, err := h.DB.CreateAuthor(ctx, "Anne McCaffrey", nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create author")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}

	// Each entry should have a subsection link.
	for i, e := range feed.Entries {
		if l := findLink(e.Links, opdspkg.RelSubsection); l == nil {
			t.Errorf("entry[%d]: missing subsection link", i)
		}
	}
}

// --- Author books feed ---

func TestAuthorBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	author, err := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	book, err := h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	require.NoError(t, h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID}), "set book authors")

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Books by Stephen King" {
		t.Errorf("title = %q, want %q", feed.Title, "Books by Stephen King")
	}
	require.Len(t, feed.Entries, 1)
	if feed.Entries[0].Title != "The Shining" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "The Shining")
	}
	if l := findLink(feed.Links, opdspkg.RelStart); l == nil {
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

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("content-type = %q, want %q", ct, opdspkg.NavContentType)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
}

func TestSeriesFeed_WithSeries(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	if _, err := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil); err != nil {
		require.NoError(t, err, "create series")
	}
	if _, err := h.DB.CreateSeries(ctx, "Discworld", nil, nil, nil); err != nil {
		require.NoError(t, err, "create series")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(feed.Entries))
	}
}

// --- Series books feed ---

func TestSeriesBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	series, err := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")
	book, err := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	pos := 1.0
	require.NoError(t, h.DB.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}}), "set book series")

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "The Dark Tower" {
		t.Errorf("title = %q, want %q", feed.Title, "The Dark Tower")
	}
	require.Len(t, feed.Entries, 1)
	if feed.Entries[0].Title != "The Gunslinger" {
		t.Errorf("entry title = %q, want %q", feed.Entries[0].Title, "The Gunslinger")
	}
	if l := findLink(feed.Links, opdspkg.RelStart); l == nil {
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
	ctx := t.Context()

	if _, err := h.DB.CreateBook(ctx, "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(ctx, "The Shining", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(ctx, "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=The", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

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

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(feed.Entries))
	}
}

func TestSearch_SpecialCharsInQuery(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	if _, err := h.DB.CreateBook(ctx, "100% Pure", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}
	if _, err := h.DB.CreateBook(ctx, "Other Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "create book")
	}

	// Search for "%" should not match everything due to LIKE wildcard escaping.
	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=%25", nil) // %25 = URL-encoded "%"
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

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

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, opdspkg.RelSelf)
	require.NotNil(t, selfLink)
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

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.SearchType {
		t.Errorf("content-type = %q, want %q", ct, opdspkg.SearchType)
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
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// Create a temp file to serve.
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("fake epub content"), 0o644), "write temp file")

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 17, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
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
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 100, nil, "/nonexistent/path.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload_UnknownFileType(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.xyz")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644), "write temp file")

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "xyz", "test.xyz", 4, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	if ct := w.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("content-type = %q, want %q", ct, "application/octet-stream")
	}
}
