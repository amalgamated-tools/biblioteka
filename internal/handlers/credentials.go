package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// maxUsernameLen is the maximum allowed length for a credential username.
const maxUsernameLen = 256

// credentialResponse is the JSON representation of a protocol credential
// returned by the Biblioteka management API.
type credentialResponse struct {
	Username  string       `json:"username"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

// credentialRequest is the body for creating/updating protocol credentials
// via the Biblioteka management API.
type credentialRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// credentialEntity holds the common fields shared by all credential types,
// abstracting over db.KOSyncCredential and db.OPDSCredential.
type credentialEntity struct {
	ID        string
	Username  string
	CreatedAt db.Timestamp
	UpdatedAt db.Timestamp
}

// credentialOps captures the protocol-specific details for credential
// management. Both KOSync and OPDS implement the same get/upsert/delete
// lifecycle; only the DB methods, audit constants, and (optionally) a
// pre-hash key-derivation step differ.
type credentialOps struct {
	db              *db.DB
	protocol        string // "KOSync" or "OPDS"
	auditEntityType string // "kosync_credential" or "opds_credential"
	auditUpsert     string
	auditDelete     string
	errConflict     error

	getByUserID func(context.Context, string) (credentialEntity, error)
	upsert      func(context.Context, string, string, string) (credentialEntity, error)
	del         func(context.Context, string) error

	// deriveKey transforms the plaintext password before bcrypt hashing.
	// If nil, the password is hashed directly.
	deriveKey func(string) string
}

// toCredentialEntity converts a db.ProtocolCredential to a credentialEntity.
func toCredentialEntity(c *db.ProtocolCredential) credentialEntity {
	return credentialEntity{
		ID:        c.ID,
		Username:  c.Username,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// credentialGetAdapter wraps a DB get-by-user-ID function to return a credentialEntity.
func credentialGetAdapter(fn func(context.Context, string) (*db.ProtocolCredential, error)) func(context.Context, string) (credentialEntity, error) {
	return func(ctx context.Context, userID string) (credentialEntity, error) {
		c, err := fn(ctx, userID)
		if err != nil {
			return credentialEntity{}, err
		}
		return toCredentialEntity(c), nil
	}
}

// credentialUpsertAdapter wraps a DB upsert function to return a credentialEntity.
func credentialUpsertAdapter(fn func(context.Context, string, string, string) (*db.ProtocolCredential, error)) func(context.Context, string, string, string) (credentialEntity, error) {
	return func(ctx context.Context, userID, username, hash string) (credentialEntity, error) {
		c, err := fn(ctx, userID, username, hash)
		if err != nil {
			return credentialEntity{}, err
		}
		return toCredentialEntity(c), nil
	}
}

// handleCredentials dispatches GET/PUT/DELETE for a credential endpoint.
func handleCredentials(ops credentialOps, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getCredential(ops, w, r)
	case http.MethodPut:
		upsertCredential(ops, w, r)
	case http.MethodDelete:
		deleteCredential(ops, w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func getCredential(ops credentialOps, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	cred, err := ops.getByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "no "+ops.protocol+" credentials configured")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get "+ops.protocol+" credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to get credentials")
		return
	}

	writeJSON(ctx, w, http.StatusOK, credentialResponse{
		Username:  cred.Username,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	})
}

func upsertCredential(ops credentialOps, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	var req credentialRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	req.Username = auth.NormalizeUsername(req.Username)
	if req.Username == "" {
		writeError(ctx, w, http.StatusBadRequest, "username is required")
		return
	}
	if len(req.Username) > maxUsernameLen {
		writeError(ctx, w, http.StatusBadRequest, "username too long")
		return
	}

	if msg := validatePassword(req.Password); msg != "" {
		writeError(ctx, w, http.StatusBadRequest, msg)
		return
	}

	toHash := req.Password
	if ops.deriveKey != nil {
		toHash = ops.deriveKey(req.Password)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(toHash), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed to hash "+ops.protocol+" password", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create credentials")
		return
	}

	cred, err := ops.upsert(ctx, userID, req.Username, string(hash))
	if ops.errConflict != nil && errors.Is(err, ops.errConflict) {
		writeError(ctx, w, http.StatusConflict, "username already taken")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert "+ops.protocol+" credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create credentials")
		return
	}

	logAudit(ctx, ops.db, userID, ops.auditUpsert, ops.auditEntityType, cred.ID, map[string]any{
		"username": cred.Username,
	})

	writeJSON(ctx, w, http.StatusOK, credentialResponse{
		Username:  cred.Username,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	})
}

func deleteCredential(ops credentialOps, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	// Fetch first so we can reference the credential ID in the audit log.
	cred, err := ops.getByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "no "+ops.protocol+" credentials configured")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get "+ops.protocol+" credentials for deletion", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete credentials")
		return
	}

	if err := ops.del(ctx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(ctx, w, http.StatusNotFound, "no "+ops.protocol+" credentials configured")
			return
		}
		slog.ErrorContext(ctx, "failed to delete "+ops.protocol+" credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete credentials")
		return
	}

	logAudit(ctx, ops.db, userID, ops.auditDelete, ops.auditEntityType, cred.ID, map[string]any{
		"username": cred.Username,
	})

	w.WriteHeader(http.StatusNoContent)
}
