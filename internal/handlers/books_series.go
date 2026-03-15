package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// respondBookSeries fetches and writes the series list for a book as JSON.
func (h *BookHandler) respondBookSeries(ctx context.Context, w http.ResponseWriter, bookID string) {
	entries, err := h.DB.GetBookSeries(ctx, bookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get book series", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to get book series")
		return
	}
	dtos := make([]bookSeriesEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, bookSeriesEntryDTO{
			Series:   toSeriesDTO(&e.Series),
			Position: e.Position,
		})
	}
	writeJSON(ctx, w, http.StatusOK, dtos)
}

// setBookSeriesRequest is the request body for setting book series.
type setBookSeriesRequest struct {
	Entries []db.BookSeriesInput `json:"entries"`
}

// getBookSeries godoc
//
//	@Summary		List book series
//	@Description	Get the list of series for a book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		bookSeriesEntryDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/series [get]
func (h *BookHandler) getBookSeries(w http.ResponseWriter, r *http.Request, bookID string) {
	h.respondBookSeries(r.Context(), w, bookID)
}

// putBookSeries godoc
//
//	@Summary		Set book series
//	@Description	Replace the list of series for a book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string					true	"Book ID"
//	@Param			body	body		setBookSeriesRequest	true	"Series entries"
//	@Success		200		{array}		bookSeriesEntryDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id}/series [put]
func (h *BookHandler) putBookSeries(w http.ResponseWriter, r *http.Request, bookID string) {
	var req setBookSeriesRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if err := h.DB.SetBookSeries(r.Context(), bookID, req.Entries); err != nil {
		slog.ErrorContext(r.Context(), "failed to set book series", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to set book series")
		return
	}
	h.respondBookSeries(r.Context(), w, bookID)
}
