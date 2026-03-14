package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
)

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		userID := auth.UserIDFromContext(r.Context())
		slog.DebugContext(
			r.Context(),
			"Incoming request",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("request_id", GetRequestID(r.Context())),
			slog.String("user_id", userID),
		)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		slog.DebugContext(
			r.Context(),
			"Request completed",
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.Int("status", rec.statusCode),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", GetRequestID(r.Context())),
			slog.String("user_id", userID),
		)
	}

	return http.HandlerFunc(fn)
}
