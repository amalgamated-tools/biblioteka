package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// validateName returns true if name is non-blank. On failure it writes a 400
// error response and returns false, so callers can simply return.
func validateName(ctx context.Context, w http.ResponseWriter, name string) bool {
	if strings.TrimSpace(name) == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return false
	}
	return true
}

// decodeJSON reads and decodes the JSON request body into v.
// It returns true on success. On failure it writes a 400 error response and
// returns false, so callers can simply return.
func decodeJSON(r *http.Request, w http.ResponseWriter, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB cap
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		slog.DebugContext(r.Context(), "failed to decode request body", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON response", slog.Any(otelkeys.Error, err))
	}
}

// errorResponse represents a JSON error returned by the API.
type errorResponse struct {
	Error string `json:"error"`
}

// writeError sends a JSON error response.
func writeError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON error response", slog.Any(otelkeys.Error, err))
	}
}

const minPasswordLength = 6

// validatePassword checks that a password meets the minimum length requirement.
// Returns an error message if invalid, or an empty string if valid.
func validatePassword(password string) string {
	if len(password) < minPasswordLength {
		return "password must be at least 6 characters"
	}
	return ""
}

// extractPathID extracts a single resource ID from a URL path by stripping the
// given prefix. It trims a trailing slash so that both /api/foo/123 and
// /api/foo/123/ resolve to "123". Returns the ID and true on success, or an
// empty string and false if the result is empty or contains additional segments.
func extractPathID(path, prefix string) (string, bool) {
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractPathSegments extracts a resource ID and optional sub-resource from a URL path.
// For "/api/books/abc123/authors" with prefix "/api/books/", it returns ("abc123", "authors", true).
// For "/api/books/abc123" with prefix "/api/books/", it returns ("abc123", "", true).
func extractPathSegments(path, prefix string) (id, sub string, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	return parts[0], parts[1], true
}

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

// logAudit persists an audit log entry on a best-effort basis. Write failures
// are logged as warnings and intentionally not propagated to the caller, so
// audit issues never block the primary request flow.
func logAudit(ctx context.Context, d *db.DB, userID, action, resourceType, resourceID string, meta map[string]any) {
	if err := d.CreateAuditLog(ctx, userID, action, resourceType, resourceID, meta); err != nil {
		slog.WarnContext(ctx, "failed to write audit log", slog.Any(otelkeys.Error, err))
	}
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
	slog.ErrorContext(ctx, "failed to update "+resource,
		slog.String(otelkeys.ID, id),
		slog.Any(otelkeys.Error, err),
	)
	writeError(ctx, w, http.StatusInternalServerError, "failed to update "+resource)
	return true
}

// mapSlice converts a slice of T to a slice of DTO using the provided converter.
func mapSlice[T any, DTO any](items []T, toDTO func(*T) DTO) []DTO {
	dtos := make([]DTO, 0, len(items))
	for i := range items {
		dtos = append(dtos, toDTO(&items[i]))
	}
	return dtos
}

// listEntities is a generic helper that implements the list-and-convert pattern
// common to named-entity list handlers. It fetches all entities, converts them
// to DTOs, and writes the JSON response.
func listEntities[T any, DTO any](
	w http.ResponseWriter,
	r *http.Request,
	resource string,
	list func(context.Context) ([]T, error),
	toDTO func(*T) DTO,
) {
	ctx := r.Context()
	slog.DebugContext(ctx, "listing "+resource)

	entities, err := list(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list "+resource, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+resource)
		return
	}

	slog.DebugContext(ctx, resource+" listed", slog.Int(otelkeys.Count, len(entities)))

	writeJSON(ctx, w, http.StatusOK, mapSlice(entities, toDTO))
}

// deleteResource is a generic helper that implements the fetch-then-delete-then-audit
// pattern common to all resource deletion handlers. It fetches the resource (to
// capture audit metadata), deletes it, writes an audit log entry, and responds
// with 204 No Content on success.
func deleteResource[T any](
	d *db.DB,
	w http.ResponseWriter,
	r *http.Request,
	id string,
	resource string,
	idKey string,
	get func(context.Context, string) (T, error),
	del func(context.Context, string) error,
	auditAction string,
	auditMeta func(T) map[string]any,
) {
	ctx := r.Context()
	slog.DebugContext(ctx, "deleting "+resource, slog.String(idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers

	entity, err := get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to get "+resource, slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	if err := del(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to delete "+resource, slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	userID := auth.UserIDFromContext(ctx)
	var meta map[string]any
	if auditMeta != nil && !isNilValue(entity) {
		meta = auditMeta(entity)
	}
	if err := d.CreateAuditLog(ctx, userID, auditAction, resource, id, meta); err != nil {
		slog.WarnContext(
			ctx,
			"failed to write audit log",
			slog.String(otelkeys.Resource, resource),
			slog.String(idKey, id), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
			slog.Any(otelkeys.Error, err),
		)
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteUserOwnedResource is a generic helper that implements the
// fetch-then-delete-then-audit pattern for user-scoped resources such as API
// keys and Kobo tokens. It mirrors deleteResource but accepts get/delete
// functions that require both a resource ID and a user ID. The resource
// parameter is a human-readable display name used in error messages (e.g.
// "API key"), while auditEntityType is a stable snake_case identifier written
// to the audit log (e.g. "api_key").
func deleteUserOwnedResource[T any](
	d *db.DB,
	w http.ResponseWriter,
	r *http.Request,
	id string,
	resource string,
	auditEntityType string,
	idKey string,
	get func(context.Context, string, string) (T, error),
	del func(context.Context, string, string) error,
	auditAction string,
	auditMeta func(T) map[string]any,
) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	slog.DebugContext(ctx, "deleting "+resource, slog.String(idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers

	entity, err := get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to get "+resource, slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	if err := del(ctx, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to delete "+resource, slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	var meta map[string]any
	if auditMeta != nil && !isNilValue(entity) {
		meta = auditMeta(entity)
	}
	if err := d.CreateAuditLog(ctx, userID, auditAction, auditEntityType, id, meta); err != nil {
		slog.WarnContext(
			ctx,
			"failed to write audit log",
			slog.String(otelkeys.Resource, auditEntityType),
			slog.String(idKey, id), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
			slog.Any(otelkeys.Error, err),
		)
	}

	w.WriteHeader(http.StatusNoContent)
}

// isNilValue reports whether v, when passed as any, wraps a nil pointer.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

// requireAdmin checks whether the authenticated user is an admin and writes the
// appropriate error response if not. It returns true when the caller is allowed
// to proceed. A deleted user (stale JWT) receives a generic 401 response.
func requireAdmin(d *db.DB, w http.ResponseWriter, r *http.Request) bool {
	userID := auth.UserIDFromContext(r.Context())
	isAdmin, err := d.IsAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(r.Context(), "user not found during admin check",
				slog.String(otelkeys.UserID, userID),
				slog.Any(otelkeys.Error, err),
			)
			writeError(r.Context(), w, http.StatusUnauthorized, "authentication required")
			return false
		}
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
