package handlers

import (
	"context"
	"log/slog"
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

// HandleSeries handles requests under /api/series/{id} and /api/series/{id}/books.
func (h *SeriesHandler) HandleSeries(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/series/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid series ID")
		return
	}

	switch sub {
	case "":
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
	case "books":
		switch r.Method {
		case http.MethodGet:
			h.listSeriesBooks(w, r, id)
		default:
			writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

type seriesListDTO struct {
	Series []seriesDTO `json:"series"`
	paginationMeta
}

// listSeries returns a paginated list of series.
//
//	@Summary		List series
//	@Description	Returns series with pagination
//	@Tags			Series
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int	false	"Number of items to skip (default 0)"
//	@Failure		401		{object}	errorResponse
//	@Success		200		{object}	seriesListDTO
//	@Failure		500		{object}	errorResponse
//	@Router			/series [get]
func (h *SeriesHandler) listSeries(w http.ResponseWriter, r *http.Request) {
	listPaginatedEntities(w, r, "series", h.DB.ListSeriesPaginated, toSeriesDTO,
		func(items []seriesDTO, total, limit, offset int) seriesListDTO {
			return seriesListDTO{
				Series: items,
				paginationMeta: paginationMeta{
					Total:  total,
					Limit:  limit,
					Offset: offset,
				},
			}
		},
	)
}

// createSeries creates a new series.
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

// getSeries returns a single series by ID.
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

// updateSeries updates an existing series.
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

// deleteSeries deletes a series by ID.
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
	deleteResource(h.DB, w, r, id, "series", "series", otelkeys.SeriesID,
		h.DB.GetSeries, h.DB.DeleteSeries,
		db.AuditActionSeriesDeleted,
		func(s *db.Series) map[string]any { return map[string]any{"name": s.Name} },
	)
}

// listSeriesBooks returns paginated books belonging to the specified series.
//
//	@Summary		List books in a series
//	@Description	Returns paginated books for a specific series, ordered by position then title
//	@Tags			Series
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Series ID"
//	@Param			limit	query		int		false	"Max items per page (default 50, max 200)"
//	@Param			offset	query		int		false	"Number of items to skip (default 0)"
//	@Success		200		{object}	bookListDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/series/{id}/books [get]
func (h *SeriesHandler) listSeriesBooks(w http.ResponseWriter, r *http.Request, seriesID string) {
	listParentBooks(w, r, seriesID,
		slog.String(otelkeys.SeriesID, seriesID),
		h.DB.ListBooksBySeriesPaginated,
		h.DB.GetSeries,
		"series",
	)
}
