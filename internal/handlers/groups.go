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

func (h *GroupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	listUserEntities(w, r, "groups", h.DB.ListGroups, toGroupDTO)
}

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

func (h *GroupHandler) getGroup(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	g, err := h.DB.GetGroup(ctx, id, userID)
	if handleDBErr(ctx, w, err, "group") {
		return
	}
	writeJSON(ctx, w, http.StatusOK, toGroupDTO(g))
}

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

func (h *GroupHandler) deleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	if err := h.DB.DeleteGroup(ctx, id, userID); err != nil {
		if handleOpErr(ctx, w, err, "group", "failed to delete group",
			slog.String(otelkeys.GroupID, id),
		) {
			return
		}
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

func (h *GroupHandler) removeGroupMember(w http.ResponseWriter, r *http.Request, groupID, memberID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	if err := h.DB.RemoveGroupMember(ctx, groupID, userID, memberID); err != nil {
		if errors.Is(err, db.ErrOwnerCannotLeaveGroup) {
			writeError(ctx, w, http.StatusBadRequest, "owner cannot leave their own group")
			return
		}
		if handleOpErr(ctx, w, err, "group", "failed to remove group member",
			slog.String(otelkeys.GroupID, groupID),
		) {
			return
		}
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

func (h *GroupHandler) unshareList(w http.ResponseWriter, r *http.Request, groupID, listID string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	if err := h.DB.UnshareListFromGroup(ctx, groupID, listID, userID); err != nil {
		if handleOpErr(ctx, w, err, "reading list", "failed to unshare reading list from group",
			slog.String(otelkeys.GroupID, groupID),
			slog.String(otelkeys.ReadingListID, listID),
		) {
			return
		}
	}

	logAudit(ctx, h.DB, userID, db.AuditActionGroupListUnshared, "group", groupID,
		map[string]any{"list_id": listID},
	)
	w.WriteHeader(http.StatusNoContent)
}

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
