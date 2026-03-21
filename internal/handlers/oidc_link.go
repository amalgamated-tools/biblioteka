package handlers

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/oauth2"
)

// CreateLinkNonce godoc
//
// @Summary		Create OIDC link nonce
// @Description	Generate a short-lived nonce for linking an OIDC account
// @Tags		OIDC
// @Produce		json
// @Security	BearerAuth
// @Failure		401	{object}	errorResponse
// @Success		200	{object}	object{nonce=string}
// @Failure		500	{object}	errorResponse
// @Router		/auth/oidc/link-nonce [post]
func (h *OIDCHandler) CreateLinkNonce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "creating OIDC link nonce", slog.String(otelkeys.UserID, userID))

	nonce, err := generateState() // reuse the same 32-byte random generator
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate link nonce", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to generate nonce")
		return
	}

	h.linkNoncesMu.Lock()
	now := time.Now()
	for k, v := range h.linkNonces {
		if now.After(v.ExpiresAt) {
			delete(h.linkNonces, k)
		}
	}
	h.linkNonces[nonce] = linkNonce{
		UserID:    userID,
		ExpiresAt: now.Add(oidcStateCookieTTL),
	}
	h.linkNoncesMu.Unlock()

	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"nonce": nonce})
}

// consumeLinkNonce validates and removes a nonce, returning the associated user ID.
// Returns empty string if the nonce is invalid or expired.
func (h *OIDCHandler) consumeLinkNonce(nonce string) string {
	h.linkNoncesMu.Lock()
	defer h.linkNoncesMu.Unlock()

	entry, ok := h.linkNonces[nonce]
	if !ok {
		return ""
	}
	delete(h.linkNonces, nonce)
	if time.Now().After(entry.ExpiresAt) {
		return ""
	}
	return entry.UserID
}

// Link godoc
//
// @Summary		Link OIDC account
// @Description	Redirects to OIDC provider to link account (requires nonce)
// @Tags		OIDC
// @Param		nonce	query	string	true	"Link nonce from CreateLinkNonce"
// @Success		302	"Redirect to OIDC provider"
// @Failure		400	{object}	errorResponse
// @Failure		401	{object}	errorResponse
// @Failure		409	{object}	errorResponse
// @Failure		500	{object}	errorResponse
// @Router		/auth/oidc/link [get]
func (h *OIDCHandler) Link(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	nonceStr := r.URL.Query().Get("nonce")
	if nonceStr == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing nonce")
		return
	}

	userID := h.consumeLinkNonce(nonceStr)
	if userID == "" {
		slog.DebugContext(r.Context(), "OIDC link nonce invalid or expired")
		writeError(r.Context(), w, http.StatusUnauthorized, "invalid or expired nonce")
		return
	}

	slog.DebugContext(r.Context(), "initiating OIDC link flow", slog.String(otelkeys.UserID, userID))

	existingUser, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get user for OIDC link", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusBadRequest, "user not found")
		return
	}
	if existingUser.OIDCSubject != nil {
		writeError(r.Context(), w, http.StatusConflict, "account already linked to an SSO provider")
		return
	}

	state, err := generateState()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to generate OIDC state", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to initiate linking")
		return
	}

	verifier := oauth2.GenerateVerifier()

	// Encode the link user ID into the OIDC state parameter using HMAC signing.
	// Format: <random>.<base64url(userID)>.<base64url(hmac)>
	// This is tamper-proof and does not require server-side state, so it works
	// correctly with horizontal scaling (no sticky sessions needed).
	signedState := h.signLinkState(state, userID)

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    signedState,
		Path:     "/",
		MaxAge:   int(oidcStateCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     oidcVerifierCookieName,
		Value:    verifier,
		Path:     "/",
		MaxAge:   int(oidcStateCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})

	http.Redirect(w, r, h.Config.AuthCodeURL(signedState, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// signLinkState encodes a link user ID into the OIDC state parameter.
// Format: <random>.<base64url(userID)>.<base64url(hmac-sha256(random + "|" + userID))>
func (h *OIDCHandler) signLinkState(randomState, userID string) string {
	payload := randomState + "|" + userID
	sig := h.JWT.HMACSign([]byte(payload))
	return randomState + "." +
		base64.RawURLEncoding.EncodeToString([]byte(userID)) + "." +
		base64.RawURLEncoding.EncodeToString(sig)
}

// parseLinkState extracts and verifies a link user ID from a signed OIDC state.
// Returns empty string if the state is a normal login (no link) or the signature is invalid.
func (h *OIDCHandler) parseLinkState(state string) string {
	parts := strings.SplitN(state, ".", 3)
	if len(parts) != 3 {
		return ""
	}

	randomState := parts[0]
	userIDBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ""
	}

	userID := string(userIDBytes)
	payload := randomState + "|" + userID
	if !h.JWT.HMACVerify([]byte(payload), sig) {
		return ""
	}
	return userID
}
