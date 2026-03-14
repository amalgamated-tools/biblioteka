package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
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

// Middleware returns an HTTP middleware that validates the JWT from the
// Authorization header and injects the user ID into the request context.
func Middleware(jwt *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				slog.InfoContext(r.Context(), "missing authorization header")
				jsonError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			if token == header {
				slog.InfoContext(r.Context(), "invalid authorization format")
				jsonError(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			claims, err := jwt.ValidateToken(token)
			if err != nil {
				slog.InfoContext(r.Context(), "invalid or expired token", slog.Any("error", err))
				jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			slog.DebugContext(r.Context(), "authentication successful", slog.String("user_id", claims.UserID))
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
	IsAdmin(userID string) (bool, error)
}

// tokenCookieName is the name of the HttpOnly cookie used as a fallback for
// browser-navigated endpoints that cannot send an Authorization header.
const tokenCookieName = "biblioteka_token"

// TokenCookieName returns the cookie name used for browser-based auth.
func TokenCookieName() string { return tokenCookieName }

// extractToken reads a JWT from the Authorization header, falling back to the
// biblioteka_token cookie for browser-navigated requests.
func extractToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		if token := strings.TrimPrefix(header, "Bearer "); token != header {
			return token
		}
	}
	if c, err := r.Cookie(tokenCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

// AdminMiddleware returns an HTTP middleware that validates a JWT (from header
// or cookie) and verifies the user is an admin before allowing the request.
func AdminMiddleware(jwt *JWTManager, checker AdminChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				slog.InfoContext(r.Context(), "admin middleware: no token found")
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			claims, err := jwt.ValidateToken(token)
			if err != nil {
				slog.InfoContext(r.Context(), "admin middleware: invalid token", slog.Any("error", err))
				jsonError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			isAdmin, err := checker.IsAdmin(claims.UserID)
			if err != nil {
				slog.ErrorContext(r.Context(), "admin middleware: failed to check admin status", slog.Any("error", err))
				jsonError(w, http.StatusInternalServerError, "failed to verify permissions")
				return
			}
			if !isAdmin {
				slog.InfoContext(r.Context(), "admin middleware: non-admin access denied", slog.String("user_id", claims.UserID))
				jsonError(w, http.StatusForbidden, "admin access required")
				return
			}

			slog.DebugContext(r.Context(), "admin authentication successful", slog.String("user_id", claims.UserID))
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
