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

// tokenError wraps an error with a client-facing message so that
// handleTokenCreate can surface distinct HTTP responses for different failure
// modes (e.g. token generation vs database persistence).
type tokenError struct {
	err     error
	message string
}

// Error implements the error interface by delegating to the wrapped error.
func (e *tokenError) Error() string { return e.err.Error() }

// Unwrap returns the underlying error so that errors.Is and errors.As can
// inspect the error chain.
func (e *tokenError) Unwrap() error { return e.err }

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
//
// If the create closure returns a *tokenError, its message is used as the
// client-facing HTTP error and the log message. Otherwise a generic
// "failed to create <resource>" message is used for both.
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
		msg := "failed to create " + ops.resource
		var te *tokenError
		if errors.As(err, &te) {
			msg = te.message
		}
		slog.ErrorContext(ctx, msg, slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, msg)
		return
	}

	logAudit(ctx, ops.db, userID, ops.auditCreate, ops.auditEntityType, entityID, map[string]any{"name": name})
	writeSecretTokenResponse(ctx, w, http.StatusCreated, resp)
}
