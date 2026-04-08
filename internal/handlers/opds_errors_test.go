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
