package handlers

import (
	"log/slog"
	"net/http"

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

	writeJSON(r.Context(), w, http.StatusOK, mapSlice(tokens, toKoboTokenDTO))
}

func (h *KoboHandler) createKoboToken(w http.ResponseWriter, r *http.Request) {
	var req koboTokenCreateRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	name, ok := validateTokenName(r.Context(), w, req.Name)
	if !ok {
		return
	}

	// Generate a random 32-byte hex token (64 hex chars).
	token, err := generateRandomHex(32)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate random bytes", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to generate Kobo token")
		return
	}
	tokenHash := auth.HashKoboToken(token)

	userID := auth.UserIDFromContext(r.Context())
	koboToken, err := h.DB.CreateKoboToken(r.Context(), userID, name, tokenHash)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create Kobo token")
		return
	}

	logAudit(r.Context(), h.DB, userID, db.AuditActionKoboTokenCreated, "kobo_token", koboToken.ID, map[string]any{"name": name})

	resp := koboTokenCreateResponse{
		koboTokenDTO: toKoboTokenDTO(koboToken),
		Token:        token,
	}
	writeSecretTokenResponse(r.Context(), w, http.StatusCreated, resp)
}

func (h *KoboHandler) deleteKoboToken(w http.ResponseWriter, r *http.Request, id string) {
	deleteUserOwnedResource(h.DB, w, r, id, "Kobo token", "kobo_token", otelkeys.KoboTokenID,
		h.DB.GetKoboToken, h.DB.DeleteKoboToken,
		db.AuditActionKoboTokenDeleted,
		func(t *db.KoboToken) map[string]any { return map[string]any{"name": t.Name} },
	)
}
