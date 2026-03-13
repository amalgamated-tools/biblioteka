package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "missing authorization header")
}

func TestMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Basic sometoken")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid authorization format")
}

func TestMiddleware_InvalidToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer notavalidtoken")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestMiddleware_ValidToken(t *testing.T) {
	jm, _ := NewJWTManager("secret", time.Hour)
	mw := Middleware(jm)

	token, _ := jm.CreateToken("user-abc")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "user-abc" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "user-abc")
	}
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	id := UserIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty user ID, got %q", id)
	}
}

func TestContextWithUserID(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithUserID(ctx, "test-user-123")
	got := UserIDFromContext(ctx)
	if got != "test-user-123" {
		t.Errorf("UserIDFromContext() = %q, want %q", got, "test-user-123")
	}
}

// assertJSONError checks that a JSON body contains the given error message.
func assertJSONError(t *testing.T, body []byte, wantMsg string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if resp["error"] != wantMsg {
		t.Errorf("error message = %q, want %q", resp["error"], wantMsg)
	}
}
