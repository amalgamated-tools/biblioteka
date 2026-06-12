package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// namedEntityOps captures the entity-specific operations for named-entity CRUD
// handlers (e.g. authors and series). It is modeled after credentialOps and
// enables the generic createNamedEntity, getNamedEntity, and updateNamedEntity
// helpers to share the common handler flow while remaining entity-agnostic.
type namedEntityOps[T any, DTO any, Req any] struct {
	db              *db.DB
	entityLabel     string // human-readable label, e.g. "author" or "series"
	auditEntityType string // stable type written to audit logs (snake_case preferred); defaults to entityLabel when blank
	entityArticle   string // with indefinite article, e.g. "an author" or "a series"
	idKey           string // otelkeys constant for the entity ID field
	errInvalidName  error
	errNameExists   error
	auditCreate     string
	auditUpdate     string

	get    func(context.Context, string) (*T, error)
	create func(context.Context, Req) (*T, error)
	update func(context.Context, string, Req) (*T, error)

	// reqName extracts the name field from a decoded request body.
	reqName func(Req) string
	// entityName extracts the name from a stored entity (used for audit metadata
	// and post-write debug logging).
	entityName func(*T) string
	// entityID extracts the ID from a stored entity (used for audit and debug
	// logging).
	entityID func(*T) string

	toDTO func(*T) DTO
}

// resolvedAuditEntityType returns the configured audit entity type when one is
// provided, otherwise it falls back to entityLabel for backward compatibility
// with admin-owned entities that historically reused their display label in
// audit logs.
func (ops namedEntityOps[T, DTO, Req]) resolvedAuditEntityType() string {
	if ops.auditEntityType != "" {
		return ops.auditEntityType
	}

	return ops.entityLabel
}

// createNamedEntity implements the common create flow for named entities:
// decode request → validate name → call create → handle errors → audit → respond.
func createNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}

	name := ops.reqName(req)
	if !validateName(ctx, w, name) {
		return
	}

	slog.DebugContext(ctx, "creating entity",
		slog.String(otelkeys.EntityType, ops.entityLabel),
		slog.String(otelkeys.Name, name),
	)

	entity, err := ops.create(ctx, req)
	if err != nil {
		if handleNameErr(ctx, w, err, ops.errInvalidName, ops.errNameExists, ops.entityArticle) {
			return
		}
		slog.ErrorContext(ctx, "failed to create entity",
			slog.String(otelkeys.EntityType, ops.entityLabel),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}
	if entity == nil {
		slog.ErrorContext(ctx, "create returned nil entity without error",
			slog.String(otelkeys.EntityType, ops.entityLabel),
		)
		writeError(ctx, w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}

	slog.DebugContext(ctx, "entity created",
		slog.String(otelkeys.EntityType, ops.entityLabel),
		slog.String(ops.idKey, ops.entityID(entity)), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		slog.String(otelkeys.Name, ops.entityName(entity)),
	)

	userID := auth.UserIDFromContext(ctx)
	logAudit(ctx, ops.db, userID, ops.auditCreate, ops.resolvedAuditEntityType(), ops.entityID(entity), map[string]any{otelkeys.Name: ops.entityName(entity)})

	writeJSON(ctx, w, http.StatusCreated, ops.toDTO(entity))
}

// getNamedEntity implements the common get-by-ID flow for named entities:
// call get → handle errors → respond.
func getNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	slog.DebugContext(ctx, "fetching entity",
		slog.String(otelkeys.EntityType, ops.entityLabel),
		slog.String(ops.idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
	entity, err := ops.get(ctx, id)
	if handleDBErr(ctx, w, err, ops.entityLabel) {
		return
	}
	if entity == nil {
		slog.ErrorContext(ctx, "get returned nil entity without error", slog.String(ops.idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to fetch "+ops.entityLabel)
		return
	}
	writeJSON(ctx, w, http.StatusOK, ops.toDTO(entity))
}

// updateNamedEntity implements the common update flow for named entities:
// decode request → validate name → call update → handle errors → audit → respond.
func updateNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}

	name := ops.reqName(req)
	if !validateName(ctx, w, name) {
		return
	}

	slog.DebugContext(ctx, "updating entity",
		slog.String(otelkeys.EntityType, ops.entityLabel),
		slog.String(ops.idKey, id), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		slog.String(otelkeys.Name, name),
	)

	entity, err := ops.update(ctx, id, req)
	if handleUpdateErr(ctx, w, err, ops.errInvalidName, ops.errNameExists, ops.entityArticle, ops.entityLabel, id) {
		return
	}
	if entity == nil {
		slog.ErrorContext(ctx, "update returned nil entity without error", slog.String(ops.idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		writeError(ctx, w, http.StatusInternalServerError, "failed to update "+ops.entityLabel)
		return
	}

	userID := auth.UserIDFromContext(ctx)
	logAudit(ctx, ops.db, userID, ops.auditUpdate, ops.resolvedAuditEntityType(), ops.entityID(entity), map[string]any{otelkeys.Name: ops.entityName(entity)})

	writeJSON(ctx, w, http.StatusOK, ops.toDTO(entity))
}
