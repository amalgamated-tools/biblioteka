package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

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

// ExtractToken tries to read a JWT from the Authorization header first,
// then falls back to the "biblioteka_token" cookie. It returns the token and
// an optional reason describing why no token could be extracted.
// This exported form allows non-middleware callers (e.g. OPDS handler) to
// reuse the same token-extraction logic without going through the full middleware.
func ExtractToken(r *http.Request) (string, string) {
	return extractToken(r)
}

// extractToken tries to read a JWT from the Authorization header first,
// then falls back to the "biblioteka_token" cookie. It returns the token and
// an optional reason describing why no token could be extracted.
func extractToken(r *http.Request) (string, string) {
	if header := r.Header.Get("Authorization"); header != "" {
		header = strings.TrimSpace(header)
		if after, ok := strings.CutPrefix(header, "Bearer "); ok {
			token := strings.TrimSpace(after)
			if token != "" {
				return token, ""
			}
		}
		// Non-Bearer or empty Bearer — fall through to cookie.
	}

	// Fallback to cookie-based authentication.
	if c, err := r.Cookie(tokenCookieName); err == nil && c.Value != "" {
		return c.Value, ""
	}

	return "", "missing token"
}

// Middleware returns an HTTP middleware that validates the JWT from the
// Authorization header or biblioteka_token cookie and injects the user ID into
// the request context.
func Middleware(jwt *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, reason := extractToken(r)
			if token == "" {
				if reason != "" {
					slog.InfoContext(r.Context(), "authentication required", slog.String(otelkeys.Reason, reason))
				} else {
					slog.InfoContext(r.Context(), "authentication required")
				}
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			claims, err := jwt.ValidateToken(r.Context(), token)
			if err != nil {
				slog.InfoContext(r.Context(), "invalid or expired token", slog.Any(otelkeys.Error, err))
				jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			slog.DebugContext(r.Context(), "authentication successful", slog.String(otelkeys.UserID, claims.UserID))
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
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
		return false, err
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

func AdminMiddleware(jwt *JWTManager, checker AdminChecker) func(http.Handler) http.Handler {
	// Wrap the provided AdminChecker with a short-lived cache to avoid
	// repeated DB lookups for high-frequency admin endpoints (e.g., /asynqmon/).
	const adminCacheTTL = 5 * time.Second
	cachedChecker := newCachingAdminChecker(checker, adminCacheTTL)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, _ := extractToken(r)
			if token == "" {
				slog.InfoContext(r.Context(), "admin middleware: no token found")
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			claims, err := jwt.ValidateToken(r.Context(), token)
			if err != nil {
				slog.InfoContext(r.Context(), "admin middleware: invalid token", slog.Any(otelkeys.Error, err))
				jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			isAdmin, err := cachedChecker.IsAdmin(r.Context(), claims.UserID)
			if err != nil {
				slog.ErrorContext(r.Context(), "admin middleware: failed to check admin status", slog.Any(otelkeys.Error, err))
				jsonError(w, http.StatusInternalServerError, "failed to verify permissions")
				return
			}
			if !isAdmin {
				slog.InfoContext(r.Context(), "admin middleware: non-admin access denied", slog.String(otelkeys.UserID, claims.UserID))
				jsonError(w, http.StatusForbidden, "admin access required")
				return
			}

			slog.DebugContext(r.Context(), "admin authentication successful", slog.String(otelkeys.UserID, claims.UserID))
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
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
