package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/require"
)

// mockOPDSChecker implements OPDSCredentialChecker for testing.
type mockOPDSChecker struct {
	creds map[string]*ProtocolCredentialResult
	err   error
}

func (m *mockOPDSChecker) GetOPDSCredential(_ context.Context, username string) (*ProtocolCredentialResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	cred, ok := m.creds[username]
	if !ok {
		return nil, errors.New("not found")
	}
	return cred, nil
}

func newOPDSCheckerWithUser(t *testing.T, username, password, userID string) *mockOPDSChecker {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err, "bcrypt hash")
	return &mockOPDSChecker{
		creds: map[string]*ProtocolCredentialResult{
			username: {UserID: userID, PasswordHash: string(hash)},
		},
	}
}

func TestOPDSBasicAuth_MissingCredentials(t *testing.T) {
	checker := &mockOPDSChecker{creds: map[string]*ProtocolCredentialResult{}}
	mw := OPDSBasicAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	got := w.Header().Get("WWW-Authenticate")
	require.Equal(t, `Basic realm="Biblioteka OPDS"`, got)
	require.Contains(t, w.Body.String(), "authentication required")
}

func TestOPDSBasicAuth_UnknownUsername(t *testing.T) {
	checker := &mockOPDSChecker{creds: map[string]*ProtocolCredentialResult{}}
	mw := OPDSBasicAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.SetBasicAuth("unknown", "password")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "invalid credentials")
}

func TestOPDSBasicAuth_WrongPassword(t *testing.T) {
	checker := newOPDSCheckerWithUser(t, "alice", "correct-password", "user-1")
	mw := OPDSBasicAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.SetBasicAuth("alice", "wrong-password")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "invalid credentials")
}

func TestOPDSBasicAuth_Success(t *testing.T) {
	checker := newOPDSCheckerWithUser(t, "alice", "correct-password", "user-1")
	mw := OPDSBasicAuthMiddleware(checker)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.SetBasicAuth("alice", "correct-password")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "user-1", gotUserID)
}

func TestOPDSBasicAuth_UsernameLowercased(t *testing.T) {
	checker := newOPDSCheckerWithUser(t, "alice", "password123", "user-1")
	mw := OPDSBasicAuthMiddleware(checker)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.SetBasicAuth("ALICE", "password123")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "user-1", gotUserID)
}

func TestOPDSBasicAuth_EmptyUsername(t *testing.T) {
	checker := &mockOPDSChecker{creds: map[string]*ProtocolCredentialResult{}}
	mw := OPDSBasicAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.SetBasicAuth("", "password")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	require.False(t, called)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOPDSBasicAuth_XMLErrorResponse(t *testing.T) {
	checker := &mockOPDSChecker{creds: map[string]*ProtocolCredentialResult{}}
	mw := OPDSBasicAuthMiddleware(checker)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	ct := w.Header().Get("Content-Type")
	require.Contains(t, ct, "application/atom+xml")
	body := w.Body.String()
	require.Contains(t, body, "<feed")
}
