package handlers

import (
	"context"
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

// HandleKoboTokens handles GET /api/kobo/tokens and POST /api/kobo/tokens.
//
//	@Summary	List and create Kobo sync tokens
//	@Description
//	@Description	GET lists all Kobo sync tokens for the authenticated user.
//	@Description	POST creates a new Kobo sync token. The raw token is returned only in the creation response and is never retrievable again.
//	@Description	POST body: {"name": "string"} (required)
//	@Tags			kobo-tokens
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		koboTokenDTO			"List of Kobo tokens"
//	@Success		201		{object}	koboTokenCreateResponse	"Kobo token created"
//	@Failure		400		{object}	errorResponse			"Bad request"
//	@Failure		401		{object}	errorResponse			"Unauthorized"
//	@Failure		405		{object}	errorResponse			"Method not allowed"
//	@Failure		500		{object}	errorResponse			"Internal server error"
//	@Router			/kobo/tokens [get]
//	@Router			/kobo/tokens [post]
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
//
//	@Summary		Delete a Kobo sync token
//	@Description	Delete a Kobo sync token by ID. The device using this token will receive 401 on its next sync.
//	@Tags			kobo-tokens
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"Kobo token ID"
//	@Success		204	"Token deleted"
//	@Failure		400	{object}	errorResponse	"Bad request"
//	@Failure		401	{object}	errorResponse	"Unauthorized"
//	@Failure		404	{object}	errorResponse	"Token not found"
//	@Failure		405	{object}	errorResponse	"Method not allowed"
//	@Failure		500	{object}	errorResponse	"Internal server error"
//	@Router			/kobo/tokens/{id} [delete]
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
	listUserEntities(w, r, "Kobo tokens", h.DB.ListKoboTokens, toKoboTokenDTO)
}

func (h *KoboHandler) createKoboToken(w http.ResponseWriter, r *http.Request) {
	handleTokenCreate(tokenOps{
		db:              h.DB,
		resource:        "Kobo token",
		auditEntityType: "kobo_token",
		auditCreate:     db.AuditActionKoboTokenCreated,
		create: func(ctx context.Context, userID, name string) (string, any, error) {
			// Generate a random 32-byte hex token (64 hex chars).
			token, err := generateRandomHex(32)
			if err != nil {
				return "", nil, &tokenError{err: err, message: "failed to generate Kobo token"}
			}
			tokenHash := auth.HashKoboToken(token)

			koboToken, err := h.DB.CreateKoboToken(ctx, userID, name, tokenHash)
			if err != nil {
				return "", nil, err
			}

			return koboToken.ID, koboTokenCreateResponse{koboTokenDTO: toKoboTokenDTO(koboToken), Token: token}, nil
		},
	}, w, r)
}

func (h *KoboHandler) deleteKoboToken(w http.ResponseWriter, r *http.Request, id string) {
	deleteUserOwnedResource(h.DB, w, r, id, "Kobo token", "kobo_token", otelkeys.KoboTokenID,
		h.DB.GetKoboToken, h.DB.DeleteKoboToken,
		db.AuditActionKoboTokenDeleted,
		func(t *db.KoboToken) map[string]any { return map[string]any{"name": t.Name} },
	)
}
