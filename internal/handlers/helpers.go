package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// requestScheme returns the HTTP scheme for the given request. It checks
// r.TLS first, then honors the X-Forwarded-Proto header (normalized to
// lowercase, trimmed). Only "http" and "https" are accepted; any other
// value is ignored and the function falls back to the TLS-based default.
func requestScheme(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		normalized := strings.ToLower(strings.TrimSpace(proto))
		if normalized == "http" || normalized == "https" {
			scheme = normalized
		}
	}
	return scheme
}

// validateName returns true if name is non-blank. On failure it writes a 400
// error response and returns false, so callers can simply return.
func validateName(ctx context.Context, w http.ResponseWriter, name string) bool {
	if strings.TrimSpace(name) == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return false
	}
	return true
}

// maxTokenNameLength is the shared maximum length for API key and Kobo token names.
const maxTokenNameLength = 100

// validateTokenName trims whitespace from name, then validates that it is
// non-empty and does not exceed maxTokenNameLength. It returns the trimmed name
// and true on success. On failure it writes the appropriate 400 error response
// and returns "", false so callers can simply return.
func validateTokenName(ctx context.Context, w http.ResponseWriter, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return "", false
	}
	if len(name) > maxTokenNameLength {
		writeError(ctx, w, http.StatusBadRequest, fmt.Sprintf("name must be at most %d characters", maxTokenNameLength))
		return "", false
	}
	return name, true
}

// generateRandomHex generates n random bytes and returns them as a lowercase
// hex-encoded string. It wraps crypto/rand.Read and returns an error if the
// random source fails.
func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// writeSecretTokenResponse sets cache-prevention headers and writes a JSON
// response. It should be used whenever the response body contains a plaintext
// secret token or key that should not be stored in HTTP caches. Note that
// these headers cannot fully prevent storage in browser history or other
// user-controlled storage.
func writeSecretTokenResponse(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(ctx, w, status, data)
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
// It is a low-level slice transformer; for complete list endpoints (fetch → log → respond),
// prefer listEntities instead.
func mapSlice[T any, DTO any](items []T, toDTO func(*T) DTO) []DTO {
	dtos := make([]DTO, 0, len(items))
	for i := range items {
		dtos = append(dtos, toDTO(&items[i]))
	}
	return dtos
}

// listUserEntities is like listEntities but filters by the authenticated user ID.
// It calls list(ctx, userID), converts entities to DTOs, and writes the JSON
// response. It always writes a JSON array (never null), matching the behavior of
// listEntities.
func listUserEntities[T any, DTO any](
	w http.ResponseWriter,
	r *http.Request,
	resource string,
	list func(context.Context, string) ([]T, error),
	toDTO func(*T) DTO,
) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	slog.DebugContext(ctx, "listing "+resource)

	entities, err := list(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list "+resource, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+resource)
		return
	}

	slog.DebugContext(ctx, resource+" listed", slog.Int(otelkeys.Count, len(entities)))

	writeJSON(ctx, w, http.StatusOK, mapSlice(entities, toDTO))
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

// deleteResourceCore is the shared implementation of the
// fetch-then-delete-then-audit pattern. get and del are pre-bound closures
// that already capture the resource ID (and, for user-owned resources, the
// user ID). resource is the human-readable display name used in error
// messages; auditEntityType is the stable snake_case identifier written to
// the audit log (for non-user-owned resources these two values are the same).
func deleteResourceCore[T any](
	d *db.DB,
	w http.ResponseWriter,
	r *http.Request,
	id string,
	resource string,
	auditEntityType string,
	idKey string,
	get func(context.Context) (T, error),
	del func(context.Context) error,
	auditAction string,
	auditMeta func(T) map[string]any,
) {
	ctx := r.Context()
	slog.DebugContext(ctx, "deleting "+resource, slog.String(idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers

	entity, err := get(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to get "+resource, slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	if err := del(ctx); err != nil {
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
	if err := d.CreateAuditLog(ctx, userID, auditAction, auditEntityType, id, meta); err != nil {
		slog.WarnContext(
			ctx,
			"failed to write audit log",
			slog.String(otelkeys.Resource, resource),
			slog.String(otelkeys.EntityType, auditEntityType),
			slog.String(idKey, id), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
			slog.Any(otelkeys.Error, err),
		)
	}

	w.WriteHeader(http.StatusNoContent)
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
	deleteResourceCore(d, w, r, id, resource, resource, idKey,
		func(ctx context.Context) (T, error) { return get(ctx, id) },
		func(ctx context.Context) error { return del(ctx, id) },
		auditAction, auditMeta,
	)
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
	userID := auth.UserIDFromContext(r.Context())
	deleteResourceCore(d, w, r, id, resource, auditEntityType, idKey,
		func(ctx context.Context) (T, error) { return get(ctx, id, userID) },
		func(ctx context.Context) error { return del(ctx, id, userID) },
		auditAction, auditMeta,
	)
}

// isNilValue reports whether v, when passed as any, wraps a nil pointer.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

const maxTokenSizeBytes = 64 * 1024

// generateBase64Token generates a cryptographically random token of n bytes
// and returns it as a Base64-encoded string.
func generateBase64Token(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("generate token: n must be positive")
	}
	if n > maxTokenSizeBytes {
		return "", fmt.Errorf("generate token: n too large (max %d bytes)", maxTokenSizeBytes)
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
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
