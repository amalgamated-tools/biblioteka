package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"
)

// ProtocolCredentialResult holds the common credential fields returned by
// protocol-specific credential-checker interfaces (e.g. OPDSCredentialChecker,
// KOSyncCredentialChecker). All protocol middlewares share this single type to
// avoid redundant per-protocol structs with identical fields.
type ProtocolCredentialResult struct {
	UserID       string
	PasswordHash string
}

// NormalizeUsername normalizes a username for storage and lookup by trimming
// surrounding whitespace and converting to lowercase.
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// lookupByUsername constructs a lookupCredential closure from a
// protocol-specific credential-fetch function and a field-extractor. The getFn
// must return (wrapped) sql.ErrNoRows when the user is not found.
func lookupByUsername[T any](
	getFn func(ctx context.Context, username string) (*T, error),
	extract func(*T) (userID, passwordHash string),
) func(ctx context.Context, username string) (string, string, error) {
	return func(ctx context.Context, username string) (string, string, error) {
		cred, err := getFn(ctx, username)
		if err != nil {
			return "", "", err
		}
		if cred == nil {
			return "", "", sql.ErrNoRows
		}
		userID, hash := extract(cred)
		return userID, hash, nil
	}
}

// jsonErrorWriter returns an http.Handler-compatible function that writes a
// JSON error response with the given HTTP status code and message.
func jsonErrorWriter(status int, message string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		jsonError(w, status, message)
	}
}

// bcryptCredConfig holds protocol-specific callbacks for bcryptCredMiddleware.
type bcryptCredConfig struct {
	// protocolName is used in structured log messages (e.g. "KOSync", "OPDS").
	protocolName string

	// dummyHash is a pre-computed bcrypt hash used for timing-safe comparisons
	// when a username is not found.
	dummyHash []byte

	// usernameAttr builds a slog attribute for the username value. Each
	// protocol provides its own constant key (e.g. otelkeys.KOSyncUsername)
	// so that sloglint can verify keys are not raw strings.
	usernameAttr func(value string) slog.Attr

	// extractCreds pulls the username and secret from the request.
	// Return ok=false when credentials are missing entirely.
	extractCreds func(r *http.Request) (username, secret string, ok bool)

	// lookupCredential looks up the stored password hash and user ID for the
	// lower-cased, trimmed username. It MUST return (wrapped) sql.ErrNoRows
	// to signal "user not found"; any other error is treated as a transient
	// service failure.
	lookupCredential func(ctx context.Context, username string) (userID, passwordHash string, err error)

	// writeMissing writes the response when no credentials were found in the
	// request (e.g. missing Authorization header or empty username).
	writeMissing func(w http.ResponseWriter, r *http.Request)

	// writeUnauthorized writes an authentication-failure response (unknown
	// username or wrong password/key).
	writeUnauthorized func(w http.ResponseWriter, r *http.Request)

	// writeServiceUnavailable writes a transient-error response for unexpected
	// credential-lookup errors. When nil, all lookup errors are treated as
	// "unknown user" and writeUnauthorized is called instead.
	writeServiceUnavailable func(w http.ResponseWriter, r *http.Request)
}

// bcryptCredMiddleware builds an HTTP middleware that validates bcrypt
// credentials using the protocol-specific callbacks in cfg. It performs a
// constant-time dummy comparison when a username is not found to mitigate
// timing-based username enumeration attacks.
func bcryptCredMiddleware(cfg bcryptCredConfig) func(http.Handler) http.Handler {
	if cfg.usernameAttr == nil || cfg.extractCreds == nil || cfg.lookupCredential == nil || cfg.writeMissing == nil || cfg.writeUnauthorized == nil {
		panic("bcryptCredMiddleware: usernameAttr, extractCreds, lookupCredential, writeMissing, and writeUnauthorized must be non-nil")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, secret, ok := cfg.extractCreds(r)
			if !ok {
				slog.InfoContext(r.Context(), cfg.protocolName+": missing credentials")
				cfg.writeMissing(w, r)
				return
			}

			normUsername := NormalizeUsername(username)
			userID, passwordHash, err := cfg.lookupCredential(r.Context(), normUsername)
			if err != nil {
				if cfg.writeServiceUnavailable != nil && !errors.Is(err, sql.ErrNoRows) {
					slog.ErrorContext(r.Context(), cfg.protocolName+": credential lookup failed",
						slog.Any(otelkeys.Error, err),
					)
					cfg.writeServiceUnavailable(w, r)
				} else {
					// Perform a dummy bcrypt comparison to prevent timing-based
					// username enumeration.
					_ = bcrypt.CompareHashAndPassword(cfg.dummyHash, []byte(secret))
					slog.InfoContext(r.Context(), cfg.protocolName+": unknown username",
						cfg.usernameAttr(normUsername),
					)
					cfg.writeUnauthorized(w, r)
				}
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(secret)); err != nil {
				slog.InfoContext(r.Context(), cfg.protocolName+": invalid credential",
					cfg.usernameAttr(normUsername),
				)
				cfg.writeUnauthorized(w, r)
				return
			}

			slog.DebugContext(r.Context(), cfg.protocolName+": authentication successful",
				slog.String(otelkeys.UserID, userID),
				cfg.usernameAttr(normUsername),
			)
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
