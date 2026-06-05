package middleware

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otel"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before delegating to the underlying
// ResponseWriter so that it can be included in the access log.
func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write defaults the captured status code to 200 OK when WriteHeader was not
// called before the first write, then delegates to the underlying ResponseWriter.
func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Flush delegates to the underlying ResponseWriter if it implements http.Flusher.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack delegates to the underlying ResponseWriter if it implements http.Hijacker.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

// Push delegates to the underlying ResponseWriter if it implements http.Pusher.
func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	p, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

// LoggingMiddleware returns an HTTP middleware that emits a structured debug
// log entry at the start of each request (method, URL, remote address, user
// agent, request ID, and user ID) and a second entry upon completion
// (method, URL, status code, elapsed duration, request ID, and user ID).
func LoggingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		userID := auth.UserIDFromContext(r.Context())
		requestURL := redactRequestURL(r.URL)
		slog.DebugContext(
			r.Context(),
			"Incoming request",
			slog.String(otelkeys.Method, r.Method),
			slog.String(otelkeys.URL, requestURL),
			slog.String(otelkeys.RemoteAddr, r.RemoteAddr),
			slog.String(otelkeys.UserAgent, r.UserAgent()),
			slog.String(otelkeys.RequestID, GetRequestID(r.Context())),
			slog.String(otelkeys.UserID, userID),
		)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		slog.DebugContext(
			r.Context(),
			"Request completed",
			slog.String(otelkeys.Method, r.Method),
			slog.String(otelkeys.URL, requestURL),
			slog.Int(otelkeys.StatusCode, rec.statusCode),
			slog.Duration(otelkeys.Duration, time.Since(start)),
			slog.String(otelkeys.RequestID, GetRequestID(r.Context())),
			slog.String(otelkeys.UserID, userID),
		)
	}

	return http.HandlerFunc(fn)
}

func redactRequestURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	clone := *u
	clone.Path = otel.SanitizePathForTelemetry(u.Path)
	if u.RawPath != "" {
		clone.RawPath = otel.SanitizePathForTelemetry(u.RawPath)
	}
	return clone.String()
}
