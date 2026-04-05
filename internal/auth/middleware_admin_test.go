package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockAdminChecker implements AdminChecker for testing.
type mockAdminChecker struct {
	admins map[string]bool
	err    error
}

func (m *mockAdminChecker) IsAdmin(_ context.Context, userID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.admins[userID], nil
}

func TestAdminMiddleware_NoToken(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{admins: map[string]bool{}}
	mw := AdminMiddleware(jm, checker, nil)

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
	assertJSONError(t, w.Body.Bytes(), "authentication required")
}

func TestAdminMiddleware_InvalidToken(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{admins: map[string]bool{}}
	mw := AdminMiddleware(jm, checker, nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer badtoken")
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

func TestAdminMiddleware_NonAdminUser(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, err := jm.CreateToken(t.Context(), "regular-user")
	require.NoError(t, err, "CreateToken() error")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called for non-admin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "admin access required")
}

func TestAdminMiddleware_AdminUser(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, err := jm.CreateToken(t.Context(), "admin-user")
	require.NoError(t, err, "CreateToken() error")

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
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}

func TestAdminMiddleware_AdminViaCookie(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, nil)

	token, err := jm.CreateToken(t.Context(), "admin-user")
	require.NoError(t, err, "CreateToken() error")

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: token})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}

func TestAdminMiddleware_CheckerError(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	checker := &mockAdminChecker{err: errors.New("db down")}
	mw := AdminMiddleware(jm, checker, nil)

	token, err := jm.CreateToken(t.Context(), "some-user")
	require.NoError(t, err, "CreateToken() error")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called on checker error")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "failed to verify permissions")
}
