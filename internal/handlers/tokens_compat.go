// tokens_compat.go provides the token creation helpers used by Kobo token endpoints.
// API key creation has moved to goauth; this file keeps the shared pattern for Kobo.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// tokenError wraps an error with a client-facing message.
type tokenError struct {
	err     error
	message string
}

func (e *tokenError) Error() string { return e.err.Error() }
func (e *tokenError) Unwrap() error { return e.err }

// tokenOps captures the token-type-specific details for token create handlers.
type tokenOps struct {
	db              *db.DB
	resource        string
	auditEntityType string
	auditCreate     string
	create          func(ctx context.Context, userID, name string) (entityID string, response any, err error)
}

// handleTokenCreate implements the common token creation flow.
func handleTokenCreate(ops tokenOps, w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(r, w, &req) {
		return
	}

	name, ok := validateTokenName(r.Context(), w, req.Name)
	if !ok {
		return
	}

	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)
	entityID, resp, err := ops.create(ctx, userID, name)
	if err != nil {
		msg := "failed to create " + ops.resource
		var te *tokenError
		if errors.As(err, &te) {
			msg = te.message
		}
		slog.ErrorContext(ctx, "failed to create token",
			slog.String(otelkeys.Resource, ops.resource),
			slog.Any(otelkeys.Error, err),
		)
		writeError(ctx, w, http.StatusInternalServerError, msg)
		return
	}

	logAudit(ctx, ops.db, userID, ops.auditCreate, ops.auditEntityType, entityID, map[string]any{"name": name})
	writeSecretTokenResponse(ctx, w, http.StatusCreated, resp)
}
