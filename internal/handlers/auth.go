package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"golang.org/x/crypto/bcrypt"
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
	Email      string `json:"email"`
	OIDCLinked bool   `json:"oidc_linked"`
	IsAdmin    bool   `json:"is_admin"`
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

// Signup godoc
// @Summary     Sign up a new user
// @Description Create a new user account with name, email, and password
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body     signupRequest true "Signup request"
// @Success     201  {object} authResponse
// @Failure     400  {object} errorResponse
// @Failure     409  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /auth/signup [post]
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email, and password are required")
		return
	}

	if msg := validatePassword(req.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	slog.DebugContext(r.Context(), "signup request", slog.String("email", redactEmail(req.Email)))

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password during signup", slog.String("email", redactEmail(req.Email)), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.DB.CreateUser(r.Context(), req.Name, req.Email, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrEmailExists) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("failed to create user", slog.String("email", redactEmail(req.Email)), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	slog.DebugContext(r.Context(), "user created via signup", slog.String("user_id", user.ID))

	token, err := h.JWT.CreateToken(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to create token for user", slog.Any("user_id", user.ID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	setAuthCookie(w, token, h.SecureCookies)
	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, IsAdmin: user.IsAdmin},
	})
}

// Login godoc
// @Summary     Log in
// @Description Authenticate with email and password
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Param       body body     loginRequest true "Login request"
// @Success     200  {object} authResponse
// @Failure     400  {object} errorResponse
// @Failure     401  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	slog.DebugContext(r.Context(), "login attempt", slog.String("email", req.Email))

	user, err := h.DB.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		slog.DebugContext(r.Context(), "login failed: user not found", slog.String("email", req.Email))
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if user.PasswordHash == "" {
		slog.DebugContext(r.Context(), "login failed: OIDC-only account", slog.String("email", req.Email))
		writeError(w, http.StatusUnauthorized, "this account uses OIDC login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.DebugContext(r.Context(), "login failed: invalid password", slog.String("email", req.Email))
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.JWT.CreateToken(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to create token for user", slog.Any("user_id", user.ID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	slog.DebugContext(r.Context(), "login successful", slog.String("user_id", user.ID), slog.String("email", user.Email))

	setAuthCookie(w, token, h.SecureCookies)
	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, IsAdmin: user.IsAdmin},
	})
}

// Me godoc
// @Summary     Get current user
// @Description Returns the authenticated user's profile
// @Tags        Auth
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Success     200 {object} userDTO
// @Failure     404 {object} errorResponse
// @Failure     500 {object} errorResponse
// @Router      /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	slog.DebugContext(r.Context(), "fetching current user", slog.String("user_id", userID))

	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("failed to get user", slog.Any("user_id", userID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	writeJSON(w, http.StatusOK, userDTO{ID: user.ID, Email: user.Email, OIDCLinked: user.OIDCSubject != nil, IsAdmin: user.IsAdmin})
}

// ChangePassword godoc
// @Summary     Change password
// @Description Change the authenticated user's password
// @Tags        Auth
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Failure     401 {object} errorResponse
// @Param       body body     changePasswordRequest true "Change password request"
// @Success     200  {object} object{message=string}
// @Failure     400  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /auth/password [put]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewPassword == "" || req.CurrentPassword == "" {
		writeError(w, http.StatusBadRequest, "current password and new password are required")
		return
	}

	if msg := validatePassword(req.NewPassword); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user for password change", slog.Any("user_id", userID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user.PasswordHash == "" {
		writeError(w, http.StatusBadRequest, "cannot change password for OIDC-only account")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash new password", slog.Any("user_id", userID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.DB.UpdatePassword(r.Context(), userID, string(hash)); err != nil {
		slog.Error("failed to update password", slog.Any("user_id", userID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	slog.DebugContext(r.Context(), "password changed", slog.String("user_id", userID))

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// Logout godoc
// @Summary     Log out
// @Description Clears the authentication cookie
// @Tags        Auth
// @Produce     json
// @Success     200 {object} object{message=string}
// @Failure     405 {object} errorResponse
// @Router      /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "invalid logout request origin")
		return
	}

	clearAuthCookie(w, h.SecureCookies)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		return matchRequestOrigin(u, r)
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		return false
	}

	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return false
	}

	return matchRequestOrigin(u, r)
}

func matchRequestOrigin(u *url.URL, r *http.Request) bool {
	// Compare scheme.
	reqScheme := "http"
	if r.TLS != nil {
		reqScheme = "https"
	}
	if !strings.EqualFold(u.Scheme, reqScheme) {
		return false
	}

	// Compare host (without port).
	originHost, originPort := parseHostPort(u.Host, defaultPort(u.Scheme))
	reqHost, reqPort := parseHostPort(r.Host, defaultPort(reqScheme))

	if !strings.EqualFold(originHost, reqHost) {
		return false
	}

	return originPort == reqPort
}

func defaultPort(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https":
		return "443"
	default:
		return "80"
	}
}

func parseHostPort(hostport, defaultPort string) (string, string) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		// Likely no port present; fall back to the provided default port.
		return hostport, defaultPort
	}
	if port == "" {
		port = defaultPort
	}
	return host, port
}
