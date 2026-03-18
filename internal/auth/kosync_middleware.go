package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
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
var dummyKOSyncBcryptHash = mustGenerateDummyKOSyncHash()

func mustGenerateDummyKOSyncHash() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-kosync-key"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy KOSync bcrypt hash: %w", err))
	}
	return hash
}

// KOSyncHeaderAuthMiddleware returns an HTTP middleware that validates KOSync
// credentials using the x-auth-user and x-auth-key request headers and injects
// the authenticated user ID into the request context.
//
// KOReader sends x-auth-key as the hex-encoded MD5 digest of the user's password.
// The stored password hash must therefore be bcrypt(md5_hex) so that
// bcrypt.CompareHashAndPassword(stored, received_md5_hex) succeeds.
func KOSyncHeaderAuthMiddleware(checker KOSyncCredentialChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := strings.TrimSpace(r.Header.Get(kosyncAuthUserHeader))
			authKey := r.Header.Get(kosyncAuthKeyHeader)

			if username == "" || authKey == "" {
				slog.InfoContext(r.Context(), "KOSync: missing credentials")
				jsonError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			cred, err := checker.GetKOSyncCredential(r.Context(), strings.ToLower(username))
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// Perform a dummy bcrypt comparison to prevent timing-based username enumeration.
					_ = bcrypt.CompareHashAndPassword(dummyKOSyncBcryptHash, []byte(authKey))
					slog.InfoContext(r.Context(), "KOSync: unknown username", slog.String(otelkeys.KOSyncUsername, username))
					jsonError(w, http.StatusUnauthorized, "Unauthorized")
				} else {
					slog.ErrorContext(r.Context(), "KOSync: credential lookup failed", slog.Any(otelkeys.Error, err))
					jsonError(w, http.StatusServiceUnavailable, "Service temporarily unavailable")
				}
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(authKey)); err != nil {
				slog.InfoContext(r.Context(), "KOSync: invalid auth key", slog.String(otelkeys.KOSyncUsername, username))
				jsonError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			slog.DebugContext(r.Context(), "KOSync: authentication successful",
				slog.String(otelkeys.UserID, cred.UserID),
				slog.String(otelkeys.KOSyncUsername, username),
			)
			ctx := context.WithValue(r.Context(), userIDKey, cred.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
