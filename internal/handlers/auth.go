package handlers

import (
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// AuthHandler holds dependencies for auth endpoints.
type AuthHandler struct {
	DB            *db.DB
	JWT           *auth.JWTManager
	SecureCookies bool
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type authResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

type userDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	OIDCLinked bool   `json:"oidc_linked"`
	IsAdmin    bool   `json:"is_admin"`
}

func toUserDTO(u *db.User) userDTO {
	return userDTO{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		OIDCLinked: u.OIDCSubject != nil,
		IsAdmin:    u.IsAdmin,
	}
}

type updateProfileRequest struct {
	Name string `json:"name"`
}

func redactEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

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
