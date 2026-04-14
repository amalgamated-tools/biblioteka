package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

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

func toAdminUserDTO(u *db.User) adminUserDTO {
	return adminUserDTO{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		IsAdmin:    u.IsAdmin,
		OIDCLinked: u.OIDCSubject != nil,
		CreatedAt:  u.CreatedAt,
	}
}

// HandleListUsers returns the full list of users in the system (admin only).
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

	if !requireAdmin(h.DB, w, r) {
		return
	}

	listEntities(w, r, "users", h.DB.ListUsers, toAdminUserDTO)
}

// HandleFTSRebuild triggers a full rebuild of the FTS5 full-text search index
// (admin only, SQLite only). On PostgreSQL the pg_trgm GIN indexes used for
// search are maintained automatically, so this endpoint returns successfully
// without performing any work.
//
// Rebuilding is necessary after running SQLite's VACUUM command, which can
// silently remap rowids and corrupt the content-table FTS index. Biblioteka
// also checks and auto-rebuilds the index at startup.
//
//	@Summary		Rebuild search index
//	@Description	Triggers a full rebuild of the FTS5 search index (admin only, SQLite only)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401	{object}	errorResponse
//	@Failure		403	{object}	errorResponse
//	@Failure		405	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Success		202	{object}	object{message=string}
//	@Router			/admin/search/reindex [post]
func (h *AdminHandler) HandleFTSRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !requireAdmin(h.DB, w, r) {
		return
	}

	callerID := auth.UserIDFromContext(r.Context())
	rebuildBaseCtx := context.WithoutCancel(r.Context())

	// Run the rebuild asynchronously so the response is not blocked by the
	// server's 10 s WriteTimeout. The rebuild context is decoupled from request
	// cancellation so it survives after the HTTP handler returns, while still
	// preserving request-scoped values for logs and audit records.
	go func() {
		rebuildCtx, cancel := context.WithTimeout(rebuildBaseCtx, 5*time.Minute)
		defer cancel()

		if err := h.DB.RebuildFTS(rebuildCtx); err != nil {
			slog.ErrorContext(rebuildCtx, "FTS rebuild failed", slog.Any(otelkeys.Error, err))
			return
		}

		logAudit(rebuildCtx, h.DB, callerID, db.AuditActionFTSRebuilt, "fts", "search_index", nil)
		slog.InfoContext(rebuildCtx, "FTS rebuild completed successfully")
	}()

	writeJSON(r.Context(), w, http.StatusAccepted, map[string]string{"message": "search index rebuild started"})
}

// @Summary		Set user admin status
// @Description	Change a user's admin status (admin only)
// @Tags			Admin
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Failure		401		{object}	errorResponse
// @Param			id		path		string			true	"User ID"
// @Param			body	body		setAdminRequest	true	"Set admin request"
// @Success		200		{object}	object{message=string}
// @Failure		400		{object}	errorResponse
// @Failure		403		{object}	errorResponse
// @Failure		404		{object}	errorResponse
// @Failure		500		{object}	errorResponse
// @Router			/admin/users/{id} [put]
func (h *AdminHandler) HandleSetAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !requireAdmin(h.DB, w, r) {
		return
	}

	callerID := auth.UserIDFromContext(r.Context())

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
		slog.ErrorContext(r.Context(), "failed to update admin status",
			slog.String(otelkeys.TargetID, targetID),
			slog.Bool(otelkeys.IsAdmin, req.IsAdmin),
			slog.Any(otelkeys.Error, err),
		)
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
