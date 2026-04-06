package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// ChangePassword godoc
//
//	@Summary		Change password
//	@Description	Change the authenticated user's password
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Failure		401		{object}	errorResponse
//	@Param			body	body		changePasswordRequest	true	"Change password request"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/auth/password [put]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req changePasswordRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	if req.NewPassword == "" || req.CurrentPassword == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "current password and new password are required")
		return
	}

	if !validatePassword(r.Context(), w, req.NewPassword) {
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusUnauthorized, "user not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to get user for password change",
			slog.Any(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user.PasswordHash == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "cannot change password for OIDC-only account")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		writeError(r.Context(), w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to hash new password",
			slog.Any(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.DB.UpdatePassword(r.Context(), userID, string(hash)); err != nil {
		slog.ErrorContext(r.Context(), "failed to update password",
			slog.Any(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to update password")
		return
	}

	slog.DebugContext(r.Context(), "password changed", slog.String(otelkeys.UserID, userID))
	logAudit(r.Context(), h.DB, userID, db.AuditActionPasswordChanged, "user", userID, nil)

	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "password updated"})
}
