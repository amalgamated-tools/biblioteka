package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// KoboTokenResult holds the fields needed by the Kobo token auth middleware.
type KoboTokenResult struct {
	UserID string
}

// HashKoboToken returns the hex-encoded SHA-256 hash of the given Kobo token.
// SHA-256 is appropriate here because tokens are high-entropy random values.
func HashKoboToken(token string) string {
	return hashHighEntropyToken(token)
}

// KoboTokenChecker is implemented by types that can look up Kobo tokens by value.
type KoboTokenChecker interface {
	GetKoboTokenByToken(ctx context.Context, token string) (*KoboTokenResult, error)
}

const koboTokenKey contextKey = "koboToken"

// KoboTokenFromContext returns the Kobo token value stored by the middleware.
func KoboTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(koboTokenKey).(string)
	return v
}

// writeKoboJSONError writes an empty JSON object with the given status,
// matching the Kobo device's expected response format.
func writeKoboJSONError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{})
}

// KoboTokenAuthMiddleware returns an HTTP middleware that extracts and validates
// a Kobo device token from the URL path. It expects paths of the form
// /kobo/{token}/... and rewrites the request path to strip the /kobo/{token}
// prefix so downstream handlers see paths like /v1/library/sync.
func KoboTokenAuthMiddleware(checker KoboTokenChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strip /kobo/ prefix and extract token.
			rest := strings.TrimPrefix(r.URL.Path, "/kobo/")
			slashIdx := strings.Index(rest, "/")
			var tokenValue, subPath string
			if slashIdx < 0 {
				tokenValue = rest
				subPath = "/"
			} else {
				tokenValue = rest[:slashIdx]
				subPath = rest[slashIdx:]
			}

			if tokenValue == "" {
				writeKoboJSONError(w, http.StatusOK)
				return
			}

			koboToken, err := checker.GetKoboTokenByToken(r.Context(), tokenValue)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeKoboJSONError(w, http.StatusUnauthorized)
					return
				}
				slog.ErrorContext(r.Context(), "failed to look up kobo token", slog.Any(otelkeys.Error, err))
				writeKoboJSONError(w, http.StatusInternalServerError)
				return
			}

			slog.DebugContext(r.Context(), "kobo device request",
				slog.String(otelkeys.UserID, koboToken.UserID),
				slog.String(otelkeys.Path, subPath),
			)

			// Inject user ID and token value into the request context.
			ctx := ContextWithUserID(r.Context(), koboToken.UserID)
			ctx = context.WithValue(ctx, koboTokenKey, tokenValue)
			r = r.WithContext(ctx)

			// Rewrite path so the sub-mux dispatches on the stripped path.
			r.URL.Path = subPath
			r.URL.RawPath = ""

			next.ServeHTTP(w, r)
		})
	}
}
