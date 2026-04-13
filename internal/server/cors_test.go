package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// nextHandlerSentinel is a handler that records whether it was called.
type nextHandlerSentinel struct {
	called bool
}

func (h *nextHandlerSentinel) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.called = true
	w.WriteHeader(http.StatusOK)
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodPost, "/api/books/capture", nil)
	req.Header.Set("Origin", "moz-extension://abc123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, next.called)
	require.Equal(t, "moz-extension://abc123", rec.Header().Get("Access-Control-Allow-Origin"))
	require.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
	require.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
	require.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodPost, "/api/books/capture", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, next.called)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "disallowed origin must not receive CORS header")
}

func TestCORSMiddleware_NoOrigin(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodPost, "/api/books/capture", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, next.called)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/books/capture", nil)
	req.Header.Set("Origin", "moz-extension://abc123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.False(t, next.called, "OPTIONS preflight must be short-circuited before next handler")
	require.Equal(t, "moz-extension://abc123", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_EmptyAllowedOrigins(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware(nil)(next)

	req := httptest.NewRequest(http.MethodPost, "/api/books/capture", nil)
	req.Header.Set("Origin", "moz-extension://abc123")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, next.called)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "empty allowed list means no CORS headers")
}

func TestCORSMiddleware_PreflightDisallowedOrigin(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/books/capture", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.True(t, next.called, "OPTIONS from disallowed origin should pass through to next handler")
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_PreflightNoOrigin(t *testing.T) {
	next := &nextHandlerSentinel{}
	mw := corsMiddleware([]string{"moz-extension://abc123"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/books/capture", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	require.True(t, next.called, "OPTIONS with no origin should pass through to next handler")
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_MultipleAllowedOrigins(t *testing.T) {
	origins := []string{
		"moz-extension://abc123",
		"chrome-extension://xyz789",
	}
	next := &nextHandlerSentinel{}
	mw := corsMiddleware(origins)(next)

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/books/capture", nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			require.Equal(t, origin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
