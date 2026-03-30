package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// tokenOps captures the token-type-specific details for token create handlers.
// Both APIKeyHandler and KoboHandler implement the same token creation lifecycle;
// only the generation logic, DB methods, and response shapes differ.
type tokenOps struct {
	db              *db.DB
	resource        string // human-readable display name, e.g. "API key"
	auditEntityType string // stable snake_case identifier for audit log, e.g. "api_key"
	auditCreate     string // audit action constant for token creation

	// create generates and persists the token for the given user and name.
	// It returns the persisted entity ID (for audit log) and the JSON response body.
	create func(ctx context.Context, userID, name string) (entityID string, response any, err error)
}

// handleTokenCreate implements the common token creation flow:
// decode name → validate → create (generate+hash+persist) → audit → respond.
func handleTokenCreate(ops tokenOps, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(r, w, &req) {
		return
	}

	ctx := r.Context()
	name, ok := validateTokenName(ctx, w, req.Name)
	if !ok {
		return
	}

	userID := auth.UserIDFromContext(ctx)
	entityID, resp, err := ops.create(ctx, userID, name)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create "+ops.resource, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create "+ops.resource)
		return
	}

	logAudit(ctx, ops.db, userID, ops.auditCreate, ops.auditEntityType, entityID, map[string]any{"name": name})
	writeSecretTokenResponse(ctx, w, http.StatusCreated, resp)
}
