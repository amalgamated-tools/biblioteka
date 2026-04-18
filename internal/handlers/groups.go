package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// GroupHandler holds dependencies for reading group endpoints.
type GroupHandler struct {
	DB *db.DB
}

type groupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type addGroupMemberRequest struct {
	UserID string `json:"user_id"`
}

type shareListRequest struct {
	ListID string `json:"list_id"`
}

type groupDTO struct {
	ID          string       `json:"id"`
	OwnerID     string       `json:"owner_id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	MemberCount int          `json:"member_count"`
	CreatedAt   db.Timestamp `json:"created_at"`
	UpdatedAt   db.Timestamp `json:"updated_at"`
}

type groupMemberDTO struct {
	GroupID  string       `json:"group_id"`
	UserID   string       `json:"user_id"`
	UserName string       `json:"user_name"`
	Role     string       `json:"role"`
	JoinedAt db.Timestamp `json:"joined_at"`
}

type groupMemberProgressDTO struct {
	UserID     string        `json:"user_id"`
	UserName   string        `json:"user_name"`
	Percentage float64       `json:"percentage"`
	UpdatedAt  *db.Timestamp `json:"updated_at"`
}

func toGroupDTO(g *db.ReadingGroup) groupDTO {
	return groupDTO{
		ID:          g.ID,
		OwnerID:     g.OwnerID,
		Name:        g.Name,
		Description: g.Description,
		MemberCount: g.MemberCount,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func toGroupMemberDTO(m *db.ReadingGroupMember) groupMemberDTO {
	return groupMemberDTO{
		GroupID:  m.GroupID,
		UserID:   m.UserID,
		UserName: m.UserName,
		Role:     m.Role,
		JoinedAt: m.JoinedAt,
	}
}

func toGroupMemberProgressDTO(p *db.GroupMemberProgress) groupMemberProgressDTO {
	return groupMemberProgressDTO{
		UserID:     p.UserID,
		UserName:   p.UserName,
		Percentage: p.Percentage,
		UpdatedAt:  p.UpdatedAt,
	}
}

// HandleGroups handles GET /api/groups and POST /api/groups.
func (h *GroupHandler) HandleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listGroups(w, r)
	case http.MethodPost:
		h.createGroup(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleGroupRoutes dispatches /api/groups/{id} and sub-paths.
func (h *GroupHandler) HandleGroupRoutes(w http.ResponseWriter, r *http.Request) {
	id, sub, ok := extractPathSegments(r.URL.Path, "/api/groups/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid group ID")
		return
	}

	switch {
	case sub == "":
		h.handleGroup(w, r, id)
	case sub == "members":
		h.handleGroupMembers(w, r, id)
	case strings.HasPrefix(sub, "members/"):
		memberID := strings.TrimPrefix(sub, "members/")
		if memberID == "" {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid member ID")
			return
		}
		h.handleGroupMember(w, r, id, memberID)
	case sub == "lists":
		h.handleGroupLists(w, r, id)
	case strings.HasPrefix(sub, "lists/"):
		listID := strings.TrimPrefix(sub, "lists/")
		if listID == "" {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid list ID")
			return
		}
		h.handleGroupList(w, r, id, listID)
	case sub == "progress":
		h.handleGroupProgress(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusNotFound, "not found")
	}
}

// listGroups returns all groups for the authenticated user.
//
//	@Summary		List groups
//	@Description	Returns all reading groups the authenticated user belongs to
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		groupDTO
//	@Failure		500	{object}	errorResponse
//	@Router			/groups [get]
func (h *GroupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	listUserEntities(w, r, "groups", h.DB.ListGroups, toGroupDTO)
}

// createGroup creates a new reading group.
//
//	@Summary		Create a group
//	@Description	Create a new reading group owned by the authenticated user
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		groupRequest	true	"Group data"
//	@Success		201		{object}	groupDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups [post]
func (h *GroupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req groupRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if !validateName(ctx, w, req.Name) {
		return
	}

	userID := auth.UserIDFromContext(ctx)
	g, err := h.DB.CreateGroup(ctx, userID, req.Name, req.Description)
	if err != nil {
		if handleNameErr(ctx, w, err, db.ErrInvalidGroupName, db.ErrGroupNameExists, "a group") {
			return
		}
		slog.ErrorContext(ctx, "failed to create group", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create group")
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupCreated, "group", g.ID,
		map[string]any{"name": g.Name},
	)
	writeJSON(ctx, w, http.StatusCreated, toGroupDTO(g))
}

func (h *GroupHandler) handleGroup(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.getGroup(w, r, id)
	case http.MethodPut:
		h.updateGroup(w, r, id)
	case http.MethodDelete:
		h.deleteGroup(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getGroup returns a single group by ID.
//
//	@Summary		Get a group
//	@Description	Returns a single reading group by ID (requester must be a member)
//	@Tags			Groups
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		200	{object}	groupDTO
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id} [get]
func (h *GroupHandler) getGroup(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	g, err := h.DB.GetGroup(ctx, id, userID)
	if handleDBErr(ctx, w, err, "group") {
		return
	}
	writeJSON(ctx, w, http.StatusOK, toGroupDTO(g))
}

// updateGroup updates an existing group.
//
//	@Summary		Update a group
//	@Description	Update the name and description of a reading group (owner only)
//	@Tags			Groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"Group ID"
//	@Param			body	body		groupRequest	true	"Group data"
//	@Success		200		{object}	groupDTO
//	@Failure		400		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/groups/{id} [put]
func (h *GroupHandler) updateGroup(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	var req groupRequest
	if !decodeJSON(r, w, &req) {
		return
	}
	if !validateName(ctx, w, req.Name) {
		return
	}

	userID := auth.UserIDFromContext(ctx)
	g, err := h.DB.UpdateGroup(ctx, id, userID, req.Name, req.Description)
	if handleUpdateErr(ctx, w, err, db.ErrInvalidGroupName, db.ErrGroupNameExists, "a group", "group", id) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupUpdated, "group", g.ID,
		map[string]any{"name": g.Name},
	)
	writeJSON(ctx, w, http.StatusOK, toGroupDTO(g))
}

// deleteGroup deletes a reading group.
//
//	@Summary		Delete a group
//	@Description	Delete a reading group (owner only)
//	@Tags			Groups
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Param			id	path		string	true	"Group ID"
//	@Success		204	"No Content"
//	@Failure		400	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/groups/{id} [delete]
func (h *GroupHandler) deleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	if handleOpErr(ctx, w, h.DB.DeleteGroup(ctx, id, userID), "group", "failed to delete group",
		slog.String(otelkeys.GroupID, id),
	) {
		return
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupDeleted, "group", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

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
//	@Failure		401		{object}	errorResponse
//	@Param			id			path	string	true	"Group ID"
//	@Param			memberID	path	string	true	"Member user ID"
//	@Success		204			"No Content"
//	@Failure		400			{object}	errorResponse
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
		map[string]any{"list_id": req.ListID},
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
		map[string]any{"list_id": listID},
	)
	w.WriteHeader(http.StatusNoContent)
}

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
