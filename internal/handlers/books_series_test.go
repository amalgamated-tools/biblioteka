package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func TestGetBookSeries_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len = %d, want 0", len(entries))
	}
}

func TestGetBookSeries_WithEntries(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	pos := 1.0
	if err := h.DB.SetBookSeries(t.Context(), b.ID, []db.BookSeriesInput{{SeriesID: s.ID, Position: &pos}}); err != nil {
		t.Fatalf("set book series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].Series.Name != "The Dark Tower" {
		t.Errorf("series name = %q, want %q", entries[0].Series.Name, "The Dark Tower")
	}
	if entries[0].Position == nil || *entries[0].Position != 1.0 {
		t.Errorf("position = %v, want 1.0", entries[0].Position)
	}
}

func TestPutBookSeries_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	pos := 1.0
	body := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{{SeriesID: s.ID, Position: &pos}},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].Series.Name != "The Dark Tower" {
		t.Errorf("series name = %q, want %q", entries[0].Series.Name, "The Dark Tower")
	}
}

func TestPutBookSeries_ClearsExisting(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	s1, err := h.DB.CreateSeries(t.Context(), "Series One", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series1: %v", err)
	}
	s2, err := h.DB.CreateSeries(t.Context(), "Series Two", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series2: %v", err)
	}

	pos1 := 1.0
	pos2 := 2.0

	// Set two series initially.
	body := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{
			{SeriesID: s1.ID, Position: &pos1},
			{SeriesID: s2.ID, Position: &pos2},
		},
	})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleBookRoutes(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("initial PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	// Replace with just one series.
	body2 := mustMarshal(t, setBookSeriesRequest{
		Entries: []db.BookSeriesInput{{SeriesID: s1.ID, Position: &pos1}},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body2))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleBookRoutes(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("replace PUT status = %d; body: %s", w2.Code, w2.Body.String())
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1 after replacement", len(entries))
	}
}

func TestPutBookSeries_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBookSeries_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPutBookSeries_EmptyEntries(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	body := mustMarshal(t, setBookSeriesRequest{Entries: []db.BookSeriesInput{}})
	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var entries []bookSeriesEntryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len = %d, want 0", len(entries))
	}
}
