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
	db             *db.DB
	entityLabel    string // human-readable label, e.g. "author" or "series"
	entityArticle  string // with indefinite article, e.g. "an author" or "a series"
	idKey          string // otelkeys constant for the entity ID field
	errInvalidName error
	errNameExists  error
	auditCreate    string
	auditUpdate    string

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

// createNamedEntity implements the common create flow for named entities:
// decode request → validate name → call create → handle errors → audit → respond.
func createNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request) {
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}

	name := ops.reqName(req)
	if !validateName(r.Context(), w, name) {
		return
	}

	slog.DebugContext(r.Context(), "creating "+ops.entityLabel, slog.String(otelkeys.Name, name))

	entity, err := ops.create(r.Context(), req)
	if err != nil {
		if handleNameErr(r.Context(), w, err, ops.errInvalidName, ops.errNameExists, ops.entityArticle) {
			return
		}
		slog.ErrorContext(r.Context(), "failed to create "+ops.entityLabel, slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}
	if entity == nil {
		slog.ErrorContext(r.Context(), "create returned nil entity without error")
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}

	slog.DebugContext(r.Context(), ops.entityLabel+" created",
		slog.String(ops.idKey, ops.entityID(entity)), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		slog.String(otelkeys.Name, ops.entityName(entity)),
	)

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), ops.db, userID, ops.auditCreate, ops.entityLabel, ops.entityID(entity), map[string]any{"name": ops.entityName(entity)})

	writeJSON(r.Context(), w, http.StatusCreated, ops.toDTO(entity))
}

// getNamedEntity implements the common get-by-ID flow for named entities:
// call get → handle errors → respond.
func getNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request, id string) {
	slog.DebugContext(r.Context(), "fetching "+ops.entityLabel,
		slog.String(ops.idKey, id)) //nolint:sloglint // idKey is always an otelkeys constant passed by callers
	entity, err := ops.get(r.Context(), id)
	if handleDBErr(r.Context(), w, err, ops.entityLabel) {
		return
	}
	if entity == nil {
		slog.ErrorContext(r.Context(), "get returned nil entity without error", slog.String(ops.idKey, id))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to fetch "+ops.entityLabel)
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, ops.toDTO(entity))
}

// updateNamedEntity implements the common update flow for named entities:
// decode request → validate name → call update → handle errors → audit → respond.
func updateNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request, id string) {
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}

	name := ops.reqName(req)
	if !validateName(r.Context(), w, name) {
		return
	}

	slog.DebugContext(r.Context(), "updating "+ops.entityLabel,
		slog.String(ops.idKey, id), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		slog.String(otelkeys.Name, name),
	)

	entity, err := ops.update(r.Context(), id, req)
	if handleUpdateErr(r.Context(), w, err, ops.errInvalidName, ops.errNameExists, ops.entityArticle, ops.entityLabel, id) {
		return
	}
	if entity == nil {
		slog.ErrorContext(r.Context(), "update returned nil entity without error", slog.String(ops.idKey, id))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update "+ops.entityLabel)
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), ops.db, userID, ops.auditUpdate, ops.entityLabel, ops.entityID(entity), map[string]any{"name": ops.entityName(entity)})

	writeJSON(r.Context(), w, http.StatusOK, ops.toDTO(entity))
}
