package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// mockKOSyncChecker implements KOSyncCredentialChecker for testing.
type mockKOSyncChecker struct {
	creds map[string]*KOSyncCredentialResult
	err   error
}

func (m *mockKOSyncChecker) GetKOSyncCredential(_ context.Context, username string) (*KOSyncCredentialResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	cred, ok := m.creds[username]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return cred, nil
}

// newKOSyncCheckerWithUser creates a mock checker with a single user whose
// stored hash is bcrypt(authKey).  In production the authKey would be
// hex(MD5(password)), mirroring what KOReader sends as x-auth-key.
func newKOSyncCheckerWithUser(t *testing.T, username, authKey, userID string) *mockKOSyncChecker {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(authKey), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	return &mockKOSyncChecker{
		creds: map[string]*KOSyncCredentialResult{
			username: {UserID: userID, PasswordHash: string(hash)},
		},
	}
}

func TestKOSyncHeaderAuth_MissingHeaders(t *testing.T) {
	checker := &mockKOSyncChecker{creds: map[string]*KOSyncCredentialResult{}}
	mw := KOSyncHeaderAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), "Unauthorized") {
		t.Errorf("body should contain 'Unauthorized', got %q", w.Body.String())
	}
}

func TestKOSyncHeaderAuth_MissingAuthKey(t *testing.T) {
	checker := &mockKOSyncChecker{creds: map[string]*KOSyncCredentialResult{}}
	mw := KOSyncHeaderAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "alice")
	// x-auth-key intentionally omitted
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestKOSyncHeaderAuth_UnknownUsername(t *testing.T) {
	checker := &mockKOSyncChecker{creds: map[string]*KOSyncCredentialResult{}}
	mw := KOSyncHeaderAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "nobody")
	r.Header.Set(kosyncAuthKeyHeader, "somekey")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestKOSyncHeaderAuth_WrongKey(t *testing.T) {
	checker := newKOSyncCheckerWithUser(t, "alice", "correct-key", "user-1")
	mw := KOSyncHeaderAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "alice")
	r.Header.Set(kosyncAuthKeyHeader, "wrong-key")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestKOSyncHeaderAuth_Success(t *testing.T) {
	checker := newKOSyncCheckerWithUser(t, "alice", "correct-key", "user-1")
	mw := KOSyncHeaderAuthMiddleware(checker)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "alice")
	r.Header.Set(kosyncAuthKeyHeader, "correct-key")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotUserID != "user-1" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "user-1")
	}
}

func TestKOSyncHeaderAuth_UsernameLowercased(t *testing.T) {
	checker := newKOSyncCheckerWithUser(t, "alice", "mykey", "user-1")
	mw := KOSyncHeaderAuthMiddleware(checker)

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "ALICE")
	r.Header.Set(kosyncAuthKeyHeader, "mykey")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotUserID != "user-1" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "user-1")
	}
}

func TestKOSyncHeaderAuth_DBError(t *testing.T) {
	checker := &mockKOSyncChecker{err: errors.New("connection refused")}
	mw := KOSyncHeaderAuthMiddleware(checker)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r.Header.Set(kosyncAuthUserHeader, "alice")
	r.Header.Set(kosyncAuthKeyHeader, "somekey")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if called {
		t.Error("next handler should not have been called")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
