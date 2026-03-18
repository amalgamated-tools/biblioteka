package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDHandler_GeneratesID(t *testing.T) {
	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = GetRequestID(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	RequestIDHandler(next).ServeHTTP(w, r)

	if gotID == "" {
		fail(t, "expected a non-empty request ID to be generated")
	}
	if w.Header().Get(RequestID) == "" {
		fail(t, "expected X-Request-ID response header to be set")
	}
	if w.Header().Get(RequestID) != gotID {
		failf(t, "response header X-Request-ID = %q, context request ID = %q; want them equal",
			w.Header().Get(RequestID), gotID)
	}
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

	if gotID != existingID {
		failf(t, "GetRequestID() = %q, want %q", gotID, existingID)
	}
	if w.Header().Get(RequestID) != existingID {
		failf(t, "response X-Request-ID = %q, want %q", w.Header().Get(RequestID), existingID)
	}
}

func TestWithRequestID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithRequestID(r.Context(), "test-id-123")

	got := GetRequestID(ctx)
	if got != "test-id-123" {
		failf(t, "GetRequestID() = %q, want %q", got, "test-id-123")
	}
}

func TestForward_SetsHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithRequestID(r.Context(), "forwarded-id")
	r = r.WithContext(ctx)

	Forward(r)

	if r.Header.Get(RequestID) != "forwarded-id" {
		failf(t, "Header X-Request-ID = %q, want %q", r.Header.Get(RequestID), "forwarded-id")
	}
}

func TestForward_NoIDDoesNothing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// No request ID in context

	Forward(r)

	if r.Header.Get(RequestID) != "" {
		failf(t, "expected empty X-Request-ID header, got %q", r.Header.Get(RequestID))
	}
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

	if !called {
		fail(t, "next handler was not called")
	}
	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
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

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLoggingMiddleware_ImplicitOKOnWrite(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	r := httptest.NewRequest(http.MethodGet, "/write-only", nil)
	w := httptest.NewRecorder()
	LoggingMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- statusRecorder unit tests ---

func TestStatusRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)

	if rec.statusCode != http.StatusCreated {
		failf(t, "statusCode = %d, want %d", rec.statusCode, http.StatusCreated)
	}
	if w.Code != http.StatusCreated {
		failf(t, "underlying Code = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestStatusRecorder_WriteWithoutHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w}

	n, err := rec.Write([]byte("data"))

	if err != nil {
		failNowf(t, "Write error: %v", err)
	}
	if n != 4 {
		failf(t, "Write returned %d, want 4", n)
	}
	if rec.statusCode != http.StatusOK {
		failf(t, "statusCode = %d, want %d", rec.statusCode, http.StatusOK)
	}
}

func TestStatusRecorder_Flush_WithFlusher(t *testing.T) {
	inner := httptest.NewRecorder()
	fr := &flusherRecorder{ResponseRecorder: inner}
	rec := &statusRecorder{ResponseWriter: fr}

	rec.Flush()

	if !fr.flushed {
		fail(t, "expected Flush to be delegated to underlying flusher")
	}
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

	if conn != nil {
		fail(t, "expected nil conn")
	}
	if buf != nil {
		fail(t, "expected nil buf")
	}
	if !errors.Is(err, http.ErrNotSupported) {
		failf(t, "err = %v, want %v", err, http.ErrNotSupported)
	}
}

func TestStatusRecorder_Push_NotSupported(t *testing.T) {
	pw := &plainResponseWriter{header: make(http.Header)}
	rec := &statusRecorder{ResponseWriter: pw}

	err := rec.Push("/resource", nil)

	if !errors.Is(err, http.ErrNotSupported) {
		failf(t, "err = %v, want %v", err, http.ErrNotSupported)
	}
}
