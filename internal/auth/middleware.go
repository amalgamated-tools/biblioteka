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

// ContextWithUserID returns a new context with the given user ID set.
// This is intended for use in tests and server-side tooling that bypasses
// the JWT middleware.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
