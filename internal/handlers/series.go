package handlers

import (
	"context"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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

// seriesOps returns the namedEntityOps configuration for the Series entity.
func (h *SeriesHandler) seriesOps() namedEntityOps[db.Series, seriesDTO, seriesRequest] {
	return namedEntityOps[db.Series, seriesDTO, seriesRequest]{
		db:             h.DB,
		entityLabel:    "series",
		entityArticle:  "a series",
		idKey:          otelkeys.SeriesID,
		errInvalidName: db.ErrInvalidSeriesName,
		errNameExists:  db.ErrSeriesNameExists,
		auditCreate:    db.AuditActionSeriesCreated,
		auditUpdate:    db.AuditActionSeriesUpdated,
		get:            h.DB.GetSeries,
		create: func(ctx context.Context, req seriesRequest) (*db.Series, error) {
			return h.DB.CreateSeries(ctx, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
		},
		update: func(ctx context.Context, id string, req seriesRequest) (*db.Series, error) {
			return h.DB.UpdateSeries(ctx, id, req.Name, req.GoodreadsID, req.HardcoverID, req.GoogleBooksID)
		},
		reqName:    func(req seriesRequest) string { return req.Name },
		entityName: func(s *db.Series) string { return s.Name },
		entityID:   func(s *db.Series) string { return s.ID },
		toDTO:      toSeriesDTO,
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
//
//	@Summary		List series
//	@Description	Returns all series
//	@Tags			Series
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		seriesDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/series [get]
func (h *SeriesHandler) listSeries(w http.ResponseWriter, r *http.Request) {
	listEntities(w, r, "series", h.DB.ListSeries, toSeriesDTO)
}

// createSeries godoc
//
//	@Summary		Create a series
//	@Description	Create a new series
//	@Tags			Series
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		seriesRequest	true	"Series data"
//	@Success		201		{object}	seriesDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/series [post]
func (h *SeriesHandler) createSeries(w http.ResponseWriter, r *http.Request) {
	createNamedEntity(h.seriesOps(), w, r)
}

// getSeries godoc
//
//	@Summary		Get a series
//	@Description	Returns a single series by ID
//	@Tags			Series
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Series ID"
//	@Success		200	{object}	seriesDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/series/{id} [get]
func (h *SeriesHandler) getSeries(w http.ResponseWriter, r *http.Request, id string) {
	getNamedEntity(h.seriesOps(), w, r, id)
}

// updateSeries godoc
//
//	@Summary		Update a series
//	@Description	Update an existing series
//	@Tags			Series
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"Series ID"
//	@Param			body	body		seriesRequest	true	"Series data"
//	@Success		200		{object}	seriesDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/series/{id} [put]
func (h *SeriesHandler) updateSeries(w http.ResponseWriter, r *http.Request, id string) {
	updateNamedEntity(h.seriesOps(), w, r, id)
}

// deleteSeries godoc
//
//	@Summary		Delete a series
//	@Description	Delete a series by ID
//	@Tags			Series
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Series ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/series/{id} [delete]
func (h *SeriesHandler) deleteSeries(w http.ResponseWriter, r *http.Request, id string) {
	deleteResource(h.DB, w, r, id, "series", otelkeys.SeriesID,
		h.DB.GetSeries, h.DB.DeleteSeries,
		db.AuditActionSeriesDeleted,
		func(s *db.Series) map[string]any { return map[string]any{"name": s.Name} },
	)
}
