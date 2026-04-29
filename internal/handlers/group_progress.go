package handlers

import (
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// handleGroupProgress dispatches GET for /api/groups/{id}/progress.
//
//	@Summary		Get group reading progress
//	@Description	Returns reading progress for all group members on a specific book (requester must be a member)
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string	true	"Group ID"
//	@Param			book_id	query		string	true	"Book ID"
//	@Success		200		{array}		groupMemberProgressDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id}/progress [get]
func (h *GroupHandler) handleGroupProgress(w http.ResponseWriter, r *http.Request, groupID string) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx := r.Context()
	bookID := r.URL.Query().Get("book_id")
	if bookID == "" {
		writeError(ctx, w, http.StatusBadRequest, "book_id is required")
		return
	}

	userID := auth.UserIDFromContext(ctx)
	progress, err := h.DB.ListGroupMemberProgress(ctx, groupID, bookID, userID)
	if handleOpErr(ctx, w, err, "group", "failed to list group member progress",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.BookID, bookID),
	) {
		return
	}

	writeJSON(ctx, w, http.StatusOK, mapSlice(progress, toGroupMemberProgressDTO))
}
