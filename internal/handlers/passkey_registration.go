package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/go-webauthn/webauthn/webauthn"
)

// HandleBeginRegistration begins the passkey registration ceremony for the authenticated user.
//
//	@Summary		Begin passkey registration
//	@Description	Start the WebAuthn registration ceremony; returns options for navigator.credentials.create()
//	@Tags			Auth
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		beginRegistrationRequest	true	"Name for the new passkey"
//	@Success		200		{object}	passkeyBeginResponse
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/auth/passkey/register/begin [post]
func (h *PasskeyHandler) HandleBeginRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.WebAuthn == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "passkeys not configured")
		return
	}

	var req beginRegistrationRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "passkey name is required")
		return
	}
	if len(req.Name) > maxTokenNameLength {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Sprintf("passkey name must be %d characters or fewer", maxTokenNameLength))
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "user not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch user for passkey registration",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	existingCreds, err := h.DB.ListPasskeyCredentials(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list passkey credentials",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	waCreds := loadWebAuthnCredentials(r.Context(), existingCreds)

	waUser := &passkeyUser{user: user, credentials: waCreds}

	// Delete expired challenges to keep the table lean.
	if err = h.DB.DeleteExpiredPasskeyChallenges(r.Context()); err != nil {
		slog.WarnContext(r.Context(), "failed to delete expired passkey challenges", slog.Any(otelkeys.Error, err))
	}

	options, sd, err := h.WebAuthn.BeginRegistration(waUser, webauthn.WithExclusions(webauthn.Credentials(waUser.WebAuthnCredentials()).CredentialDescriptors()))
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to begin passkey registration",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to begin registration")
		return
	}

	uid := userID
	sessionID, err := h.storeChallenge(r.Context(), &uid, sd, req.Name)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to store passkey challenge",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to store challenge")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, passkeyBeginResponse{
		SessionID: sessionID,
		Options:   options,
	})
}

// HandleFinishRegistration completes the passkey registration ceremony.
// The session_id must be provided as a query parameter; the request body is
// the raw PublicKeyCredential JSON from navigator.credentials.create().
//
//	@Summary		Finish passkey registration
//	@Description	Complete the WebAuthn registration ceremony; body is the raw credential from navigator.credentials.create()
//	@Tags			Auth
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			session_id	query		string	true	"Session ID from begin registration"
//	@Success		201			{object}	passkeyCredentialDTO
//	@Failure		400			{object}	errorResponse
//	@Failure		401			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Router			/auth/passkey/register/finish [post]
func (h *PasskeyHandler) HandleFinishRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.WebAuthn == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "passkeys not configured")
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "session_id is required")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	challengeData, storedUserID, err := h.loadChallenge(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid or expired session")
			return
		}
		if errors.Is(err, errPasskeySessionExpired) {
			writeError(r.Context(), w, http.StatusBadRequest, "passkey session expired")
			return
		}
		slog.ErrorContext(r.Context(), "failed to load passkey challenge",
			slog.String(otelkeys.PasskeySessionID, sessionID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to load session")
		return
	}

	if storedUserID != nil && *storedUserID != userID {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid or expired session")
		return
	}

	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "user not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch user for passkey finish",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	existingCreds, err := h.DB.ListPasskeyCredentials(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list passkey credentials",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	waCreds := loadWebAuthnCredentials(r.Context(), existingCreds)

	waUser := &passkeyUser{user: user, credentials: waCreds}

	credential, err := h.WebAuthn.FinishRegistration(waUser, challengeData.SessionData, r)
	if err != nil {
		slog.WarnContext(r.Context(), "passkey registration verification failed",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadRequest, "registration verification failed")
		return
	}

	credentialData, err := json.Marshal(credential)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to marshal passkey credential",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to store credential")
		return
	}

	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	aaguid := base64.RawURLEncoding.EncodeToString(credential.Authenticator.AAGUID)

	stored, err := h.DB.CreatePasskeyCredential(r.Context(), userID, challengeData.Name, credentialID, string(credentialData), aaguid)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to store passkey credential",
			slog.String(otelkeys.UserID, userID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to store credential")
		return
	}

	slog.DebugContext(r.Context(), "passkey registered",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.PasskeyCredentialID, stored.ID),
	)

	logAudit(r.Context(), h.DB, userID, db.AuditActionPasskeyCreated, "passkey", stored.ID, map[string]any{"name": stored.Name})

	writeJSON(r.Context(), w, http.StatusCreated, toPasskeyCredentialDTO(stored))
}
