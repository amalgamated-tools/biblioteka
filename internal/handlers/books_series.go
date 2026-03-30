package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// toBookSeriesEntryDTO converts a BookSeriesEntry to a bookSeriesEntryDTO.
func toBookSeriesEntryDTO(e *db.BookSeriesEntry) bookSeriesEntryDTO {
	return bookSeriesEntryDTO{
		Series:   toSeriesDTO(&e.Series),
		Position: e.Position,
	}
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
	respondBookSubResource(r.Context(), w, bookID, h.DB.GetBookSeries, toBookSeriesEntryDTO, "book series")
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
	putBookSubResource(w, r, bookID, h.DB.GetBookSeries, h.DB.SetBookSeries,
		func(req *setBookSeriesRequest) []db.BookSeriesInput { return req.Entries },
		toBookSeriesEntryDTO, "book series")
}
