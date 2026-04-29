package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func (h *GroupHandler) handleGroupMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	switch r.Method {
	case http.MethodGet:
		h.listGroupMembers(w, r, groupID)
	case http.MethodPost:
		h.addGroupMember(w, r, groupID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// listGroupMembers returns all members of a group.
//
//	@Summary		List group members
//	@Description	Returns all members of a reading group (requester must be a member)
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{array}		groupMemberDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id}/members [get]
func (h *GroupHandler) listGroupMembers(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	members, err := h.DB.ListGroupMembers(ctx, groupID, userID)
	if handleOpErr(ctx, w, err, "group", "failed to list group members",
		slog.String(otelkeys.GroupID, groupID),
	) {
		return
	}
	writeJSON(ctx, w, http.StatusOK, mapSlice(members, toGroupMemberDTO))
}

// addGroupMember adds a user to a reading group.
//
//	@Summary		Add a group member
//	@Description	Add a user to a reading group (owner only; idempotent)
//	@Tags			Groups
//	@Accept			json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string					true	"Group ID"
//	@Param			body	body		addGroupMemberRequest	true	"Member to add"
//	@Success		204		"No Content"
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id}/members [post]
func (h *GroupHandler) addGroupMember(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx := r.Context()
	var req addGroupMemberRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if req.UserID == "" {
		writeError(ctx, w, http.StatusBadRequest, "user_id is required")
		return
	}

	userID := auth.UserIDFromContext(ctx)
	added, err := h.DB.AddGroupMember(ctx, groupID, userID, req.UserID)
	if err != nil {
		if errors.Is(err, db.ErrMemberUserNotFound) {
			writeError(ctx, w, http.StatusNotFound, "user not found")
			return
		}
		if handleOpErr(ctx, w, err, "group", "failed to add group member",
			slog.String(otelkeys.GroupID, groupID),
		) {
			return
		}
	}

	if added {
		logAudit(ctx, h.DB, userID, db.AuditActionGroupMemberAdded, "group", groupID,
			map[string]any{"member_user_id": req.UserID},
		)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *GroupHandler) handleGroupMember(w http.ResponseWriter, r *http.Request, groupID, memberID string) {
	switch r.Method {
	case http.MethodDelete:
		h.removeGroupMember(w, r, groupID, memberID)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// removeGroupMember removes a member from a reading group.
//
//	@Summary		Remove a group member
//	@Description	Remove a member from a reading group. The owner can remove any member; members can remove themselves.
//	@Tags			Groups
//	@Security		BearerAuth
//	@Failure		401			{object}	errorResponse
//	@Param			id			path		string	true	"Group ID"
//	@Param			memberID	path		string	true	"Member user ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	errorResponse
//	@Failure		404			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Router			/groups/{id}/members/{memberID} [delete]
func (h *GroupHandler) removeGroupMember(w http.ResponseWriter, r *http.Request, groupID, memberID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	err := h.DB.RemoveGroupMember(ctx, groupID, userID, memberID)
	if errors.Is(err, db.ErrOwnerCannotLeaveGroup) {
		writeError(ctx, w, http.StatusBadRequest, "owner cannot leave their own group")
		return
	}
	if handleOpErr(ctx, w, err, "group", "failed to remove group member",
		slog.String(otelkeys.GroupID, groupID),
	) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupMemberRemoved, "group", groupID,
		map[string]any{"member_user_id": memberID},
	)
	w.WriteHeader(http.StatusNoContent)
}
