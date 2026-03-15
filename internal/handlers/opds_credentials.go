package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// OPDSCredentialHandler manages OPDS credentials via the JSON API.
type OPDSCredentialHandler struct {
	DB *db.DB
}

type opdsCredentialResponse struct {
	Username  string       `json:"username"`
	CreatedAt db.Timestamp `json:"created_at"`
	UpdatedAt db.Timestamp `json:"updated_at"`
}

type opdsCredentialRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleOPDSCredentials dispatches GET/PUT/DELETE for /api/opds/credentials.
func (h *OPDSCredentialHandler) HandleOPDSCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getCredentials(w, r)
	case http.MethodPut:
		h.upsertCredentials(w, r)
	case http.MethodDelete:
		h.deleteCredentials(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *OPDSCredentialHandler) getCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	cred, err := h.DB.GetOPDSCredentialByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "no OPDS credentials configured")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get OPDS credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to get credentials")
		return
	}

	writeJSON(ctx, w, http.StatusOK, opdsCredentialResponse{
		Username:  cred.Username,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	})
}

func (h *OPDSCredentialHandler) upsertCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	var req opdsCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(ctx, w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Username = strings.ToLower(req.Username)
	if req.Username == "" {
		writeError(ctx, w, http.StatusBadRequest, "username is required")
		return
	}

	if msg := validatePassword(req.Password); msg != "" {
		writeError(ctx, w, http.StatusBadRequest, msg)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed to hash password", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create credentials")
		return
	}

	cred, err := h.DB.UpsertOPDSCredential(ctx, userID, req.Username, string(hash))
	if errors.Is(err, db.ErrOPDSUsernameExists) {
		writeError(ctx, w, http.StatusConflict, "username already taken")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert OPDS credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to create credentials")
		return
	}

	_ = h.DB.CreateAuditLog(ctx, userID, db.AuditActionOPDSCredentialUpdated, "opds_credential", cred.ID, map[string]any{
		"username": cred.Username,
	})

	writeJSON(ctx, w, http.StatusOK, opdsCredentialResponse{
		Username:  cred.Username,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	})
}

func (h *OPDSCredentialHandler) deleteCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	// Get the credential ID for audit logging before deleting.
	cred, err := h.DB.GetOPDSCredentialByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(ctx, w, http.StatusNotFound, "no OPDS credentials configured")
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get OPDS credentials for deletion", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete credentials")
		return
	}

	if err := h.DB.DeleteOPDSCredential(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "failed to delete OPDS credentials", slog.Any(otelkeys.Error, err))
		writeError(ctx, w, http.StatusInternalServerError, "failed to delete credentials")
		return
	}

	_ = h.DB.CreateAuditLog(ctx, userID, db.AuditActionOPDSCredentialDeleted, "opds_credential", cred.ID, map[string]any{
		"username": cred.Username,
	})

	w.WriteHeader(http.StatusNoContent)
}
