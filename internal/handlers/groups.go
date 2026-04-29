package handlers

import (
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
	deleteUserOwnedResource(h.DB, w, r, id, "reading group", "reading_group", otelkeys.GroupID,
		h.DB.GetGroup, h.DB.DeleteGroup,
		db.AuditActionGroupDeleted,
		func(g *db.ReadingGroup) map[string]any { return map[string]any{"name": g.Name} },
	)
}
