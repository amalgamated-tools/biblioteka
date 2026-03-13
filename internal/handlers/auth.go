package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler holds dependencies for auth endpoints.
type AuthHandler struct {
	DB  *db.DB
	JWT *auth.JWTManager
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

// Signup handles POST /api/auth/signup
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password during signup", slog.Any("email", req.Email), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.DB.CreateUser(req.Name, req.Email, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrEmailExists) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("failed to create user", slog.Any("email", req.Email), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := h.JWT.CreateToken(user.ID)
	if err != nil {
		slog.Error("failed to create token for user", slog.Any("user_id", user.ID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, IsAdmin: user.IsAdmin},
	})
}

// Login handles POST /api/auth/login
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

	user, err := h.DB.GetUserByEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if user.PasswordHash == "" {
		writeError(w, http.StatusUnauthorized, "this account uses OIDC login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.JWT.CreateToken(user.ID)
	if err != nil {
		slog.Error("failed to create token for user", slog.Any("user_id", user.ID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userDTO{ID: user.ID, Email: user.Email, IsAdmin: user.IsAdmin},
	})
}

// Me handles GET /api/auth/me (requires auth middleware)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	user, err := h.DB.GetUserByID(userID)
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

// ChangePassword handles PUT /api/auth/password (requires auth middleware)
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
	user, err := h.DB.GetUserByID(userID)
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

	if err := h.DB.UpdatePassword(userID, string(hash)); err != nil {
		slog.Error("failed to update password", slog.Any("user_id", userID), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
