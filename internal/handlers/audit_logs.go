package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// AuditLogHandler holds dependencies for audit log endpoints.
type AuditLogHandler struct {
	DB *db.DB
}

type auditLogDTO struct {
	ID         string          `json:"id"`
	UserID     *string         `json:"user_id"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Metadata   json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt  db.Timestamp    `json:"created_at"`
}

type auditLogListDTO struct {
	Entries []auditLogDTO `json:"entries"`
	paginationMeta
}

func toAuditLogDTO(e *db.AuditLog) auditLogDTO {
	var metadata json.RawMessage
	if e.Metadata != nil && *e.Metadata != "" {
		metadata = json.RawMessage(*e.Metadata)
	}

	return auditLogDTO{
		ID:         e.ID,
		UserID:     e.UserID,
		Action:     e.Action,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		Metadata:   metadata,
		CreatedAt:  e.CreatedAt,
	}
}

// HandleAuditLogs handles GET /api/audit-logs (admin only).
// It accepts optional query parameters:
//
//	limit  - number of entries per page (default 50, max 200)
//	offset - number of entries to skip (default 0)
//
//	@Summary		List audit log entries
//	@Description	Returns a paginated list of all audit log entries (admin only)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int	false	"Max entries to return (default 50, max 200; invalid or out-of-range values are silently clamped)"
//	@Param			offset	query		int	false	"Entries to skip (default 0; invalid or negative values are silently ignored)"
//	@Success		200		{object}	auditLogListDTO
//	@Failure		401		{object}	errorResponse
//	@Failure		403		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/audit-logs [get]
func (h *AuditLogHandler) HandleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !requireAdmin(h.DB, w, r) {
		return
	}

	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)

	slog.DebugContext(r.Context(), "listing audit logs",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	entries, total, err := h.DB.ListAuditLogs(r.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list audit logs", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, auditLogListDTO{
		Entries: mapSlice(entries, toAuditLogDTO),
		paginationMeta: paginationMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}
