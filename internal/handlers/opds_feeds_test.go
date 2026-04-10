package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.NavContentType, ct)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "Biblioteka OPDS Catalog", feed.Title)

	// Root feed has 4 navigation entries.
	require.Len(t, feed.Entries, 4)
	titles := []string{"All Books", "Recent Books", "Authors", "Series"}
	for i, want := range titles {
		require.Equal(t, want, feed.Entries[i].Title)
	}

	// Must have self, start, and search links.
	require.NotNil(t, findLink(feed.Links, opdspkg.RelSelf), "missing self link")
	require.NotNil(t, findLink(feed.Links, opdspkg.RelStart), "missing start link")
	require.NotNil(t, findLink(feed.Links, opdspkg.RelSearch), "missing search link")
}

func TestRootFeed_TrailingSlash(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRootFeed_HEAD(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodHead, "/opds", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

// --- All books feed ---

func TestAllBooks_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "All Books", feed.Title)
	require.Len(t, feed.Entries, 0)
}

func TestAllBooks_WithBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Alpha"})

	require.NoError(t, err, "create book Alpha")
	_, err = h.DB.CreateBook(ctx, db.BookInput{Title: "Beta"})
	require.NoError(t, err, "create book Beta")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 2)
	// Books should be sorted by title.
	require.Equal(t, "Alpha", feed.Entries[0].Title)
}

func TestAllBooks_WithDescription(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	desc := "A great book"
	_, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Alpha", Description: &desc})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)
	require.NotNil(t, feed.Entries[0].Content)
	require.Equal(t, "A great book", feed.Entries[0].Content.Value)
}

func TestAllBooks_WithAuthorsAndFiles(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "The Gunslinger"})
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
	require.Len(t, entry.Authors, 1)
	require.Equal(t, "Stephen King", entry.Authors[0].Name)

	acqLink := findLink(entry.Links, opdspkg.RelAcquisition)
	require.NotNil(t, acqLink)
	require.Equal(t, "application/epub+zip", acqLink.Type)
}

// --- Recent books feed ---

func TestRecentBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateBook(ctx, db.BookInput{Title: "First"})

	require.NoError(t, err, "create book First")
	_, err = h.DB.CreateBook(ctx, db.BookInput{Title: "Second"})
	require.NoError(t, err, "create book Second")

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "Recent Books", feed.Title)
	require.Len(t, feed.Entries, 2)
}

// --- Authors feed ---

func TestAuthorsFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.NavContentType, ct)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 0)
	require.NotNil(t, findLink(feed.Links, opdspkg.RelStart), "missing start link")
}

func TestAuthorsFeed_WithAuthors(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateAuthor(ctx, "Brandon Sanderson", nil, nil, nil, nil)

	require.NoError(t, err, "create author")
	_, err = h.DB.CreateAuthor(ctx, "Anne McCaffrey", nil, nil, nil, nil)
	require.NoError(t, err, "create author")

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 2)

	// Each entry should have a subsection link.
	for i, e := range feed.Entries {
		require.NotNil(t, findLink(e.Links, opdspkg.RelSubsection), "entry[%d]: missing subsection link", i)
	}
}

// --- Author books feed ---

func TestAuthorBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	author, err := h.DB.CreateAuthor(ctx, "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "The Shining"})
	require.NoError(t, err, "create book")
	require.NoError(t, h.DB.SetBookAuthors(ctx, book.ID, []string{author.ID}), "set book authors")

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "Books by Stephen King", feed.Title)
	require.Len(t, feed.Entries, 1)
	require.Equal(t, "The Shining", feed.Entries[0].Title)
	require.NotNil(t, findLink(feed.Links, opdspkg.RelStart), "missing start link")
}

func TestAuthorBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// --- Series feed ---

func TestSeriesFeed_Empty(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.NavContentType, ct)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 0)
}

func TestSeriesFeed_WithSeries(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)

	require.NoError(t, err, "create series")
	_, err = h.DB.CreateSeries(ctx, "Discworld", nil, nil, nil)
	require.NoError(t, err, "create series")

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 2)
}

// --- Series books feed ---

func TestSeriesBooks(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	series, err := h.DB.CreateSeries(ctx, "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")
	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	pos := 1.0
	require.NoError(t, h.DB.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{{SeriesID: series.ID, Position: &pos}}), "set book series")

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "The Dark Tower", feed.Title)
	require.Len(t, feed.Entries, 1)
	require.Equal(t, "The Gunslinger", feed.Entries[0].Title)
	require.NotNil(t, findLink(feed.Links, opdspkg.RelStart), "missing start link")
}

func TestSeriesBooks_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/series/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// --- Search ---

func TestSearch_WithResults(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateBook(ctx, db.BookInput{Title: "The Gunslinger"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(ctx, db.BookInput{Title: "The Shining"})
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(ctx, db.BookInput{Title: "Dune"})
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=The", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 2)
	require.Contains(t, feed.Title, "The")
}

func TestSearch_NoResults(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 0)
}

func TestSearch_SpecialCharsInQuery(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	_, err := h.DB.CreateBook(ctx, db.BookInput{Title: "100% Pure"})

	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBook(ctx, db.BookInput{Title: "Other Book"})
	require.NoError(t, err, "create book")

	// Search for "%" should not match everything due to LIKE wildcard escaping.
	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=%25", nil) // %25 = URL-encoded "%"
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)
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
	require.NotContains(t, selfLink.Href, "q=foo bar")
}

// --- OpenSearch description ---

func TestOpenSearchDescription(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/search", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.SearchType, ct)
	body := w.Body.String()
	require.Contains(t, body, "OpenSearchDescription")
	require.Contains(t, body, "{searchTerms}")
}

// --- Download ---

func TestDownload_Success(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")

	// Create a temp file to serve and register it as a library root.
	tmpDir := t.TempDir()
	registerTestLibrary(t, h.DB, tmpDir)
	filePath := filepath.Join(tmpDir, "test.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("fake epub content"), 0o644), "write temp file")

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 17, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, "application/epub+zip", ct)
	disp := w.Header().Get("Content-Disposition")
	require.Contains(t, disp, "test.epub", "content-disposition should contain filename")
	require.Equal(t, "fake epub content", w.Body.String())
}

func TestDownload_NotFound(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds/download/nonexistent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownload_FileMissing(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	tmpDir := t.TempDir()
	registerTestLibrary(t, h.DB, tmpDir)

	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(ctx, book.ID, "epub", "test.epub", 100, nil, filepath.Join(tmpDir, "nonexistent.epub"))
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownload_UnknownFileType(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")

	tmpDir := t.TempDir()
	registerTestLibrary(t, h.DB, tmpDir)
	filePath := filepath.Join(tmpDir, "test.xyz")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644), "write temp file")

	bf, err := h.DB.CreateBookFile(ctx, book.ID, "xyz", "test.xyz", 4, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/opds/download/"+bf.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, "application/octet-stream", ct)
}
