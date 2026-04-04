package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
)

// --- DB error paths (opds_feeds.go) ---

func TestAllBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.AcqContentType)
	}
	// Response must still be valid XML (OPDS error feed)
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestRecentBooks_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/recent", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.AcqContentType)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestAuthorsFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.NavContentType)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSeriesFeed_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.NavContentType)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

func TestSearch_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/search?q=test", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.AcqContentType)
	}
	parseOPDSFeed(t, w.Body.Bytes())
}

// --- bookEntries error paths ---

func TestBookEntries_AuthorLoadError(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	book, err := h.DB.CreateBook(ctx, "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	// Close DB so batch author/file loads fail.
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

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
	if len(entries[0].Links) != 0 {
		t.Errorf("links = %v, want empty (batch load failed)", entries[0].Links)
	}
}
