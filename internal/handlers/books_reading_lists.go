package handlers

import (
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// getBookReadingLists returns all reading lists (owned by the authenticated
// user) that contain the specified book.
//
//	@Summary		List reading lists containing a book
//	@Description	Returns all reading lists owned by the authenticated user that contain the specified book
//	@Tags			Books
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Book ID"
//	@Success		200	{array}		readingListDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/books/{id}/reading-lists [get]
func (h *BookHandler) getBookReadingLists(w http.ResponseWriter, r *http.Request, bookID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	lists, err := h.DB.GetReadingListsForBook(ctx, bookID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get reading lists for book",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to get reading lists for book")
		return
	}

	writeJSON(ctx, w, http.StatusOK, mapSlice(lists, toReadingListDTO))
}
