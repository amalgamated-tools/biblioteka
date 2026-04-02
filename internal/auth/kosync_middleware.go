package auth

import (
	"context"
	"log/slog"
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
		usernameAttr: func(v string) slog.Attr { return slog.String(otelkeys.KOSyncUsername, v) },
		extractCreds: func(r *http.Request) (username, secret string, ok bool) {
			username = strings.TrimSpace(r.Header.Get(kosyncAuthUserHeader))
			secret = r.Header.Get(kosyncAuthKeyHeader)
			return username, secret, username != "" && secret != ""
		},
		lookupCredential: lookupByUsername(checker.GetKOSyncCredential, func(c *KOSyncCredentialResult) (string, string) {
			return c.UserID, c.PasswordHash
		}),
		writeMissing:            jsonErrorWriter(http.StatusUnauthorized, "Unauthorized"),
		writeUnauthorized:       jsonErrorWriter(http.StatusUnauthorized, "Unauthorized"),
		writeServiceUnavailable: jsonErrorWriter(http.StatusServiceUnavailable, "Service temporarily unavailable"),
	})
}
