package handlers

import (
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (h *GroupHandler) handleGroupLists(w http.ResponseWriter, r *http.Request, groupID string) {
	switch r.Method {
	case http.MethodGet:
		h.listGroupLists(w, r, groupID)
	case http.MethodPost:
		h.shareList(w, r, groupID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listGroupLists returns all reading lists shared with a group.
//
//	@Summary		List group reading lists
//	@Description	Returns all reading lists shared with a reading group (requester must be a member)
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{array}		readingListDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id}/lists [get]
func (h *GroupHandler) listGroupLists(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	lists, err := h.DB.ListGroupReadingLists(ctx, groupID, userID)
	if handleOpErr(ctx, w, err, "group", "failed to list group reading lists",
		slog.String(otelkeys.GroupID, groupID),
	) {
		return
	}
	writeJSON(ctx, w, http.StatusOK, mapSlice(lists, toReadingListDTO))
}

// shareList shares a reading list with a group.
//
//	@Summary		Share a reading list with a group
//	@Description	Share a reading list with a reading group. The user must own the list and be a group member.
//	@Tags			Groups
//	@Accept			json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string				true	"Group ID"
//	@Param			body	body		shareListRequest	true	"List to share"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id}/lists [post]
func (h *GroupHandler) shareList(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx := r.Context()
	var req shareListRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if req.ListID == "" {
		writeError(ctx, w, http.StatusBadRequest, "list_id is required")
		return
	}

	userID := auth.UserIDFromContext(ctx)
	_, err := h.DB.ShareListWithGroup(ctx, groupID, req.ListID, userID)
	if handleOpErr(ctx, w, err, "reading list", "failed to share reading list with group",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.ReadingListID, req.ListID),
	) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupListShared, "group", groupID,
		map[string]any{otelkeys.ListID: req.ListID},
	)
	w.WriteHeader(http.StatusNoContent)
}

func (h *GroupHandler) handleGroupList(w http.ResponseWriter, r *http.Request, groupID, listID string) {
	switch r.Method {
	case http.MethodDelete:
		h.unshareList(w, r, groupID, listID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// unshareList removes a reading list from a group.
//
//	@Summary		Unshare a reading list from a group
//	@Description	Remove a reading list from a reading group. The user must own the list.
//	@Tags			Groups
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string	true	"Group ID"
//	@Param			listID	path		string	true	"Reading list ID"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id}/lists/{listID} [delete]
func (h *GroupHandler) unshareList(w http.ResponseWriter, r *http.Request, groupID, listID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	if handleOpErr(ctx, w, h.DB.UnshareListFromGroup(ctx, groupID, listID, userID), "reading list", "failed to unshare reading list from group",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.ReadingListID, listID),
	) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupListUnshared, "group", groupID,
		map[string]any{otelkeys.ListID: listID},
	)
	w.WriteHeader(http.StatusNoContent)
}
