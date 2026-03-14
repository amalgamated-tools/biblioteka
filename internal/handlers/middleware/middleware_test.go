package middleware

import (
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
		t.Error("expected a non-empty request ID to be generated")
	}
	if w.Header().Get(RequestID) == "" {
		t.Error("expected X-Request-ID response header to be set")
	}
	if w.Header().Get(RequestID) != gotID {
		t.Errorf("response header X-Request-ID = %q, context request ID = %q; want them equal",
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
		t.Errorf("GetRequestID() = %q, want %q", gotID, existingID)
	}
	if w.Header().Get(RequestID) != existingID {
		t.Errorf("response X-Request-ID = %q, want %q", w.Header().Get(RequestID), existingID)
	}
}

func TestGetRequestID_NilContext(t *testing.T) {
	id := GetRequestID(t.Context()) //nolint:staticcheck
	if id != "" {
		t.Errorf("GetRequestID(nil) = %q, want empty string", id)
	}
}

func TestWithRequestID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithRequestID(r.Context(), "test-id-123")

	got := GetRequestID(ctx)
	if got != "test-id-123" {
		t.Errorf("GetRequestID() = %q, want %q", got, "test-id-123")
	}
}

func TestForward_SetsHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithRequestID(r.Context(), "forwarded-id")
	r = r.WithContext(ctx)

	Forward(r)

	if r.Header.Get(RequestID) != "forwarded-id" {
		t.Errorf("Header X-Request-ID = %q, want %q", r.Header.Get(RequestID), "forwarded-id")
	}
}

func TestForward_NoIDDoesNothing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// No request ID in context

	Forward(r)

	if r.Header.Get(RequestID) != "" {
		t.Errorf("expected empty X-Request-ID header, got %q", r.Header.Get(RequestID))
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
		t.Error("next handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
