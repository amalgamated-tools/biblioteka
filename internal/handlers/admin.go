package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// AdminHandler holds dependencies for admin endpoints.
type AdminHandler struct {
	DB *db.DB
}

type adminUserDTO struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	IsAdmin    bool         `json:"is_admin"`
	OIDCLinked bool         `json:"oidc_linked"`
	CreatedAt  db.Timestamp `json:"created_at"`
}

type setAdminRequest struct {
	IsAdmin bool `json:"is_admin"`
}

// HandleListUsers godoc
// @Summary     List all users
// @Description Returns a list of all users (admin only)
// @Tags        Admin
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {array}  adminUserDTO
// @Failure     403 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /admin/users [get]
func (h *AdminHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "admin listing users", slog.String("caller_id", userID))
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.Error("failed to check admin status", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	users, err := h.DB.ListUsers()
	if err != nil {
		slog.Error("failed to list users", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	slog.DebugContext(r.Context(), "users listed", slog.Int("count", len(users)))
	dtos := make([]adminUserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, adminUserDTO{
			ID:         u.ID,
			Name:       u.Name,
			Email:      u.Email,
			IsAdmin:    u.IsAdmin,
			OIDCLinked: u.OIDCSubject != nil,
			CreatedAt:  u.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, dtos)
}

// HandleSetAdmin godoc
// @Summary     Set user admin status
// @Description Change a user's admin status (admin only)
// @Tags        Admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       id   path     string          true "User ID"
// @Param       body body     setAdminRequest true "Set admin request"
// @Success     200  {object} object{message=string}
// @Failure     400  {object} errorResponse
// @Failure     403  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /admin/users/{id} [put]
func (h *AdminHandler) HandleSetAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "setting admin status", slog.String("caller_id", callerID))
	isAdmin, err := h.DB.IsAdmin(r.Context(), callerID)
	if err != nil {
		slog.Error("failed to check admin status", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	targetID, ok := extractPathID(r.URL.Path, "/api/admin/users/")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if targetID == callerID {
		writeError(w, http.StatusBadRequest, "cannot change your own admin status")
		return
	}

	var req setAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.DB.SetAdmin(targetID, req.IsAdmin); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("failed to set admin status", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update admin status")
		return
	}

	slog.DebugContext(r.Context(), "admin status updated", slog.String("target_id", targetID), slog.Bool("is_admin", req.IsAdmin))

	writeJSON(w, http.StatusOK, map[string]string{"message": "admin status updated"})
}
