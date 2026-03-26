package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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
//
//	@Summary		List all users
//	@Description	Returns a list of all users (admin only)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Success		200	{array}		adminUserDTO
//	@Failure		403	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/admin/users [get]
func (h *AdminHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "admin listing users", slog.String(otelkeys.CallerID, userID))
	if !requireAdmin(h.DB, w, r) {
		return
	}

	users, err := h.DB.ListUsers(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list users", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list users")
		return
	}

	slog.DebugContext(r.Context(), "users listed", slog.Int(otelkeys.Count, len(users)))
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

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

// HandleSetAdmin godoc
//
//	@Summary		Set user admin status
//	@Description	Change a user's admin status (admin only)
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			id		path		string			true	"User ID"
//	@Param			body	body		setAdminRequest	true	"Set admin request"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	errorResponse
//	@Failure		403		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/admin/users/{id} [put]
func (h *AdminHandler) HandleSetAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "setting admin status", slog.String(otelkeys.CallerID, callerID))
	isAdmin, err := h.DB.IsAdmin(r.Context(), callerID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "admin access required")
		return
	}

	targetID, ok := extractPathID(r.URL.Path, "/api/admin/users/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if targetID == callerID {
		writeError(r.Context(), w, http.StatusBadRequest, "cannot change your own admin status")
		return
	}

	var req setAdminRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if err := h.DB.SetAdmin(r.Context(), targetID, req.IsAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "user not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to set admin status", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update admin status")
		return
	}

	slog.DebugContext(r.Context(), "admin status updated",
		slog.String(otelkeys.TargetID, targetID),
		slog.Bool(otelkeys.IsAdmin, req.IsAdmin),
	)

	logAudit(r.Context(), h.DB, callerID, db.AuditActionAdminUpdated, "user", targetID, map[string]any{"is_admin": req.IsAdmin})

	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "admin status updated"})
}
