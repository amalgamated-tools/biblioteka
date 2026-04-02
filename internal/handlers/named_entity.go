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
// enables the generic createNamedEntity, getNamedEntity, updateNamedEntity,
// handleNamedEntityCollection, and handleNamedEntitySingle helpers to share the
// common handler flow while remaining entity-agnostic.
type namedEntityOps[T any, DTO any, Req any] struct {
	db             *db.DB
	entityLabel    string // human-readable label, e.g. "author" or "series"
	entityArticle  string // with indefinite article, e.g. "an author" or "a series"
	idKey          string // otelkeys constant for the entity ID field
	errInvalidName error
	errNameExists  error
	auditCreate    string
	auditUpdate    string
	auditDelete    string

	// pathPrefix is the API path prefix used to extract the entity ID from the
	// URL, e.g. "/api/authors/". Required when using handleNamedEntitySingle.
	pathPrefix string
	// collectionLabel is the plural resource label used for list logging,
	// e.g. "authors" or "series". Required when using handleNamedEntityCollection.
	collectionLabel string

	get    func(context.Context, string) (*T, error)
	create func(context.Context, Req) (*T, error)
	update func(context.Context, string, Req) (*T, error)
	list   func(context.Context) ([]T, error)
	del    func(context.Context, string) error

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
	ctx := r.Context()
	var req Req
	if !decodeJSON(r, w, &req) {
		return
	}

	name := ops.reqName(req)
	if !validateName(ctx, w, name) {
		return
	}

	slog.DebugContext(ctx, "creating "+ops.entityLabel, slog.String(otelkeys.Name, name))

	entity, err := ops.create(ctx, req)
	if err != nil {
		if handleNameErr(ctx, w, err, ops.errInvalidName, ops.errNameExists, ops.entityArticle) {
			return
		}
		slog.ErrorContext(ctx, "failed to create "+ops.entityLabel, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}
	if entity == nil {
		slog.ErrorContext(ctx, "create returned nil entity without error")
		writeError(ctx, w, http.StatusInternalServerError, "failed to create "+ops.entityLabel)
		return
	}

	slog.DebugContext(ctx, ops.entityLabel+" created",
		slog.String(ops.idKey, ops.entityID(entity)), //nolint:sloglint // idKey is always an otelkeys constant passed by callers
		slog.String(otelkeys.Name, ops.entityName(entity)),
	)

	userID := auth.UserIDFromContext(ctx)
	logAudit(ctx, ops.db, userID, ops.auditCreate, ops.entityLabel, ops.entityID(entity), map[string]any{"name": ops.entityName(entity)})

	writeJSON(ctx, w, http.StatusCreated, ops.toDTO(entity))
}

// getNamedEntity implements the common get-by-ID flow for named entities:
// call get → handle errors → respond.
func getNamedEntity[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	slog.DebugContext(ctx, "fetching "+ops.entityLabel,
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

	slog.DebugContext(ctx, "updating "+ops.entityLabel,
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
	logAudit(ctx, ops.db, userID, ops.auditUpdate, ops.entityLabel, ops.entityID(entity), map[string]any{"name": ops.entityName(entity)})

	writeJSON(ctx, w, http.StatusOK, ops.toDTO(entity))
}

// handleNamedEntityCollection implements the collection-level routing for named
// entities: GET → list all, POST → create. It dispatches to the appropriate
// helper based on HTTP method.
func handleNamedEntityCollection[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listEntities(w, r, ops.collectionLabel, ops.list, ops.toDTO)
	case http.MethodPost:
		createNamedEntity(ops, w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleNamedEntitySingle implements the single-entity routing for named
// entities: GET → get by ID, PUT → update, DELETE → delete. It extracts the
// entity ID from ops.pathPrefix and dispatches to the appropriate helper.
func handleNamedEntitySingle[T, DTO, Req any](ops namedEntityOps[T, DTO, Req], w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, ops.pathPrefix)
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid "+ops.entityLabel+" ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		getNamedEntity(ops, w, r, id)
	case http.MethodPut:
		updateNamedEntity(ops, w, r, id)
	case http.MethodDelete:
		deleteResource(ops.db, w, r, id, ops.entityLabel, ops.idKey,
			ops.get, ops.del,
			ops.auditDelete,
			func(e *T) map[string]any { return map[string]any{"name": ops.entityName(e)} },
		)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
