package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookieName      = "oidc_state"
	oidcVerifierCookieName   = "oidc_verifier"
	oidcLinkUserIDCookieName = "oidc_link_user_id"
	oidcStateCookieTTL       = 5 * time.Minute
)

// linkNonce is a short-lived, single-use token mapping to a user ID.
type linkNonce struct {
	UserID    string
	ExpiresAt time.Time
}

// OIDCHandler holds dependencies for OIDC auth endpoints.
type OIDCHandler struct {
	DB            *db.DB
	JWT           *auth.JWTManager
	Provider      *oidc.Provider
	Config        oauth2.Config
	SecureCookies bool

	linkNonces   map[string]linkNonce
	linkNoncesMu sync.Mutex
}

// NewOIDCHandler creates a new OIDCHandler by performing OIDC discovery on the issuer URL.
func NewOIDCHandler(ctx context.Context, database *db.DB, jwt *auth.JWTManager, issuerURL, clientID, clientSecret, redirectURI string, secureCookies bool) (*OIDCHandler, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	return &OIDCHandler{
		DB:            database,
		JWT:           jwt,
		Provider:      provider,
		Config:        config,
		SecureCookies: secureCookies,
		linkNonces:    make(map[string]linkNonce),
	}, nil
}

// Login godoc
// @Summary     OIDC login
// @Description Redirects to the OIDC provider's authorization endpoint
// @Tags        OIDC
// @Success     302 "Redirect to OIDC provider"
// @Failure     404 {object} errorResponse "OIDC not configured"
// @Failure     500 {object} errorResponse
// @Router      /auth/oidc/login [get]
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slog.DebugContext(r.Context(), "initiating OIDC login flow")

	state, err := generateState()
	if err != nil {
		slog.Error("failed to generate OIDC state", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to initiate login")
		return
	}

	verifier := oauth2.GenerateVerifier()

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
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

	http.Redirect(w, r, h.Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// CreateLinkNonce godoc
// @Summary     Create OIDC link nonce
// @Description Generate a short-lived nonce for linking an OIDC account
// @Tags        OIDC
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} object{nonce=string}
// @Failure     500 {object} errorResponse
// @Router      /auth/oidc/link-nonce [post]
func (h *OIDCHandler) CreateLinkNonce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "creating OIDC link nonce", slog.String("user_id", userID))

	nonce, err := generateState() // reuse the same 32-byte random generator
	if err != nil {
		slog.Error("failed to generate link nonce", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate nonce")
		return
	}

	h.linkNoncesMu.Lock()
	// Purge expired nonces while we hold the lock
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

	writeJSON(w, http.StatusOK, map[string]string{"nonce": nonce})
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
// @Summary     Link OIDC account
// @Description Redirects to OIDC provider to link account (requires nonce)
// @Tags        OIDC
// @Param       nonce query    string true "Link nonce from CreateLinkNonce"
// @Success     302   "Redirect to OIDC provider"
// @Failure     400   {object} errorResponse
// @Failure     401   {object} errorResponse
// @Failure     409   {object} errorResponse
// @Failure     500   {object} errorResponse
// @Router      /auth/oidc/link [get]
func (h *OIDCHandler) Link(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	nonceStr := r.URL.Query().Get("nonce")
	if nonceStr == "" {
		writeError(w, http.StatusBadRequest, "missing nonce")
		return
	}

	userID := h.consumeLinkNonce(nonceStr)
	if userID == "" {
		slog.DebugContext(r.Context(), "OIDC link nonce invalid or expired")
		writeError(w, http.StatusUnauthorized, "invalid or expired nonce")
		return
	}

	slog.DebugContext(r.Context(), "initiating OIDC link flow", slog.String("user_id", userID))

	// Fail fast if the user already has an OIDC subject linked
	existingUser, err := h.DB.GetUserByID(userID)
	if err != nil {
		slog.Error("failed to get user for OIDC link", "error", err)
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}
	if existingUser.OIDCSubject != nil {
		writeError(w, http.StatusConflict, "account already linked to an SSO provider")
		return
	}

	state, err := generateState()
	if err != nil {
		slog.Error("failed to generate OIDC state", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to initiate linking")
		return
	}

	verifier := oauth2.GenerateVerifier()

	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
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
	http.SetCookie(w, &http.Cookie{
		Name:     oidcLinkUserIDCookieName,
		Value:    userID,
		Path:     "/",
		MaxAge:   int(oidcStateCookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})

	http.Redirect(w, r, h.Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

// Callback godoc
// @Summary     OIDC callback
// @Description Handles the OIDC provider's redirect after authentication
// @Tags        OIDC
// @Param       state query    string true "OIDC state parameter"
// @Param       code  query    string true "Authorization code"
// @Success     302   "Redirect to frontend with token"
// @Failure     400   {object} errorResponse
// @Failure     401   {object} errorResponse
// @Failure     500   {object} errorResponse
// @Router      /auth/oidc/callback [get]
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slog.DebugContext(r.Context(), "OIDC callback received")

	// Validate state
	cookie, err := r.Cookie(oidcStateCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusBadRequest, "missing state cookie")
		return
	}

	if r.URL.Query().Get("state") != cookie.Value {
		writeError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	// Read the PKCE verifier
	verifierCookie, err := r.Cookie(oidcVerifierCookieName)
	if err != nil || verifierCookie.Value == "" {
		writeError(w, http.StatusBadRequest, "missing PKCE verifier cookie")
		return
	}

	// Clear the state and verifier cookies
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     oidcVerifierCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})

	// Check for error response from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.Error("OIDC provider returned error", "error_code", errParam, "description", r.URL.Query().Get("error_description"))
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	// Exchange authorization code for tokens
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	oauth2Token, err := h.Config.Exchange(r.Context(), code, oauth2.VerifierOption(verifierCookie.Value))
	if err != nil {
		slog.Error("failed to exchange OIDC code", "error", err)
		writeError(w, http.StatusUnauthorized, "failed to exchange authorization code")
		return
	}

	slog.DebugContext(r.Context(), "OIDC code exchanged successfully")

	// Extract and verify the ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing id_token in response")
		return
	}

	verifier := h.Provider.Verifier(&oidc.Config{ClientID: h.Config.ClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		slog.Error("failed to verify OIDC id_token", "error", err)
		writeError(w, http.StatusUnauthorized, "invalid id_token")
		return
	}

	// Extract claims
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("failed to parse OIDC claims", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to parse user info")
		return
	}

	if claims.Sub == "" {
		writeError(w, http.StatusBadRequest, "sub claim is required")
		return
	}
	if claims.Email == "" {
		writeError(w, http.StatusBadRequest, "email claim is required")
		return
	}
	if claims.Name == "" {
		claims.Name = claims.Email
	}

	// Read and clear the link flow cookie now that OIDC validation has succeeded
	var linkUserID string
	if linkCookie, err := r.Cookie(oidcLinkUserIDCookieName); err == nil && linkCookie.Value != "" {
		linkUserID = linkCookie.Value
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcLinkUserIDCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.SecureCookies,
	})

	// Handle link flow: attach OIDC subject to an existing authenticated user
	if linkUserID != "" {
		slog.DebugContext(r.Context(), "OIDC link flow: linking account", slog.String("user_id", linkUserID), slog.String("subject", claims.Sub))
		user, err := h.DB.GetUserByID(linkUserID)
		if err != nil {
			slog.Error("failed to get user for OIDC link", "error", err)
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("User not found"), http.StatusFound)
			return
		}
		if user.OIDCSubject != nil {
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("Account is already linked to an SSO provider"), http.StatusFound)
			return
		}
		// Check if this OIDC subject is already linked to a different user
		if existing, err := h.DB.GetUserByOIDCSubject(claims.Sub); err == nil && existing.ID != linkUserID {
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("This SSO identity is already linked to another account"), http.StatusFound)
			return
		}
		if err := h.DB.LinkOIDCSubject(linkUserID, claims.Sub); err != nil {
			slog.Error("failed to link OIDC subject to user", "user_id", linkUserID, "error", err)
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("Failed to link account"), http.StatusFound)
			return
		}
		slog.DebugContext(r.Context(), "OIDC account linked successfully", slog.String("user_id", linkUserID))
		http.Redirect(w, r, "/?oidc_linked=true", http.StatusFound)
		return
	}

	// Find or create user (normal login flow)
	user, err := h.findOrCreateUser(r.Context(), claims.Sub, claims.Email, claims.Name)
	if err != nil {
		slog.Error("failed to find or create OIDC user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to process user")
		return
	}

	// Issue a biblioteka JWT
	token, err := h.JWT.CreateToken(user.ID)
	if err != nil {
		slog.Error("failed to create token for OIDC user", "user_id", user.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	slog.DebugContext(r.Context(), "OIDC login successful", slog.String("user_id", user.ID))

	// Redirect to frontend with token
	http.Redirect(w, r, "/?token="+token, http.StatusFound)
}

// findOrCreateUser looks up a user by OIDC subject, then by email, creating if needed.
// It handles races between concurrent logins by retrying lookups if CreateOIDCUser fails.
func (h *OIDCHandler) findOrCreateUser(ctx context.Context, subject, email, name string) (*db.User, error) {
	// First, try by OIDC subject
	slog.DebugContext(ctx, "OIDC findOrCreateUser: looking up by subject", slog.String("subject", subject))
	user, err := h.DB.GetUserByOIDCSubject(subject)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: found existing user by subject", slog.String("user_id", user.ID))
		return user, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Try by email — link the OIDC subject to an existing account
	slog.DebugContext(ctx, "OIDC findOrCreateUser: looking up by email", slog.String("email", email))
	user, err = h.DB.GetUserByEmail(email)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: linking subject to existing user", slog.String("user_id", user.ID))
		if err := h.DB.LinkOIDCSubject(user.ID, subject); err != nil {
			return nil, err
		}
		return user, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create a new OIDC user. If this fails due to a race with a concurrent
	// login, retry the lookups to find the user the other request created/linked.
	slog.DebugContext(ctx, "OIDC findOrCreateUser: creating new user", slog.String("email", email))
	user, err = h.DB.CreateOIDCUser(name, email, subject)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: new user created", slog.String("user_id", user.ID))
		return user, nil
	}

	// Another goroutine may have created or linked this user concurrently.
	// Re-run lookups before propagating the error.
	if u, lookupErr := h.DB.GetUserByOIDCSubject(subject); lookupErr == nil {
		return u, nil
	}
	if u, lookupErr := h.DB.GetUserByEmail(email); lookupErr == nil {
		if linkErr := h.DB.LinkOIDCSubject(u.ID, subject); linkErr != nil {
			return nil, linkErr
		}
		return u, nil
	}

	return nil, err
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
