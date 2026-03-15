package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// OPDSCredentialResult holds the fields needed by the OPDS Basic Auth middleware.
type OPDSCredentialResult struct {
	UserID       string
	PasswordHash string
}

// OPDSCredentialChecker is implemented by types that can look up OPDS credentials by username.
type OPDSCredentialChecker interface {
	GetOPDSCredential(ctx context.Context, username string) (*OPDSCredentialResult, error)
}

// dummyOPDSBcryptHash is a precomputed valid bcrypt hash used for timing-safe
// comparisons when a username is not found, to mitigate username enumeration
// via timing attacks.
var dummyOPDSBcryptHash = mustGenerateDummyOPDSHash()

func mustGenerateDummyOPDSHash() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-opds-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy OPDS bcrypt hash: %w", err))
	}
	return hash
}

// OPDSBasicAuthMiddleware returns an HTTP middleware that validates OPDS
// credentials using HTTP Basic Authentication and injects the user ID into
// the request context.
func OPDSBasicAuthMiddleware(checker OPDSCredentialChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username == "" {
				slog.InfoContext(r.Context(), "OPDS: missing credentials")
				w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka OPDS"`)
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			cred, err := checker.GetOPDSCredential(r.Context(), strings.ToLower(username))
			if err != nil {
				// Perform a dummy bcrypt comparison to prevent timing-based username enumeration.
				_ = bcrypt.CompareHashAndPassword(dummyOPDSBcryptHash, []byte(password))
				slog.InfoContext(r.Context(), "OPDS: unknown username", slog.String(otelkeys.OPDSUsername, username))
				w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka OPDS"`)
				jsonError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)); err != nil {
				slog.InfoContext(r.Context(), "OPDS: invalid password", slog.String(otelkeys.OPDSUsername, username))
				w.Header().Set("WWW-Authenticate", `Basic realm="Biblioteka OPDS"`)
				jsonError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}

			slog.DebugContext(r.Context(), "OPDS: authentication successful",
				slog.String(otelkeys.UserID, cred.UserID),
				slog.String(otelkeys.OPDSUsername, username),
			)
			ctx := context.WithValue(r.Context(), userIDKey, cred.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
