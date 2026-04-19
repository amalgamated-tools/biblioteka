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
		return "", "", ErrNotFound
	}
	return entry.userID, entry.keyID, nil
}

func (m *mockAPIKeyValidator) TouchAPIKeyLastUsed(_ context.Context, id string) error {
	m.touched = append(m.touched, id)
	return nil
}

// The remaining APIKeyStore methods are unused by middleware tests; they exist
// only so *mockAPIKeyValidator satisfies the full goauth interface.
func (m *mockAPIKeyValidator) CreateAPIKey(_ context.Context, _, _, _, _ string) (*APIKey, error) {
	return nil, sql.ErrNoRows
}

func (m *mockAPIKeyValidator) ListAPIKeysByUser(_ context.Context, _ string) ([]APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyValidator) FindAPIKeyByIDAndUser(_ context.Context, _, _ string) (*APIKey, error) {
	return nil, sql.ErrNoRows
}

func (m *mockAPIKeyValidator) DeleteAPIKey(_ context.Context, _, _ string) error {
	return sql.ErrNoRows
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour, "test")
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, testConfig(), validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "apikey-user", gotUserID)

	require.Len(t, validator.touched, 1)
	require.Equal(t, "key-1", validator.touched[0])
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour, "test")
	require.NoError(t, err, "NewJWTManager() error")
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{},
	}
	mw := Middleware(jm, testConfig(), validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer bib_invalidkey00000000000000000000")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestMiddleware_APIKeyViaCookieRejected(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour, "test")
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "apikey-user", keyID: "key-1"},
		},
	}
	mw := Middleware(jm, testConfig(), validator)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookieName, Value: apiKey})
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assertJSONError(t, w.Body.Bytes(), "invalid or expired token")
}

func TestAdminMiddleware_ValidAPIKey(t *testing.T) {
	jm, err := NewJWTManager("secret", time.Hour, "test")
	require.NoError(t, err, "NewJWTManager() error")
	apiKey := "bib_abcdef1234567890abcdef1234567890"
	keyHash := HashAPIKey(apiKey)
	validator := &mockAPIKeyValidator{
		keys: map[string]struct{ userID, keyID string }{
			keyHash: {userID: "admin-user", keyID: "key-2"},
		},
	}
	checker := &mockAdminChecker{admins: map[string]bool{"admin-user": true}}
	mw := AdminMiddleware(jm, checker, testConfig(), validator)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "admin-user", gotUserID)
}
