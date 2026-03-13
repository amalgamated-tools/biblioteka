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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleSeries handles GET/PUT/DELETE /api/series/{id}.
func (h *SeriesHandler) HandleSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/series/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid series ID")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *SeriesHandler) listSeries(w http.ResponseWriter, _ *http.Request) {
	list, err := h.DB.ListSeries()
	if err != nil {
		slog.Error("failed to list series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list series")
		return
	}

	dtos := make([]seriesDTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, toSeriesDTO(&list[i]))
	}

	writeJSON(w, http.StatusOK, dtos)
}

func (h *SeriesHandler) createSeries(w http.ResponseWriter, r *http.Request) {
	var req seriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	s, err := h.DB.CreateSeries(req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
	if err != nil {
		if err == db.ErrSeriesNameExists {
			writeError(w, http.StatusConflict, "a series with that name already exists")
			return
		}
		slog.Error("failed to create series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create series")
		return
	}

	writeJSON(w, http.StatusCreated, toSeriesDTO(s))
}

func (h *SeriesHandler) getSeries(w http.ResponseWriter, _ *http.Request, id string) {
	s, err := h.DB.GetSeries(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "series not found")
			return
		}
		slog.Error("failed to get series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get series")
		return
	}

	writeJSON(w, http.StatusOK, toSeriesDTO(s))
}

func (h *SeriesHandler) updateSeries(w http.ResponseWriter, r *http.Request, id string) {
	var req seriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	s, err := h.DB.UpdateSeries(id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "series not found")
			return
		}
		if err == db.ErrSeriesNameExists {
			writeError(w, http.StatusConflict, "a series with that name already exists")
			return
		}
		slog.Error("failed to update series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update series")
		return
	}

	writeJSON(w, http.StatusOK, toSeriesDTO(s))
}

func (h *SeriesHandler) deleteSeries(w http.ResponseWriter, _ *http.Request, id string) {
	err := h.DB.DeleteSeries(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "series not found")
			return
		}
		slog.Error("failed to delete series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete series")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
