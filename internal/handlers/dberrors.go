package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// handleDBErr writes the appropriate HTTP error for a DB lookup error.
// Returns true if an error was written (caller should return).
func handleDBErr(ctx context.Context, w http.ResponseWriter, err error, resource string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, resource+" not found")
		return true
	}
	slog.ErrorContext(ctx, "failed to get resource",
		slog.String(otelkeys.Resource, resource),
		slog.Any(otelkeys.Error, err),
	)
	writeError(ctx, w, http.StatusInternalServerError, "failed to get "+resource)
	return true
}

// handleOpErr handles operation errors with a shared pattern:
// sql.ErrNoRows → 404 "<resource> not found", otherwise logs and writes
// 500 "<op>". Returns true when it wrote a response.
func handleOpErr(ctx context.Context, w http.ResponseWriter, err error, resource, op string, attrs ...slog.Attr) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, resource+" not found")
		return true
	}

	logAttrs := make([]any, 0, len(attrs)+1)
	for _, attr := range attrs {
		logAttrs = append(logAttrs, attr)
	}
	logAttrs = append(logAttrs, slog.Any(otelkeys.Error, err))

	slog.ErrorContext(ctx, op, logAttrs...)
	writeError(ctx, w, http.StatusInternalServerError, op)
	return true
}

// handleNameErr handles the ErrInvalidName / ErrNameExists error pattern that is
// common to all named-resource create and update handlers. resourceArticle should
// be the resource noun with its indefinite article, e.g. "an author" or "a series".
// Returns true if an error was handled and the caller should return immediately.
func handleNameErr(ctx context.Context, w http.ResponseWriter, err, errInvalid, errExists error, resourceArticle string) bool {
	if errors.Is(err, errInvalid) {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return true
	}
	if errors.Is(err, errExists) {
		writeError(ctx, w, http.StatusConflict, resourceArticle+" with that name already exists")
		return true
	}
	return false
}

// handleUpdateErr handles the full error block common to named-entity update
// handlers: sql.ErrNoRows → 404, invalid/duplicate name errors via handleNameErr,
// and a generic 500 fallback. Returns true when it wrote a response (caller should
// return).
func handleUpdateErr(ctx context.Context, w http.ResponseWriter, err, errInvalid, errExists error, resourceArticle, resource, id string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, resource+" not found")
		return true
	}
	if handleNameErr(ctx, w, err, errInvalid, errExists, resourceArticle) {
		return true
	}
	slog.ErrorContext(ctx, "failed to update entity",
		slog.String(otelkeys.Resource, resource),
		slog.String(otelkeys.EntityID, id),
		slog.Any(otelkeys.Error, err),
	)
	writeError(ctx, w, http.StatusInternalServerError, "failed to update "+resource)
	return true
}

// logAudit persists an audit log entry on a best-effort basis. Write failures
// are logged as warnings and intentionally not propagated to the caller, so
// audit issues never block the primary request flow.
func logAudit(ctx context.Context, d *db.DB, userID, action, resourceType, resourceID string, meta map[string]any) {
	if err := d.CreateAuditLog(ctx, userID, action, resourceType, resourceID, meta); err != nil {
		slog.WarnContext(ctx, "failed to write audit log", slog.Any(otelkeys.Error, err))
	}
}
