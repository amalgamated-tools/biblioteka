package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func setupSeriesHandler(t *testing.T) (*SeriesHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &SeriesHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestCreateSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{Name: "The Dark Tower"})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto seriesDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "The Dark Tower" {
		t.Errorf("name = %q, want %q", dto.Name, "The Dark Tower")
	}
}

func TestCreateSeries_MissingName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSeries_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateSeries_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	body := mustMarshal(t, seriesRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPut, "/api/series/"+s.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSeries_Duplicate(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{Name: "The Dark Tower"})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleSeriesList(w, r)

	r2 := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleSeriesList(w2, r2)

	if w2.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusConflict)
	}
}

func TestListSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	if _, err := h.DB.CreateSeries(t.Context(), "Discworld", nil, nil, nil); err != nil {
		t.Fatalf("create series: %v", err)
	}
	if _, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil); err != nil {
		t.Fatalf("create series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []seriesDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
}

func TestGetSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetSeries_NotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/series/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/series/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != "series not found" {
		t.Errorf("error = %q, want %q", resp.Error, "series not found")
	}
}

func TestListSeriesBooks_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	b1, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	b2, err := h.DB.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	if err := h.DB.SetBookSeries(t.Context(), b1.ID, []db.BookSeriesInput{{SeriesID: s.ID}}); err != nil {
		t.Fatalf("set book series: %v", err)
	}
	if err := h.DB.SetBookSeries(t.Context(), b2.ID, []db.BookSeriesInput{{SeriesID: s.ID}}); err != nil {
		t.Fatalf("set book series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}
	if len(result.Books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(result.Books))
	}
}

func TestListSeriesBooks_SeriesNotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/series/nonexistent/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListSeriesBooks_Empty(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "Empty Series", nil, nil, nil)
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result bookListDTO
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total = %d, want 0", result.Total)
	}
	if result.Books == nil {
		t.Error("books should be empty slice, not nil")
	}
}
