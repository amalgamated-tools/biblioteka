package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/go-webauthn/webauthn/webauthn"
)

const passkeyChallengeExpiry = 5 * time.Minute

// errPasskeySessionExpired is returned by loadChallenge when the challenge's
// TTL has elapsed.
var errPasskeySessionExpired = errors.New("passkey session expired")

// PasskeyHandler holds dependencies for passkey/WebAuthn endpoints.
type PasskeyHandler struct {
	DB            *db.DB
	WebAuthn      *webauthn.WebAuthn
	JWT           *auth.JWTManager
	SecureCookies bool
}

// passkeyUser adapts db.User and its credentials to implement webauthn.User.
type passkeyUser struct {
	user        *db.User
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte          { return []byte(u.user.ID) }
func (u *passkeyUser) WebAuthnName() string        { return u.user.Email }
func (u *passkeyUser) WebAuthnDisplayName() string { return u.user.Name }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// passkeyChallengeData is stored in the passkey_challenges table.
// It wraps the WebAuthn session data and an optional passkey name for registration.
type passkeyChallengeData struct {
	SessionData webauthn.SessionData `json:"session_data"`
	Name        string               `json:"name,omitempty"`
}

// passkeyCredentialDTO is the public representation of a passkey credential.
type passkeyCredentialDTO struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	AAGUID    string       `json:"aaguid"`
	CreatedAt db.Timestamp `json:"created_at"`
}

func toPasskeyCredentialDTO(c *db.PasskeyCredential) passkeyCredentialDTO {
	return passkeyCredentialDTO{
		ID:        c.ID,
		Name:      c.Name,
		AAGUID:    c.AAGUID,
		CreatedAt: c.CreatedAt,
	}
}

type beginRegistrationRequest struct {
	Name string `json:"name"`
}

type passkeyBeginResponse struct {
	SessionID string `json:"session_id"`
	Options   any    `json:"options"`
}

// loadWebAuthnCredentials deserializes a slice of db.PasskeyCredential into webauthn.Credential values.
// Corrupted entries are logged and skipped rather than aborting the entire operation.
func loadWebAuthnCredentials(ctx context.Context, creds []db.PasskeyCredential) []webauthn.Credential {
	result := make([]webauthn.Credential, 0, len(creds))
	for i := range creds {
		var waCred webauthn.Credential
		if err := json.Unmarshal([]byte(creds[i].CredentialData), &waCred); err != nil {
			slog.WarnContext(ctx, "skipping corrupted passkey credential",
				slog.String(otelkeys.PasskeyCredentialID, creds[i].ID),
				slog.Any(otelkeys.Error, err),
			)
			continue
		}
		result = append(result, waCred)
	}
	return result
}

// storeChallenge JSON-encodes the session data and persists it. Returns the session ID.
func (h *PasskeyHandler) storeChallenge(ctx context.Context, userID *string, sd *webauthn.SessionData, name string) (string, error) {
	data := passkeyChallengeData{
		SessionData: *sd,
		Name:        name,
	}
	enc, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal challenge: %w", err)
	}

	expiresAt := time.Now().UTC().Add(passkeyChallengeExpiry)
	challenge, err := h.DB.CreatePasskeyChallenge(ctx, userID, string(enc), expiresAt)
	if err != nil {
		return "", fmt.Errorf("store challenge: %w", err)
	}
	return challenge.ID, nil
}

// loadChallenge retrieves, deletes, and decodes a stored challenge.
// Returns the decoded data, the stored user ID (nil for login challenges), and an error if expired.
func (h *PasskeyHandler) loadChallenge(ctx context.Context, id string) (*passkeyChallengeData, *string, error) {
	rec, err := h.DB.GetAndDeletePasskeyChallenge(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if time.Now().UTC().After(rec.ExpiresAt.Time) {
		return nil, nil, errPasskeySessionExpired
	}
	var data passkeyChallengeData
	if err = json.Unmarshal([]byte(rec.SessionData), &data); err != nil {
		return nil, nil, fmt.Errorf("unmarshal challenge: %w", err)
	}
	return &data, rec.UserID, nil
}

// HandlePasskeyEnabled reports whether passkeys are configured on this server.
//
//	@Summary		Check if passkeys are enabled
//	@Description	Returns whether WebAuthn/passkey authentication is configured
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	map[string]bool
//	@Router			/auth/passkey/enabled [get]
func (h *PasskeyHandler) HandlePasskeyEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(r.Context(), w, http.StatusOK, map[string]bool{"enabled": h.WebAuthn != nil})
}

// HandlePasskeyCredentials dispatches GET (list) for /api/auth/passkey/credentials.
//
//	@Summary		List passkey credentials
//	@Description	List all passkey credentials for the authenticated user
//	@Tags			Auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{array}		passkeyCredentialDTO
//	@Failure		401	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/auth/passkey/credentials [get]
func (h *PasskeyHandler) HandlePasskeyCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listUserEntities(w, r, "passkey credentials", h.DB.ListPasskeyCredentials, toPasskeyCredentialDTO)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandlePasskeyCredential dispatches DELETE for /api/auth/passkey/credentials/{id}.
//
//	@Summary		Delete a passkey credential
//	@Description	Delete a specific passkey credential owned by the authenticated user
//	@Tags			Auth
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Passkey credential ID"
//	@Success		204	"Passkey deleted"
//	@Failure		400	{object}	errorResponse
//	@Failure		401	{object}	errorResponse
//	@Failure		404	{object}	errorResponse
//	@Failure		405	{object}	errorResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/auth/passkey/credentials/{id} [delete]
func (h *PasskeyHandler) HandlePasskeyCredential(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/auth/passkey/credentials/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid passkey credential ID")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		deleteUserOwnedResource(h.DB, w, r, id, "passkey", "passkey", otelkeys.PasskeyCredentialID,
			h.DB.GetPasskeyCredential, h.DB.DeletePasskeyCredential,
			db.AuditActionPasskeyDeleted,
			func(c *db.PasskeyCredential) map[string]any { return map[string]any{"name": c.Name} },
		)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

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
