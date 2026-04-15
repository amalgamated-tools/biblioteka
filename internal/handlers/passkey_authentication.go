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

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/go-webauthn/webauthn/webauthn"
)

// HandleBeginAuthentication begins the passkey authentication ceremony (discoverable login).
//
//	@Summary		Begin passkey login
//	@Description	Start the WebAuthn authentication ceremony; returns options for navigator.credentials.get()
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	passkeyBeginResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/auth/passkey/login/begin [post]
func (h *PasskeyHandler) HandleBeginAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.WebAuthn == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "passkeys not configured")
		return
	}

	// Delete expired challenges to keep the table lean.
	if err := h.DB.DeleteExpiredPasskeyChallenges(r.Context()); err != nil {
		slog.WarnContext(r.Context(), "failed to delete expired passkey challenges", slog.Any(otelkeys.Error, err))
	}

	options, sd, err := h.WebAuthn.BeginDiscoverableLogin()
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to begin passkey login", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to begin login")
		return
	}

	sessionID, err := h.storeChallenge(r.Context(), nil, sd, "")
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to store passkey login challenge", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to store challenge")
		return
	}

	writeJSON(r.Context(), w, http.StatusOK, passkeyBeginResponse{
		SessionID: sessionID,
		Options:   options,
	})
}

// HandleFinishAuthentication completes the passkey authentication ceremony.
// The session_id must be provided as a query parameter; the request body is
// the raw PublicKeyCredential JSON from navigator.credentials.get().
//
//	@Summary		Finish passkey login
//	@Description	Complete the WebAuthn authentication ceremony and return a JWT on success
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			session_id	query		string	true	"Session ID from begin login"
//	@Success		200			{object}	authResponse
//	@Failure		400			{object}	errorResponse
//	@Failure		401			{object}	errorResponse
//	@Failure		500			{object}	errorResponse
//	@Router			/auth/passkey/login/finish [post]
func (h *PasskeyHandler) HandleFinishAuthentication(w http.ResponseWriter, r *http.Request) {
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

	challengeData, _, err := h.loadChallenge(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusUnauthorized, "invalid or expired session")
			return
		}
		if errors.Is(err, errPasskeySessionExpired) {
			writeError(r.Context(), w, http.StatusUnauthorized, "passkey session expired")
			return
		}
		slog.ErrorContext(r.Context(), "failed to load passkey challenge",
			slog.String(otelkeys.PasskeySessionID, sessionID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to load session")
		return
	}

	// authedUserID is captured from the handler closure during FinishPasskeyLogin.
	var authedUserID string
	var authedCredentialID string

	handler := webauthn.DiscoverableUserHandler(func(rawID, userHandle []byte) (webauthn.User, error) {
		credID := base64.RawURLEncoding.EncodeToString(rawID)
		cred, lookupErr := h.DB.GetPasskeyCredentialByCredentialID(r.Context(), credID)
		if lookupErr != nil {
			return nil, fmt.Errorf("credential not found: %w", lookupErr)
		}

		user, userErr := h.DB.GetUserByID(r.Context(), cred.UserID)
		if userErr != nil {
			return nil, fmt.Errorf("user not found: %w", userErr)
		}

		userCreds, credsErr := h.DB.ListPasskeyCredentials(r.Context(), user.ID)
		if credsErr != nil {
			return nil, fmt.Errorf("list credentials: %w", credsErr)
		}

		waCreds := loadWebAuthnCredentials(r.Context(), userCreds)

		authedUserID = user.ID
		authedCredentialID = credID

		return &passkeyUser{user: user, credentials: waCreds}, nil
	})

	updatedCred, authedUser, err := h.WebAuthn.FinishPasskeyLogin(handler, challengeData.SessionData, r)
	if err != nil {
		slog.WarnContext(r.Context(), "passkey login verification failed", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusUnauthorized, "authentication failed")
		return
	}

	_ = authedUser // user ID is captured in authedUserID

	// Persist updated credential data (sign count may have incremented).
	updatedData, err := json.Marshal(updatedCred)
	if err != nil {
		slog.WarnContext(r.Context(), "failed to marshal updated passkey credential",
			slog.String(otelkeys.UserID, authedUserID),
			slog.Any(otelkeys.Error, err),
		)
	} else {
		if updateErr := h.DB.UpdatePasskeyCredentialData(r.Context(), authedUserID, authedCredentialID, string(updatedData)); updateErr != nil {
			if errors.Is(updateErr, sql.ErrNoRows) {
				slog.ErrorContext(r.Context(), "passkey credential not found during sign-count update",
					slog.String(otelkeys.UserID, authedUserID),
					slog.String(otelkeys.PasskeyRawID, authedCredentialID),
					slog.Any(otelkeys.Error, updateErr),
				)
			} else {
				slog.WarnContext(r.Context(), "failed to update passkey credential sign count",
					slog.String(otelkeys.UserID, authedUserID),
					slog.Any(otelkeys.Error, updateErr),
				)
			}
		}
	}

	user, err := h.DB.GetUserByID(r.Context(), authedUserID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch user after passkey login",
			slog.String(otelkeys.UserID, authedUserID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to complete login")
		return
	}

	token, err := h.JWT.CreateToken(r.Context(), authedUserID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create token after passkey login",
			slog.String(otelkeys.UserID, authedUserID),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create token")
		return
	}

	slog.DebugContext(r.Context(), "passkey login successful", slog.String(otelkeys.UserID, authedUserID))

	setAuthCookie(w, token, h.SecureCookies)
	writeJSON(r.Context(), w, http.StatusOK, authResponse{
		Token: token,
		User:  toUserDTO(user),
	})
}
