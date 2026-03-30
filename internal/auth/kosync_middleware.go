package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

const (
	// kosyncAuthUserHeader is the header name carrying the KOReader sync username.
	kosyncAuthUserHeader = "x-auth-user"
	// kosyncAuthKeyHeader is the header name carrying the KOReader sync auth key
	// (the hex-encoded MD5 digest of the user's password, as sent by KOReader).
	kosyncAuthKeyHeader = "x-auth-key"
)

// KOSyncCredentialResult holds the fields needed by the KOSync header auth middleware.
type KOSyncCredentialResult struct {
	UserID       string
	PasswordHash string
}

// KOSyncCredentialChecker is implemented by types that can look up KOSync credentials by username.
type KOSyncCredentialChecker interface {
	GetKOSyncCredential(ctx context.Context, username string) (*KOSyncCredentialResult, error)
}

// dummyKOSyncBcryptHash is a precomputed valid bcrypt hash used for timing-safe
// comparisons when a username is not found, to mitigate username enumeration
// via timing attacks.
var dummyKOSyncBcryptHash = mustGenerateDummyBcryptHash("dummy-kosync-key", "KOSync")

// KOSyncHeaderAuthMiddleware returns an HTTP middleware that validates KOSync
// credentials using the x-auth-user and x-auth-key request headers and injects
// the authenticated user ID into the request context.
//
// KOReader sends x-auth-key as the hex-encoded MD5 digest of the user's password.
// The stored password hash must therefore be bcrypt(md5_hex) so that
// bcrypt.CompareHashAndPassword(stored, received_md5_hex) succeeds.
func KOSyncHeaderAuthMiddleware(checker KOSyncCredentialChecker) func(http.Handler) http.Handler {
	return bcryptCredMiddleware(bcryptCredConfig{
		protocolName: "KOSync",
		dummyHash:    dummyKOSyncBcryptHash,
		usernameKey:  otelkeys.KOSyncUsername,
		extractCreds: func(r *http.Request) (username, secret string, ok bool) {
			username = r.Header.Get(kosyncAuthUserHeader)
			secret = r.Header.Get(kosyncAuthKeyHeader)
			// Trim only for the empty check; normalization is done by the
			// shared middleware before the credential lookup.
			if strings.TrimSpace(username) == "" || secret == "" {
				return "", "", false
			}
			return username, secret, true
		},
		lookupCredential: func(ctx context.Context, username string) (string, string, error) {
			cred, err := checker.GetKOSyncCredential(ctx, username)
			if err != nil {
				return "", "", err
			}
			return cred.UserID, cred.PasswordHash, nil
		},
		writeMissing: func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, http.StatusUnauthorized, "Unauthorized")
		},
		writeUnauthorized: func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, http.StatusUnauthorized, "Unauthorized")
		},
		writeServiceUnavailable: func(w http.ResponseWriter, r *http.Request) {
			jsonError(w, http.StatusServiceUnavailable, "Service temporarily unavailable")
		},
	})
}
