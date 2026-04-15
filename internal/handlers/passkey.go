package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
