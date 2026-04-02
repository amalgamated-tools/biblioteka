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
		db:              h.DB,
		entityLabel:     "series",
		entityArticle:   "a series",
		idKey:           otelkeys.SeriesID,
		errInvalidName:  db.ErrInvalidSeriesName,
		errNameExists:   db.ErrSeriesNameExists,
		auditCreate:     db.AuditActionSeriesCreated,
		auditUpdate:     db.AuditActionSeriesUpdated,
		auditDelete:     db.AuditActionSeriesDeleted,
		pathPrefix:      "/api/series/",
		collectionLabel: "series",
		get:             h.DB.GetSeries,
		list:            h.DB.ListSeries,
		del:             h.DB.DeleteSeries,
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
func (h *SeriesHandler) HandleSeriesList(w http.ResponseWriter, r *http.Request) {
	handleNamedEntityCollection(h.seriesOps(), w, r)
}

// HandleSeries handles GET/PUT/DELETE /api/series/{id}.
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
func (h *SeriesHandler) HandleSeries(w http.ResponseWriter, r *http.Request) {
	handleNamedEntitySingle(h.seriesOps(), w, r)
}
