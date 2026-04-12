package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestIDHandler_GeneratesID(t *testing.T) {
	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	RequestIDHandler(next).ServeHTTP(w, r)

	require.NotEqual(t, "", gotID)
	require.NotEqual(t, "", w.Header().Get(RequestID))
	require.Equal(t, gotID, w.Header().Get(RequestID), "response header X-Request-ID and context request ID should be equal")
}

func TestRequestIDHandler_UsesExistingID(t *testing.T) {
	existingID := "my-custom-request-id"
	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(RequestID, existingID)
	w := httptest.NewRecorder()
	RequestIDHandler(next).ServeHTTP(w, r)

	require.Equal(t, existingID, gotID)
	require.Equal(t, existingID, w.Header().Get(RequestID))
}

func TestWithRequestID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithRequestID(r.Context(), "test-id-123")

	got := GetRequestID(ctx)
	require.Equal(t, "test-id-123", got)
}

func TestLoggingMiddleware_CallsNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, r)

	require.True(t, called)
	require.Equal(t, http.StatusOK, w.Code)
}

// --- Mock types for statusRecorder tests ---

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
	f.ResponseRecorder.Flush()
}

type plainResponseWriter struct {
	code   int
	header http.Header
	body   []byte
}

func (w *plainResponseWriter) Header() http.Header { return w.header }
func (w *plainResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}
func (w *plainResponseWriter) WriteHeader(code int) { w.code = code }

// --- LoggingMiddleware tests ---

func TestLoggingMiddleware_PreservesStatus(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLoggingMiddleware_ImplicitOKOnWrite(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	r := httptest.NewRequest(http.MethodGet, "/write-only", nil)
	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

// --- statusRecorder unit tests ---

func TestStatusRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)

	require.Equal(t, http.StatusCreated, rec.statusCode)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestStatusRecorder_WriteWithoutHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w}

	n, err := rec.Write([]byte("data"))
	require.NoError(t, err, "Write error")
	require.Equal(t, 4, n)
	require.Equal(t, http.StatusOK, rec.statusCode)
}

func TestStatusRecorder_Flush_WithFlusher(t *testing.T) {
	inner := httptest.NewRecorder()
	fr := &flusherRecorder{ResponseRecorder: inner}
	rec := &statusRecorder{ResponseWriter: fr}

	rec.Flush()

	require.True(t, fr.flushed)
}

func TestStatusRecorder_Flush_WithoutFlusher(t *testing.T) {
	pw := &plainResponseWriter{header: make(http.Header)}
	rec := &statusRecorder{ResponseWriter: pw}

	// Should not panic
	rec.Flush()
}

func TestStatusRecorder_Hijack_NotSupported(t *testing.T) {
	pw := &plainResponseWriter{header: make(http.Header)}
	rec := &statusRecorder{ResponseWriter: pw}

	conn, buf, err := rec.Hijack()

	require.Nil(t, conn)
	require.Nil(t, buf)
	require.ErrorIs(t, err, http.ErrNotSupported)
}

func TestStatusRecorder_Push_NotSupported(t *testing.T) {
	pw := &plainResponseWriter{header: make(http.Header)}
	rec := &statusRecorder{ResponseWriter: pw}

	err := rec.Push("/resource", nil)

	require.ErrorIs(t, err, http.ErrNotSupported)
}

// --- SecurityHeadersMiddleware tests ---

func TestSecurityHeadersMiddleware_SetsHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	NewSecurityHeadersMiddleware(SecurityHeadersConfig{})(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, globalCSP, w.Header().Get("Content-Security-Policy"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeadersMiddleware_HeadersCanBeOverridden(t *testing.T) {
	overrideCSP := "default-src 'none';"
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate a route handler that overrides the global CSP.
		w.Header().Set("Content-Security-Policy", overrideCSP)
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	w := httptest.NewRecorder()
	NewSecurityHeadersMiddleware(SecurityHeadersConfig{})(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, overrideCSP, w.Header().Get("Content-Security-Policy"))
	// Other security headers should still be set by the global middleware.
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeadersMiddleware_CallsNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	NewSecurityHeadersMiddleware(SecurityHeadersConfig{})(next).ServeHTTP(w, r)

	require.True(t, called)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestSecurityHeadersMiddleware_HSTS_SecureCookiesTrue(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	NewSecurityHeadersMiddleware(SecurityHeadersConfig{SecureCookies: true})(next).ServeHTTP(w, r)

	require.Equal(t, hsts, w.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersMiddleware_HSTS_SecureCookiesFalse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	NewSecurityHeadersMiddleware(SecurityHeadersConfig{SecureCookies: false})(next).ServeHTTP(w, r)

	require.Empty(t, w.Header().Get("Strict-Transport-Security"))
}
