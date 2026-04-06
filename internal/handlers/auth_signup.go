package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// Signup godoc
//
//	@Summary		Sign up a new user
//	@Description	Create a new user account with name, email, and password
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		signupRequest	true	"Signup request"
//	@Success		201		{object}	authResponse
//	@Failure		400		{object}	errorResponse
//	@Failure		409		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/auth/signup [post]
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.DisableSignup {
		writeError(r.Context(), w, http.StatusForbidden, "signup is disabled")
		return
	}

	var req signupRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name, email, and password are required")
		return
	}

	if !validatePassword(r.Context(), w, req.Password) {
		return
	}

	slog.DebugContext(r.Context(), "signup request", slog.String(otelkeys.Email, redactEmail(req.Email)))

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to hash password during signup",
			slog.String(otelkeys.Email, redactEmail(req.Email)),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.DB.CreateUser(r.Context(), req.Name, req.Email, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrEmailExists) {
			writeError(r.Context(), w, http.StatusConflict, "email already registered")
			return
		}
		slog.ErrorContext(r.Context(), "failed to create user",
			slog.String(otelkeys.Email, redactEmail(req.Email)),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create user")
		return
	}

	slog.DebugContext(r.Context(), "user created via signup", slog.String(otelkeys.UserID, user.ID))

	token, err := h.JWT.CreateToken(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create token for user",
			slog.Any(otelkeys.UserID, user.ID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create token")
		return
	}

	setAuthCookie(w, token, h.SecureCookies)

	logAudit(r.Context(), h.DB, user.ID, db.AuditActionUserSignedUp, "user", user.ID, map[string]any{"email": user.Email, "name": user.Name})

	writeJSON(r.Context(), w, http.StatusCreated, authResponse{
		Token: token,
		User:  toUserDTO(user),
	})
}
