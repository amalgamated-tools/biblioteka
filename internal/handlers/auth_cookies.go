package handlers

import (
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
)

// setAuthCookie sets an HttpOnly cookie with the JWT token for browser-based
// access to server-rendered UIs (e.g. asynqmon).
func setAuthCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:  auth.TokenCookieName(),
		Value: token,
		Path:  "/",
		// MaxAge 0 makes this a session cookie; JWT expiry is enforced by token validation.
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

// clearAuthCookie removes the auth cookie by setting MaxAge to -1.
func clearAuthCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.TokenCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}
