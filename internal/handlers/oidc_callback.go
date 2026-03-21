package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Callback godoc
//
// @Summary		OIDC callback
// @Description	Handles the OIDC provider's redirect after authentication
// @Tags		OIDC
// @Param		state	query	string	true	"OIDC state parameter"
// @Param		code	query	string	true	"Authorization code"
// @Success		302	"Redirect to frontend with token"
// @Failure		400	{object}	errorResponse
// @Failure		401	{object}	errorResponse
// @Failure		500	{object}	errorResponse
// @Router		/auth/oidc/callback [get]
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slog.DebugContext(r.Context(), "OIDC callback received")

	cookie, err := r.Cookie(oidcStateCookieName)
	if err != nil || cookie.Value == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing state cookie")
		return
	}

	if r.URL.Query().Get("state") != cookie.Value {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	verifierCookie, err := r.Cookie(oidcVerifierCookieName)
	if err != nil || verifierCookie.Value == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing PKCE verifier cookie")
		return
	}

	// Clear all OIDC flow cookies upfront, before processing any outcomes.
	for _, name := range []string{oidcStateCookieName, oidcVerifierCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   h.SecureCookies,
		})
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		slog.ErrorContext(r.Context(), "OIDC provider returned error",
			slog.String(otelkeys.ErrorCode, errParam),
			slog.String(otelkeys.Description, r.URL.Query().Get("error_description")),
		)
		writeError(r.Context(), w, http.StatusUnauthorized, "authentication failed")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing authorization code")
		return
	}

	oauth2Token, err := h.Config.Exchange(r.Context(), code, oauth2.VerifierOption(verifierCookie.Value))
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to exchange OIDC code", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusUnauthorized, "failed to exchange authorization code")
		return
	}

	slog.DebugContext(r.Context(), "OIDC code exchanged successfully")

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		writeError(r.Context(), w, http.StatusUnauthorized, "missing id_token in response")
		return
	}

	verifier := h.Provider.Verifier(&oidc.Config{ClientID: h.Config.ClientID})
	idToken, err := verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to verify OIDC id_token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusUnauthorized, "invalid id_token")
		return
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified *bool  `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		slog.ErrorContext(r.Context(), "failed to parse OIDC claims", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to parse user info")
		return
	}

	if claims.Sub == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "sub claim is required")
		return
	}
	if claims.Email == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "email claim is required")
		return
	}
	if claims.Name == "" {
		claims.Name = claims.Email
	}

	// Check if this callback is part of a link flow by looking up the state
	// in the server-side linkStates map. This prevents cookie forgery attacks
	// where an attacker could set a fake link user ID cookie.
	linkUserID := h.consumeLinkState(cookie.Value)

	if linkUserID == "" && (claims.EmailVerified == nil || !*claims.EmailVerified) {
		slog.WarnContext(r.Context(), "OIDC login rejected: email not verified by identity provider",
			slog.String(otelkeys.Email, claims.Email),
		)
		writeError(r.Context(), w, http.StatusUnauthorized, "OIDC email must be verified by the identity provider")
		return
	}

	if linkUserID != "" {
		slog.DebugContext(r.Context(), "OIDC link flow: linking account",
			slog.String(otelkeys.UserID, linkUserID),
			slog.String(otelkeys.Subject, claims.Sub),
		)
		user, err := h.DB.GetUserByID(r.Context(), linkUserID)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to get user for OIDC link", slog.Any(otelkeys.Error, err))
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("User not found"), http.StatusFound)
			return
		}
		if user.OIDCSubject != nil {
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("Account is already linked to an SSO provider"), http.StatusFound)
			return
		}
		if existing, err := h.DB.GetUserByOIDCSubject(r.Context(), claims.Sub); err == nil && existing.ID != linkUserID {
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("This SSO identity is already linked to another account"), http.StatusFound)
			return
		}
		if err := h.DB.LinkOIDCSubject(r.Context(), linkUserID, claims.Sub); err != nil {
			slog.ErrorContext(r.Context(), "failed to link OIDC subject to user",
				slog.String(otelkeys.UserID, linkUserID),
				slog.Any(otelkeys.Error, err),
			)
			http.Redirect(w, r, "/?oidc_link_error="+url.QueryEscape("Failed to link account"), http.StatusFound)
			return
		}
		slog.DebugContext(r.Context(), "OIDC account linked successfully", slog.String(otelkeys.UserID, linkUserID))
		http.Redirect(w, r, "/?oidc_linked=true", http.StatusFound)
		return
	}

	user, err := h.findOrCreateUser(r.Context(), claims.Sub, claims.Email, claims.Name)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to find or create OIDC user", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to process user")
		return
	}

	token, err := h.JWT.CreateToken(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create token for OIDC user",
			slog.String(otelkeys.UserID, user.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create session")
		return
	}

	slog.DebugContext(r.Context(), "OIDC login successful", slog.String(otelkeys.UserID, user.ID))

	setAuthCookie(w, token, h.SecureCookies)
	http.Redirect(w, r, "/?oidc_login=1", http.StatusFound)
}

// findOrCreateUser looks up a user by OIDC subject, then by email, creating if needed.
// It handles races between concurrent logins by retrying lookups if CreateOIDCUser fails.
func (h *OIDCHandler) findOrCreateUser(ctx context.Context, subject, email, name string) (*db.User, error) {
	slog.DebugContext(ctx, "OIDC findOrCreateUser: looking up by subject", slog.String(otelkeys.Subject, subject))
	user, err := h.DB.GetUserByOIDCSubject(ctx, subject)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: found existing user by subject", slog.String(otelkeys.UserID, user.ID))
		return user, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	slog.DebugContext(ctx, "OIDC findOrCreateUser: looking up by email", slog.String(otelkeys.Email, email))
	user, err = h.DB.GetUserByEmail(ctx, email)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: linking subject to existing user", slog.String(otelkeys.UserID, user.ID))
		if err := h.DB.LinkOIDCSubject(ctx, user.ID, subject); err != nil {
			return nil, err
		}
		return user, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	slog.DebugContext(ctx, "OIDC findOrCreateUser: creating new user", slog.String(otelkeys.Email, email))
	user, err = h.DB.CreateOIDCUser(ctx, name, email, subject)
	if err == nil {
		slog.DebugContext(ctx, "OIDC findOrCreateUser: new user created", slog.String(otelkeys.UserID, user.ID))
		return user, nil
	}

	if u, lookupErr := h.DB.GetUserByOIDCSubject(ctx, subject); lookupErr == nil {
		return u, nil
	}
	if u, lookupErr := h.DB.GetUserByEmail(ctx, email); lookupErr == nil {
		if linkErr := h.DB.LinkOIDCSubject(ctx, u.ID, subject); linkErr != nil {
			return nil, linkErr
		}
		return u, nil
	}

	return nil, err
}
