package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupSeriesHandler(t *testing.T) (*SeriesHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &SeriesHandler{DB: d}
	user, err := d.CreateUser(context.Background(), "Test User", "test@example.com", "password1")
	if err != nil {
		failNowf(t, "create user: %v", err)
	}
	return h, user.ID
}

func TestCreateSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body, _ := json.Marshal(seriesRequest{Name: "The Dark Tower"})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusCreated {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto seriesDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if dto.Name != "The Dark Tower" {
		failf(t, "name = %q, want %q", dto.Name, "The Dark Tower")
	}
}

func TestCreateSeries_MissingName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body, _ := json.Marshal(seriesRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusBadRequest {
		failf(t, "status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSeries_Duplicate(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body, _ := json.Marshal(seriesRequest{Name: "The Dark Tower"})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleSeriesList(w, r)

	r2 := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleSeriesList(w2, r2)

	if w2.Code != http.StatusConflict {
		failf(t, "status = %d, want %d", w2.Code, http.StatusConflict)
	}
}

func TestListSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	h.DB.CreateSeries(context.Background(), "Discworld", nil, nil, nil)
	h.DB.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []seriesDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(dtos) != 2 {
		failf(t, "len = %d, want 2", len(dtos))
	}
}

func TestGetSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, _ := h.DB.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetSeries_NotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/series/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, _ := h.DB.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	r := httptest.NewRequest(http.MethodDelete, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	if w.Code != http.StatusNoContent {
		failf(t, "status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
