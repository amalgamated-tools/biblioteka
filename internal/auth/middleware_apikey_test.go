package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockAPIKeyValidator implements APIKeyValidator for testing.
type mockAPIKeyValidator struct {
	keys    map[string]struct{ userID, keyID string } // keyHash -> (userID, keyID)
	touched []string                                  // keyIDs passed to TouchAPIKeyLastUsed
	err     error
}

func (m *mockAPIKeyValidator) ValidateAPIKey(_ context.Context, keyHash string) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	entry, ok := m.keys[keyHash]
	if !ok {
		return "", "", sql.ErrNoRows
	}
	return entry.userID, entry.keyID, nil
}

func (m *mockAPIKeyValidator) TouchAPIKeyLastUsed(_ context.Context, id string) error {
	m.touched = append(m.touched, id)
	return nil
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "apikey-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "apikey-user")
	}

	if len(validator.touched) != 1 || validator.touched[0] != "key-1" {
		t.Errorf("TouchAPIKeyLastUsed called with %v, want [key-1]", validator.touched)
	}
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{},
	}
	mw := Middleware(jm, validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer bib_invalidkey00000000000000000000")
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

func TestMiddleware_APIKeyViaCookieRejected(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: apiKey})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called for API key via cookie")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestAdminMiddleware_ValidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour)
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "admin-user", keyID: "key-2"},
		},
	}
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if gotUserID != "admin-user" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "admin-user")
	}
}
