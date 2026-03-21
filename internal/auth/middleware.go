package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// APIKeyPrefix is the prefix that distinguishes API keys from JWT tokens.
const APIKeyPrefix = "bib_"

// APIKeyValidator looks up an API key by its SHA-256 hash.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, keyHash string) (userID string, apiKeyID string, err error)
	TouchAPIKeyLastUsed(ctx context.Context, id string) error
}

// HashAPIKey returns the hex-encoded SHA-256 hash of the given API key.
// SHA-256 is appropriate here because API keys are high-entropy random tokens
// (128 bits), not user-chosen passwords. Expensive hashing (bcrypt/argon2) is
// unnecessary — an attacker cannot brute-force 128-bit keys regardless of hash speed.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key)) // #nosec G401 -- not a password; high-entropy API key
	return hex.EncodeToString(h[:])
}

var (
	apiKeyTouchMu       sync.Mutex
	apiKeyLastTouchedAt = make(map[string]time.Time)
)

// apiKeyTouchInterval controls how often we will update last_used_at for a
// given API key. This throttles DB writes while keeping the field reasonably fresh.
const apiKeyTouchInterval = 5 * time.Minute

// shouldTouchAPIKeyLastUsed returns true if we should issue a TouchAPIKeyLastUsed
// call for the given API key ID at the provided time. It uses an in-memory,
// process-local cache to avoid writing on every request for frequently used keys.
// Expired entries are swept when the cache exceeds a size threshold to prevent
// unbounded memory growth.
func shouldTouchAPIKeyLastUsed(id string, now time.Time) bool {
	apiKeyTouchMu.Lock()
	defer apiKeyTouchMu.Unlock()

	last, ok := apiKeyLastTouchedAt[id]
	if ok && now.Sub(last) < apiKeyTouchInterval {
		return false
	}

	apiKeyLastTouchedAt[id] = now

	// Sweep expired entries when the map grows beyond a reasonable size.
	const sweepThreshold = 100
	const maxCacheSize = 200
	if len(apiKeyLastTouchedAt) >= sweepThreshold {
		for k, v := range apiKeyLastTouchedAt {
			if now.Sub(v) >= apiKeyTouchInterval {
				delete(apiKeyLastTouchedAt, k)
			}
		}

		// Enforce a hard upper bound even if nothing was old enough to sweep.
		if len(apiKeyLastTouchedAt) > maxCacheSize {
			for k := range apiKeyLastTouchedAt {
				if k == id {
					continue // keep the entry we just inserted
				}
				delete(apiKeyLastTouchedAt, k)
				if len(apiKeyLastTouchedAt) <= maxCacheSize {
					break
				}
			}
		}
	}

	return true
}

// resolveUser determines the user ID from the given token. If the token starts
// with "bib_" and was sourced from the Authorization header, it is treated as
// an API key; API keys from cookies are rejected to prevent CSRF attacks.
// Otherwise it is validated as a JWT.
func resolveUser(ctx context.Context, token string, source tokenSource, jwt *JWTManager, apiKeys APIKeyValidator) (userID string, err error) {
	if apiKeys != nil && strings.HasPrefix(token, APIKeyPrefix) {
		if source != tokenSourceHeader {
			// API keys from non-header sources (e.g., cookies) are considered invalid to
			// prevent CSRF; treat this as an authentication failure, not a server error.
			return "", ErrInvalidToken
		}
		keyHash := HashAPIKey(token)
		uid, keyID, err := apiKeys.ValidateAPIKey(ctx, keyHash)
		if err != nil {
			// A "not found" API key is an auth failure and should be normalized to
			// ErrInvalidToken so that middleware returns 401 instead of 500.
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrInvalidToken
			}
			// Unexpected validator/DB errors are propagated so they surface as 500s.
			return "", err
		}

		// Throttle last_used_at updates to avoid excessive DB writes on hot keys.
		if shouldTouchAPIKeyLastUsed(keyID, time.Now()) {
			if err := apiKeys.TouchAPIKeyLastUsed(ctx, keyID); err != nil {
				slog.WarnContext(ctx, "failed to touch API key last_used_at",
					slog.String(otelkeys.UserID, uid),
					slog.Any(otelkeys.Error, err),
				)
			}
		}

		return uid, nil
	}
	claims, err := jwt.ValidateToken(ctx, token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

type contextKey string

const userIDKey contextKey = "userID"

// jsonError writes a JSON-formatted error response with the given status code.
// we use this instead of the writeError function from handlers/helpers.go to avoid
// importing the entire handlers package and creating a circular dependency.
func jsonError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// tokenSource indicates where a token was extracted from.
type tokenSource int

const (
	tokenSourceNone   tokenSource = iota
	tokenSourceHeader             // Authorization header
	tokenSourceCookie             // biblioteka_token cookie
)

// extractToken tries to read a JWT from the Authorization header first,
// then falls back to the "biblioteka_token" cookie. It returns the token,
// its source, and an optional reason describing why no token could be extracted.
func extractToken(r *http.Request) (string, tokenSource, string) {
	if header := r.Header.Get("Authorization"); header != "" {
		header = strings.TrimSpace(header)
		if after, ok := strings.CutPrefix(header, "Bearer "); ok {
			token := strings.TrimSpace(after)
			if token != "" {
				return token, tokenSourceHeader, ""
			}
		}
		// Non-Bearer or empty Bearer — fall through to cookie.
	}

	// Fallback to cookie-based authentication.
	if c, err := r.Cookie(tokenCookieName); err == nil && c.Value != "" {
		return c.Value, tokenSourceCookie, ""
	}

	return "", tokenSourceNone, "missing token"
}

// Middleware returns an HTTP middleware that validates the JWT or API key from the
// Authorization header or biblioteka_token cookie and injects the user ID into
// the request context. The apiKeys parameter may be nil to disable API key auth.
func Middleware(jwt *JWTManager, apiKeys APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, source, reason := extractToken(r)
			if token == "" {
				if reason != "" {
					slog.InfoContext(r.Context(), "authentication required", slog.String(otelkeys.Reason, reason))
				} else {
					slog.InfoContext(r.Context(), "authentication required")
				}
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			userID, err := resolveUser(r.Context(), token, source, jwt, apiKeys)
			if err != nil {
				// Distinguish between expected auth failures (invalid/expired token)
				// and unexpected internal errors (e.g., database/network issues).
				if errors.Is(err, ErrInvalidToken) {
					slog.InfoContext(r.Context(), "invalid or expired token", slog.Any(otelkeys.Error, err))
					jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				} else {
					slog.ErrorContext(
						r.Context(),
						"failed to resolve authenticated user",
						slog.Any(otelkeys.Error, err),
					)
					jsonError(w, http.StatusInternalServerError, "internal server error")
				}
				return
			}

			slog.DebugContext(r.Context(), "authentication successful", slog.String(otelkeys.UserID, userID))
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the user ID set by the auth middleware.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// AdminChecker is implemented by types that can verify whether a user is an admin.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

// tokenCookieName is the name of the HttpOnly cookie used as a fallback for
// browser-navigated endpoints that cannot send an Authorization header.
const tokenCookieName = "biblioteka_token"

// TokenCookieName returns the cookie name used for browser-based auth.
func TokenCookieName() string { return tokenCookieName }

// adminCacheEntry caches the result of an admin check along with its
// expiration time to avoid repeated calls to the underlying AdminChecker.
type adminCacheEntry struct {
	isAdmin   bool
	expiresAt time.Time
}

type cachingAdminChecker struct {
	delegate AdminChecker
	ttl      time.Duration

	mu      sync.RWMutex
	entries map[string]adminCacheEntry
}

func newCachingAdminChecker(delegate AdminChecker, ttl time.Duration) AdminChecker {
	if ttl <= 0 {
		// Fallback to a small default TTL if an invalid value is provided.
		ttl = 5 * time.Second
	}
	return &cachingAdminChecker{
		delegate: delegate,
		ttl:      ttl,
		entries:  make(map[string]adminCacheEntry),
	}
}

func (c *cachingAdminChecker) IsAdmin(ctx context.Context, userID string) (bool, error) {
	// Fast path: check cache under read lock.
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[userID]
	c.mu.RUnlock()

	if ok && now.Before(entry.expiresAt) {
		return entry.isAdmin, nil
	}

	// Cache miss or expired; consult the underlying checker.
	isAdmin, err := c.delegate.IsAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("admin check failed: %w", err)
	}

	// Store/refresh the cache entry under write lock; sweep expired entries
	// only when the cache exceeds a reasonable size threshold.
	const sweepThreshold = 100
	c.mu.Lock()
	if len(c.entries) >= sweepThreshold {
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
	c.entries[userID] = adminCacheEntry{
		isAdmin:   isAdmin,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()

	return isAdmin, nil
}

func AdminMiddleware(jwt *JWTManager, checker AdminChecker, apiKeys APIKeyValidator) func(http.Handler) http.Handler {
	// Wrap the provided AdminChecker with a short-lived cache to avoid
	// repeated DB lookups for high-frequency admin endpoints (e.g., /asynqmon/).
	const adminCacheTTL = 5 * time.Second
	cachedChecker := newCachingAdminChecker(checker, adminCacheTTL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, source, _ := extractToken(r)
			if token == "" {
				slog.InfoContext(r.Context(), "admin middleware: no token found")
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			userID, err := resolveUser(r.Context(), token, source, jwt, apiKeys)
			if err != nil {
				if errors.Is(err, ErrInvalidToken) {
					slog.InfoContext(r.Context(), "admin middleware: invalid token", slog.Any(otelkeys.Error, err))
					jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				} else {
					slog.ErrorContext(r.Context(), "admin middleware: failed to resolve user", slog.Any(otelkeys.Error, err))
					jsonError(w, http.StatusInternalServerError, "internal authentication error")
				}
				return
			}

			isAdmin, err := cachedChecker.IsAdmin(r.Context(), userID)
			if err != nil {
				slog.ErrorContext(r.Context(), "admin middleware: failed to check admin status", slog.Any(otelkeys.Error, err))
				jsonError(w, http.StatusInternalServerError, "failed to verify permissions")
				return
			}
			if !isAdmin {
				slog.InfoContext(r.Context(), "admin middleware: non-admin access denied", slog.String(otelkeys.UserID, userID))
				jsonError(w, http.StatusForbidden, "admin access required")
				return
			}

			slog.DebugContext(r.Context(), "admin authentication successful", slog.String(otelkeys.UserID, userID))
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContextWithUserID returns a new context with the given user ID set.
// This is intended for use in tests and server-side tooling that bypasses
// the JWT middleware.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
