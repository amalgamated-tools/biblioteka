package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// respondBookAuthors fetches and writes the author list for a book as JSON.
func (h *BookHandler) respondBookAuthors(ctx context.Context, w http.ResponseWriter, bookID string) {
	authors, err := h.DB.GetBookAuthors(ctx, bookID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get book authors", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to get book authors")
		return
	}
	dtos := make([]authorDTO, 0, len(authors))
	for i := range authors {
		dtos = append(dtos, toAuthorDTO(&authors[i]))
	}
	writeJSON(ctx, w, http.StatusOK, dtos)
}

// setBookAuthorsRequest is the request body for setting book authors.
type setBookAuthorsRequest struct {
	AuthorIDs []string `json:"author_ids"`
}

// getBookAuthors godoc
//
//	@Summary		List book authors
//	@Description	Get the list of authors for a book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		authorDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/authors [get]
func (h *BookHandler) getBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
	h.respondBookAuthors(r.Context(), w, bookID)
}

// putBookAuthors godoc
//
//	@Summary		Set book authors
//	@Description	Replace the list of authors for a book
//	@Tags			Books
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string					true	"Book ID"
//	@Param			body	body		setBookAuthorsRequest	true	"Author IDs"
//	@Success		200		{array}		authorDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/books/{id}/authors [put]
func (h *BookHandler) putBookAuthors(w http.ResponseWriter, r *http.Request, bookID string) {
	var req setBookAuthorsRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if err := h.DB.SetBookAuthors(r.Context(), bookID, req.AuthorIDs); err != nil {
		slog.ErrorContext(r.Context(), "failed to set book authors", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to set book authors")
		return
	}
	h.respondBookAuthors(r.Context(), w, bookID)
}
