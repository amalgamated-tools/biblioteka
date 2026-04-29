package handlers

import (
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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
