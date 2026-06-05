package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

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
	slog.DebugContext(ctx, "listing entities", slog.String(otelkeys.Resource, resource))

	entities, err := list(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list entities",
			slog.String(otelkeys.Resource, resource),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+resource)
		return
	}

	slog.DebugContext(ctx, "entities listed",
		slog.String(otelkeys.Resource, resource),
		slog.Int(otelkeys.Count, len(entities)),
	)

	writeJSON(ctx, w, http.StatusOK, mapSlice(entities, toDTO))
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

	slog.DebugContext(ctx, "listing entities", slog.String(otelkeys.Resource, resource))

	entities, err := list(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list entities",
			slog.String(otelkeys.Resource, resource),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+resource)
		return
	}

	slog.DebugContext(ctx, "entities listed",
		slog.String(otelkeys.Resource, resource),
		slog.Int(otelkeys.Count, len(entities)),
	)

	writeJSON(ctx, w, http.StatusOK, mapSlice(entities, toDTO))
}

// listPaginatedEntities is a generic helper for paginated list endpoints. It
// parses limit/offset query params, calls the paginated list function, converts
// entities to DTOs, and writes the JSON response.
func listPaginatedEntities[T any, DTO any, ListDTO any](
	w http.ResponseWriter,
	r *http.Request,
	resource string,
	list func(context.Context, int, int) ([]T, int, error),
	toDTO func(*T) DTO,
	toListDTO func([]DTO, int, int, int) ListDTO,
) {
	ctx := r.Context()
	limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)

	slog.DebugContext(ctx, "listing entities", slog.String(otelkeys.Resource, resource))

	entities, total, err := list(ctx, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list entities",
			slog.String(otelkeys.Resource, resource),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to list "+resource)
		return
	}

	slog.DebugContext(ctx, "entities listed",
		slog.String(otelkeys.Resource, resource),
		slog.Int(otelkeys.Count, len(entities)),
	)

	writeJSON(ctx, w, http.StatusOK, toListDTO(mapSlice(entities, toDTO), total, limit, offset))
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
	userID string,
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
	slog.DebugContext(ctx, "deleting resource", slog.String(otelkeys.Resource, resource), slog.String(idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers

	entity, err := get(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to get resource", slog.String(otelkeys.Resource, resource), slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete "+resource)
		return
	}

	if err := del(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, resource+" not found")
			return
		}
		slog.ErrorContext(ctx, "failed to delete resource", slog.String(otelkeys.Resource, resource), slog.String(idKey, id), slog.Any(otelkeys.Error, err)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
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
	auditEntityType string,
	idKey string,
	get func(context.Context, string) (T, error),
	del func(context.Context, string) error,
	auditAction string,
	auditMeta func(T) map[string]any,
) {
	if !requireAdmin(d, w, r) {
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	deleteResourceCore(d, w, r, userID, id, resource, auditEntityType, idKey,
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
	deleteResourceCore(d, w, r, userID, id, resource, auditEntityType, idKey,
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
