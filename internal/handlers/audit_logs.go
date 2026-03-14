package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// AuditLogHandler holds dependencies for audit log endpoints.
type AuditLogHandler struct {
	DB *db.DB
}

type auditLogDTO struct {
	ID         string       `json:"id"`
	UserID     *string      `json:"user_id"`
	Action     string       `json:"action"`
	EntityType string       `json:"entity_type"`
	EntityID   string       `json:"entity_id"`
	Metadata   *string      `json:"metadata"`
	CreatedAt  db.Timestamp `json:"created_at"`
}

type auditLogListDTO struct {
	Entries []auditLogDTO `json:"entries"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
}

func toAuditLogDTO(e *db.AuditLog) auditLogDTO {
	return auditLogDTO{
		ID:         e.ID,
		UserID:     e.UserID,
		Action:     e.Action,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Metadata:   e.Metadata,
		CreatedAt:  e.CreatedAt,
	}
}

// requireAdmin checks whether the authenticated user is an admin and writes the
// appropriate error response if not. It returns true when the caller is allowed
// to proceed.
func (h *AuditLogHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := h.DB.IsAdmin(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to check admin status", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to verify permissions")
		return false
	}
	if !isAdmin {
		writeError(r.Context(), w, http.StatusForbidden, "admin access required")
		return false
	}
	return true
}

// HandleAuditLogs handles GET /api/audit-logs (admin only).
// It accepts optional query parameters:
//
//	limit  - number of entries per page (default 50, max 200)
//	offset - number of entries to skip (default 0)
//
// @Summary     List audit log entries
// @Description Returns a paginated list of all audit log entries (admin only)
// @Tags        Admin
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query    int false "Max entries to return (default 50)"
// @Param       offset query    int false "Entries to skip (default 0)"
// @Success     200    {object} auditLogListDTO
// @Failure     400    {object} errorResponse
// @Failure     401    {object} errorResponse
// @Failure     403    {object} errorResponse
// @Failure     500    {object} errorResponse
// @Router      /audit-logs [get]
func (h *AuditLogHandler) HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !h.requireAdmin(w, r) {
		return
	}

	const defaultLimit = 50
	const maxLimit = 200

	limit := defaultLimit
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid limit parameter")
			return
		}
		if n > maxLimit {
			n = maxLimit
		}
		limit = n
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid offset parameter")
			return
		}
		offset = n
	}

	slog.DebugContext(r.Context(), "listing audit logs",
		slog.Int("limit", limit),
		slog.Int("offset", offset),
	)

	entries, total, err := h.DB.ListAuditLogs(r.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list audit logs", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	dtos := make([]auditLogDTO, 0, len(entries))
	for i := range entries {
		dtos = append(dtos, toAuditLogDTO(&entries[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, auditLogListDTO{
		Entries: dtos,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
