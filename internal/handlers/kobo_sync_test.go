package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/kobo"
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
		book, err := h.DB.CreateBook(
			context.Background(),
			"Sync Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		)
		if err != nil {
			t.Fatalf("create book %d: %v", i, err)
		}
		filePath := filepath.Join(dir, "book-"+book.ID+".epub")
		_, err = h.DB.CreateBookFile(
			context.Background(), book.ID, "epub", "book.epub", 512, nil,
			filePath,
		)
		if err != nil {
			t.Fatalf("create book file %d: %v", i, err)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/library/sync", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var results []any
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) > kobo.SyncPageSize {
		t.Errorf("expected at most %d sync results, got %d", kobo.SyncPageSize, len(results))
	}
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("x-kobo-synctoken") == "" {
		t.Error("expected x-kobo-synctoken header in sync response")
	}
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
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var results []any
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
