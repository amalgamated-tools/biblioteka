package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

type koboTokenCreateRequest struct {
	Name string `json:"name"`
}

type koboTokenDTO struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Name      string       `json:"name"`
	TokenHash string       `json:"token_hash"`
	CreatedAt db.Timestamp `json:"created_at"`
}

type koboTokenCreateResponse struct {
	koboTokenDTO
	Token string `json:"token"`
}

func toKoboTokenDTO(token *db.KoboToken) koboTokenDTO {
	return koboTokenDTO{
		ID:        token.ID,
		UserID:    token.UserID,
		Name:      token.Name,
		TokenHash: token.TokenHash,
		CreatedAt: token.CreatedAt,
	}
}

// HandleKoboTokens handles GET/POST /api/kobo/tokens.
func (h *KoboHandler) HandleKoboTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listKoboTokens(w, r)
	case http.MethodPost:
		h.createKoboToken(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleKoboToken handles DELETE /api/kobo/tokens/{id}.
func (h *KoboHandler) HandleKoboToken(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/kobo/tokens/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid token ID")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		h.deleteKoboToken(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *KoboHandler) listKoboTokens(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	tokens, err := h.DB.ListKoboTokens(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list kobo tokens", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list Kobo tokens")
		return
	}
	if tokens == nil {
		tokens = []db.KoboToken{}
	}

	dtos := make([]koboTokenDTO, 0, len(tokens))
	for i := range tokens {
		dtos = append(dtos, toKoboTokenDTO(&tokens[i]))
	}
	writeJSON(r.Context(), w, http.StatusOK, dtos)
}

func (h *KoboHandler) createKoboToken(w http.ResponseWriter, r *http.Request) {
	var req koboTokenCreateRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		writeError(r.Context(), w, http.StatusBadRequest, "name must be at most 100 characters")
		return
	}

	// Generate a random 32-byte hex token (64 hex chars).
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		slog.ErrorContext(r.Context(), "failed to generate random bytes", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to generate Kobo token")
		return
	}
	token := hex.EncodeToString(raw)
	tokenHash := auth.HashKoboToken(token)

	userID := auth.UserIDFromContext(r.Context())
	koboToken, err := h.DB.CreateKoboToken(r.Context(), userID, name, tokenHash)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create Kobo token")
		return
	}

	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionKoboTokenCreated, "kobo_token", koboToken.ID, map[string]any{"name": name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	resp := koboTokenCreateResponse{
		koboTokenDTO: toKoboTokenDTO(koboToken),
		Token:        token,
	}
	writeJSON(r.Context(), w, http.StatusCreated, resp)
}

func (h *KoboHandler) deleteKoboToken(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.UserIDFromContext(r.Context())

	token, err := h.DB.GetKoboToken(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "Kobo token not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete Kobo token")
		return
	}

	if err := h.DB.DeleteKoboToken(r.Context(), id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "Kobo token not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete Kobo token")
		return
	}

	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionKoboTokenDeleted, "kobo_token", id, map[string]any{"name": token.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}
