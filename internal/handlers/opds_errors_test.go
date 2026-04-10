package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"

	"github.com/stretchr/testify/require"
)

// --- DB error paths (opds_feeds.go) ---

func TestAllBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	// Response must still be valid XML (OPDS error feed)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestRecentBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestAuthorsFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.NavContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSeriesFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.NavContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSearch_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=test", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

// --- writeEntityBooksFeed error paths (getFn returns non-sql.ErrNoRows) ---

func TestAuthorBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	// Create an author so the ID is valid, then close DB to trigger a real DB error.
	author, err := h.DB.CreateAuthor(ctx, "DB Error Author", nil, nil, nil, nil)
	require.NoError(t, err, "create author")
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/authors/"+author.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSeriesBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	// Create a series so the ID is valid, then close DB to trigger a real DB error.
	series, err := h.DB.CreateSeries(ctx, "DB Error Series", nil, nil, nil)
	require.NoError(t, err, "create series")
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/series/"+series.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	parseOPDSFeed(t, w.Body.Bytes())
}

// --- bookEntries error paths ---

func TestBookEntries_AuthorLoadError(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")

	// Close DB so batch author/file loads fail.
	require.NoError(t, h.DB.Close(), "close db")

	books := []db.Book{*book}
	entries := h.bookEntries(ctx, books, "http://example.com/opds")

	// Should still return entries, just without authors or download links.
	require.Len(t, entries, 1)
	require.Equal(t, "Test Book", entries[0].Title)
	require.Len(t, entries[0].Authors, 0)
	require.Len(t, entries[0].Links, 0)
}
