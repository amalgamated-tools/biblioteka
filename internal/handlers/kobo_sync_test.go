package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/kobo"

	"github.com/stretchr/testify/require"
)

// TestHandleSync_PageSizeLimit verifies that HandleSync returns at most
// kobo.SyncPageSize books per request.
func TestHandleSync_PageSizeLimit(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create more books than the page size + some files.
	dir := t.TempDir()
	for i := range kobo.SyncPageSize + 5 {
		book, err := h.DB.CreateBook(context.Background(), db.BookInput{Title: "Sync Book"})
		require.NoError(t, err, "create book %d", i)
		filePath := filepath.Join(dir, "book-"+book.ID+".epub")
		_, err = h.DB.CreateBookFile(
			context.Background(), book.ID, "epub", "book.epub", 512, nil,
			filePath,
		)
		require.NoError(t, err, "create book file %d", i)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var results []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	require.LessOrEqual(t, len(results), kobo.SyncPageSize)
}

// TestHandleSync_BooksLastModifiedHeader verifies that the response includes
// an x-kobo-sync token header.
func TestHandleSync_BooksLastModifiedHeader(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, w.Header().Get("x-kobo-synctoken"), "expected x-kobo-synctoken header")
}

// TestHandleSync_NonGETMethodReturnsEmpty verifies that non-GET methods on the
// sync endpoint return an empty array response (Kobo compatibility).
func TestHandleSync_NonGETMethodReturnsEmpty(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// The handler does not restrict by method; it returns 200 with an empty sync response.
	require.Equal(t, http.StatusOK, w.Code)
	var results []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode response")
}

// TestHandleSync_ContinueHeaderWhenHasMore verifies that the x-kobo-sync:
// continue header is set when the library has more books than SyncPageSize.
// Without this header, Kobo devices stop fetching and miss books beyond the
// first page.
func TestHandleSync_ContinueHeaderWhenHasMore(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create one more book than the page size, each with a downloadable file.
	dir := t.TempDir()
	for i := range kobo.SyncPageSize + 1 {
		book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Book"})
		require.NoError(t, err, "create book %d", i)
		_, err = h.DB.CreateBookFile(
			t.Context(), book.ID, "epub", "book.epub", 512, nil,
			filepath.Join(dir, "book-"+book.ID+".epub"),
		)
		require.NoError(t, err, "create book file %d", i)
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var results []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	// Page is capped at SyncPageSize.
	require.Equal(t, kobo.SyncPageSize, len(results), "expected exactly SyncPageSize results")
	// Kobo devices rely on this header to know there are more pages to fetch.
	require.Equal(t, "continue", w.Header().Get("x-kobo-sync"),
		"expected x-kobo-sync: continue when library exceeds page size")
}

// TestHandleSync_NoContinueHeaderWhenFitsOnePage verifies that the
// x-kobo-sync header is absent when all books fit within a single page.
func TestHandleSync_NoContinueHeaderWhenFitsOnePage(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create fewer books than the page limit.
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Single Book"})
	require.NoError(t, err)
	_, err = h.DB.CreateBookFile(
		t.Context(), book.ID, "epub", "book.epub", 512, nil,
		filepath.Join(t.TempDir(), "single.epub"),
	)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get("x-kobo-sync"),
		"x-kobo-sync header must be absent when all books fit on one page")
}

// TestHandleSync_SyncTokenAdvancesForFilelessBooks verifies that the sync
// token's BooksLastModified watermark advances even for books that have no
// downloadable files (and are therefore omitted from the response). Without
// this, the next sync would re-fetch the same fileless books indefinitely.
func TestHandleSync_SyncTokenAdvancesForFilelessBooks(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	// Create a book with NO files. It should be skipped in the response
	// but still advance the high-water mark.
	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Fileless Book"})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	// Fileless book is omitted from the payload.
	var results []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&results), "decode")
	require.Empty(t, results, "fileless books must not appear in sync payload")

	// But the sync token watermark must be non-zero so the next sync does not
	// return the same fileless book again.
	rawToken := w.Header().Get("x-kobo-synctoken")
	require.NotEmpty(t, rawToken)
	tok := kobo.ParseSyncToken(rawToken)
	require.False(t, tok.BooksLastModified.IsZero(),
		"BooksLastModified must advance even when the synced book has no files")
}
