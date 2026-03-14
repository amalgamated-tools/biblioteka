package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// SeriesHandler holds dependencies for series endpoints.
type SeriesHandler struct {
	DB *db.DB
}

type seriesRequest struct {
	Name          string  `json:"name"`
	GoodreadsID   *string `json:"goodreads_id"`
	HardcoverID   *string `json:"hardcover_id"`
	GoogleBooksID *string `json:"google_books_id"`
}

type seriesDTO struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	GoodreadsID   *string      `json:"goodreads_id"`
	HardcoverID   *string      `json:"hardcover_id"`
	GoogleBooksID *string      `json:"google_books_id"`
	CreatedAt     db.Timestamp `json:"created_at"`
	UpdatedAt     db.Timestamp `json:"updated_at"`
}

func toSeriesDTO(s *db.Series) seriesDTO {
	return seriesDTO{
		ID:            s.ID,
		Name:          s.Name,
		GoodreadsID:   s.GoodreadsID,
		HardcoverID:   s.HardcoverID,
		GoogleBooksID: s.GoogleBooksID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// HandleSeriesList handles GET /api/series and POST /api/series.
func (h *SeriesHandler) HandleSeriesList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSeries(w, r)
	case http.MethodPost:
		h.createSeries(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleSeries handles GET/PUT/DELETE /api/series/{id}.
func (h *SeriesHandler) HandleSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/series/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid series ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSeries(w, r, id)
	case http.MethodPut:
		h.updateSeries(w, r, id)
	case http.MethodDelete:
		h.deleteSeries(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listSeries godoc
// @Summary     List series
// @Description Returns all series
// @Tags        Series
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {array}  seriesDTO
// @Failure     500 {object} errorResponse
// @Router      /series [get]
func (h *SeriesHandler) listSeries(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "listing series")
	list, err := h.DB.ListSeries(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list series", slog.Any("error", err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list series")
		return
	}

	slog.DebugContext(r.Context(), "series listed", slog.Int("count", len(list)))

	dtos := make([]seriesDTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, toSeriesDTO(&list[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

// createSeries godoc
// @Summary     Create a series
// @Description Create a new series
// @Tags        Series
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       body body     seriesRequest true "Series data"
// @Success     201  {object} seriesDTO
// @Failure     400  {object} errorResponse
// @Failure     409  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /series [post]
func (h *SeriesHandler) createSeries(w http.ResponseWriter, r *http.Request) {
	var req seriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "creating series", slog.String("name", req.Name))

	s, err := h.DB.CreateSeries(r.Context(), req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
	if err != nil {
		if err == db.ErrSeriesNameExists {
			writeError(r.Context(), w, http.StatusConflict, "a series with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to create series", slog.Any("error", err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create series")
		return
	}

	slog.DebugContext(r.Context(), "series created", slog.String("series_id", s.ID), slog.String("name", s.Name))
	writeJSON(r.Context(), w, http.StatusCreated, toSeriesDTO(s))
}

// getSeries godoc
// @Summary     Get a series
// @Description Returns a single series by ID
// @Tags        Series
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Series ID"
// @Success     200 {object} seriesDTO
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /series/{id} [get]
func (h *SeriesHandler) getSeries(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching series", slog.String("series_id", id))
	s, err := h.DB.GetSeries(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "series not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get series", slog.Any("error", err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get series")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toSeriesDTO(s))
}

// updateSeries godoc
// @Summary     Update a series
// @Description Update an existing series
// @Tags        Series
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id   path     string        true "Series ID"
// @Param       body body     seriesRequest true "Series data"
// @Success     200  {object} seriesDTO
// @Failure     400  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     409  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /series/{id} [put]
func (h *SeriesHandler) updateSeries(w http.ResponseWriter, r *http.Request, id string) {
	var req seriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}

	slog.DebugContext(r.Context(), "updating series", slog.String("series_id", id), slog.String("name", req.Name))

	s, err := h.DB.UpdateSeries(r.Context(), id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "series not found")
			return
		}
		if err == db.ErrSeriesNameExists {
			writeError(r.Context(), w, http.StatusConflict, "a series with that name already exists")
			return
		}
		slog.ErrorContext(r.Context(), "failed to update series", slog.Any("error", err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update series")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, toSeriesDTO(s))
}

// deleteSeries godoc
// @Summary     Delete a series
// @Description Delete a series by ID
// @Tags        Series
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id  path     string true "Series ID"
// @Success     204 "No Content"
// @Failure     400 {object} errorResponse
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /series/{id} [delete]
func (h *SeriesHandler) deleteSeries(w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "deleting series", slog.String("series_id", id))
	err := h.DB.DeleteSeries(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "series not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete series", slog.Any("error", err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete series")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
