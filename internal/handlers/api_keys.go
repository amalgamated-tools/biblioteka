package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const apiKeyDisplayPrefixHexLen = 12

// APIKeyHandler holds dependencies for API key endpoints.
type APIKeyHandler struct {
	DB *db.DB
}

type apiKeyCreateRequest struct {
	Name string `json:"name"`
}

type apiKeyDTO struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	KeyPrefix  string        `json:"key_prefix"`
	LastUsedAt *db.Timestamp `json:"last_used_at"`
	CreatedAt  db.Timestamp  `json:"created_at"`
}

type apiKeyCreateResponse struct {
	apiKeyDTO
	Key string `json:"key"`
}

func toAPIKeyDTO(k *db.APIKey) apiKeyDTO {
	return apiKeyDTO{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
}

const maxAPIKeyNameLength = 100

// HandleAPIKeys handles GET /api/api-keys and POST /api/api-keys.
//
// @Summary List and create API keys
// @Description
// @Description List all API keys for the authenticated user with GET /api/api-keys.
// @Description Create a new API key for the authenticated user with POST /api/api-keys.
// @Tags api-keys
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} apiKeyDTO "List API keys"
// @Success 201 {object} apiKeyCreateResponse "API key created"
// @Failure 400 {object} errorResponse "Bad request"
// @Failure 401 {object} errorResponse "Unauthorized"
// @Failure 405 {object} errorResponse "Method not allowed"
// @Failure 500 {object} errorResponse "Internal server error"
// @Router /api-keys [get]
// @Router /api-keys [post]
func (h *APIKeyHandler) HandleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAPIKeys(w, r)
	case http.MethodPost:
		h.createAPIKey(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleAPIKey handles DELETE /api/api-keys/{id}.
//
// @Summary Delete an API key
// @Description Delete a specific API key owned by the authenticated user.
// @Tags api-keys
// @Security BearerAuth
// @Param id path string true "API key ID"
// @Success 204 "API key deleted"
// @Failure 400 {object} errorResponse "Invalid API key ID"
// @Failure 401 {object} errorResponse "Unauthorized"
// @Failure 404 {object} errorResponse "API key not found"
// @Failure 405 {object} errorResponse "Method not allowed"
// @Failure 500 {object} errorResponse "Internal server error"
// @Router /api-keys/{id} [delete]
func (h *APIKeyHandler) HandleAPIKey(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/api-keys/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid API key ID")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.deleteAPIKey(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *APIKeyHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	keys, err := h.DB.ListAPIKeys(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list API keys", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list API keys")
		return
	}

	dtos := make([]apiKeyDTO, 0, len(keys))
	for i := range keys {
		dtos = append(dtos, toAPIKeyDTO(&keys[i]))
	}

	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

func (h *APIKeyHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req apiKeyCreateRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}

	if len(name) > maxAPIKeyNameLength {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Sprintf("name must be at most %d characters", maxAPIKeyNameLength))
		return
	}

	// Generate 32 random hex characters (16 bytes).
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		slog.ErrorContext(r.Context(), "failed to generate random bytes", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to generate API key")
		return
	}
	fullKey := auth.APIKeyPrefix + hex.EncodeToString(randomBytes)

	// Hash the full key with SHA-256. This is appropriate because API keys are
	// high-entropy random tokens (128 bits), not user-chosen passwords. Expensive
	// hashing (bcrypt/argon2) is unnecessary for cryptographically random secrets.
	keyHash := auth.HashAPIKey(fullKey)

	// Store a longer display prefix: static prefix + the first apiKeyDisplayPrefixHexLen hex characters.
	hexPart := fullKey[len(auth.APIKeyPrefix):]
	keyPrefix := auth.APIKeyPrefix + hexPart[:apiKeyDisplayPrefixHexLen]

	userID := auth.UserIDFromContext(r.Context())
	apiKey, err := h.DB.CreateAPIKey(r.Context(), userID, name, keyHash, keyPrefix)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create API key", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionAPIKeyCreated, "api_key", apiKey.ID, map[string]any{"name": name})

	resp := apiKeyCreateResponse{
		apiKeyDTO: toAPIKeyDTO(apiKey),
		Key:       fullKey,
	}

	// Prevent caching of the response that contains the full API key.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	writeJSON(r.Context(), w, http.StatusCreated, resp)
}

func (h *APIKeyHandler) deleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.UserIDFromContext(r.Context())

	// Fetch the key first for audit log metadata.
	apiKey, err := h.DB.GetAPIKey(r.Context(), id, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "API key not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch API key", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	if err := h.DB.DeleteAPIKey(r.Context(), id, userID); err != nil {
		if err == sql.ErrNoRows {
			writeError(r.Context(), w, http.StatusNotFound, "API key not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete API key", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionAPIKeyDeleted, "api_key", id, map[string]any{"name": apiKey.Name})

	w.WriteHeader(http.StatusNoContent)
}
