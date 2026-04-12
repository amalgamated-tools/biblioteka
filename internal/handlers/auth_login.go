package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// dummyLoginBcryptHash is a pre-computed bcrypt hash used for timing-safe
// comparisons when a user is not found or has no password hash (OIDC-only).
// This prevents attackers from distinguishing these cases via response time.
var dummyLoginBcryptHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-login-password"), auth.BcryptCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy login bcrypt hash: %w", err))
	}
	return hash
}()

// Login authenticates a user with email and password, returning a signed JWT and setting an auth cookie on success.
//
//	@Summary		Log in
//	@Description	Authenticate with email and password
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login request"
//	@Success		200		{object}	authResponse
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "email and password are required")
		return
	}

	slog.DebugContext(r.Context(), "login attempt", slog.String(otelkeys.Email, redactEmail(req.Email)))

	user, err := h.DB.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.DebugContext(r.Context(), "login failed: user not found", slog.String(otelkeys.Email, redactEmail(req.Email)))
			_ = bcrypt.CompareHashAndPassword(dummyLoginBcryptHash, []byte(req.Password))
			writeError(r.Context(), w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		slog.ErrorContext(r.Context(), "failed to look up user by email",
			slog.String(otelkeys.Email, redactEmail(req.Email)),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "internal server error")
		return
	}

	if user.PasswordHash == "" {
		slog.DebugContext(r.Context(), "login failed: OIDC-only account", slog.String(otelkeys.Email, redactEmail(req.Email)))
		_ = bcrypt.CompareHashAndPassword(dummyLoginBcryptHash, []byte(req.Password))
		writeError(r.Context(), w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		slog.DebugContext(r.Context(), "login failed: invalid password", slog.String(otelkeys.Email, redactEmail(req.Email)))
		writeError(r.Context(), w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.JWT.CreateToken(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create token for user",
			slog.String(otelkeys.UserID, user.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create token")
		return
	}

	slog.DebugContext(r.Context(), "login successful",
		slog.String(otelkeys.UserID, user.ID),
		slog.String(otelkeys.Email, redactEmail(user.Email)),
	)

	setAuthCookie(w, token, h.SecureCookies)
	writeJSON(r.Context(), w, http.StatusOK, authResponse{
		Token: token,
		User:  toUserDTO(user),
	})
}
