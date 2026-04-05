package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func setupSeriesHandler(t *testing.T) (*SeriesHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &SeriesHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestCreateSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{Name: "The Dark Tower"})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto seriesDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "The Dark Tower", dto.Name)
}

func TestCreateSeries_MissingName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSeries_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	body := mustMarshal(t, seriesRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/series", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSeries_WhitespaceOnlyName(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	body := mustMarshal(t, seriesRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPut, "/api/series/"+s.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusConflict, w2.Code)
}

func TestListSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	_, err := h.DB.CreateSeries(t.Context(), "Discworld", nil, nil, nil)
	require.NoError(t, err, "create series")
	_, err = h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	r := httptest.NewRequest(http.MethodGet, "/api/series", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeriesList(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []seriesDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 2)
}

func TestGetSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetSeries_NotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/series/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSeries_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	r := httptest.NewRequest(http.MethodDelete, "/api/series/"+s.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteSeries_NotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/series/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "series not found", resp.Error)
}

func TestListSeriesBooks_Handler(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "create series")

	b1, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	b2, err := h.DB.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	require.NoError(t, h.DB.SetBookSeries(t.Context(), b1.ID, []db.BookSeriesInput{{SeriesID: s.ID}}), "set book series")
	require.NoError(t, h.DB.SetBookSeries(t.Context(), b2.ID, []db.BookSeriesInput{{SeriesID: s.ID}}), "set book series")

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	require.Equal(t, 2, result.Total)
	require.Len(t, result.Books, 2)
}

func TestListSeriesBooks_SeriesNotFound(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/series/nonexistent/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSeriesBooks_Empty(t *testing.T) {
	h, userID := setupSeriesHandler(t)

	s, err := h.DB.CreateSeries(t.Context(), "Empty Series", nil, nil, nil)
	require.NoError(t, err, "create series")

	r := httptest.NewRequest(http.MethodGet, "/api/series/"+s.ID+"/books", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleSeries(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var result bookListDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result), "unmarshal")
	require.Equal(t, 0, result.Total)
	require.NotNil(t, result.Books)
}
