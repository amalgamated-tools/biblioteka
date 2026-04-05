package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

// inMemoryCredStore is a simple in-memory credential store used to test the
// generic credential helpers without coupling to any specific protocol schema.
type inMemoryCredStore struct {
	mu    sync.Mutex
	creds map[string]credentialEntity // keyed by userID
}

func newInMemoryCredStore() *inMemoryCredStore {
	return &inMemoryCredStore{creds: make(map[string]credentialEntity)}
}

func (s *inMemoryCredStore) getByUserID(_ context.Context, userID string) (credentialEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.creds[userID]
	if !ok {
		return credentialEntity{}, sql.ErrNoRows
	}
	return c, nil
}

func (s *inMemoryCredStore) upsert(_ context.Context, userID, username, hash string) (credentialEntity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := db.Timestamp{}
	c := credentialEntity{
		ID:        "cred-" + userID,
		Username:  username,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.creds[userID] = c
	return c, nil
}

func (s *inMemoryCredStore) del(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.creds[userID]; !ok {
		return sql.ErrNoRows
	}
	delete(s.creds, userID)
	return nil
}

var errTestConflict = errors.New("test username conflict")

// makeTestCredOps builds a minimal credentialOps backed by in-memory closures,
// decoupled from any protocol schema.
func makeTestCredOps(t *testing.T) (credentialOps, string) {
	t.Helper()
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "CredUser", "cred@example.com", "password1")
	require.NoError(t, err, "create user")

	store := newInMemoryCredStore()

	ops := credentialOps{
		db:              d,
		protocol:        "TestProto",
		auditEntityType: "testproto_credential",
		auditUpsert:     "testproto_credential.updated",
		auditDelete:     "testproto_credential.deleted",
		errConflict:     errTestConflict,
		getByUserID:     store.getByUserID,
		upsert:          store.upsert,
		del:             store.del,
	}
	return ops, user.ID
}

// ---- handleCredentials dispatch ----

func TestHandleCredentials_MethodNotAllowed(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ---- getCredential ----

func TestGetCredential_NotFound(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	r := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestGetCredential_Error(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:       d,
		protocol: "TestProto",
		getByUserID: func(_ context.Context, _ string) (credentialEntity, error) {
			return credentialEntity{}, errors.New("db error")
		},
	}

	r := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetCredential_Success(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	// Create a credential first via PUT.
	body := `{"username":"myuser","password":"validpassword"}`
	putR := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	putR = withUserID(putR, userID)
	putW := httptest.NewRecorder()
	handleCredentials(ops, putW, putR)
	require.Equal(t, http.StatusOK, putW.Code)

	// Now GET it.
	r := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Username != "myuser" {
		t.Errorf("username = %q, want %q", resp.Username, "myuser")
	}
}

// ---- upsertCredential ----

func TestUpsertCredential_EmptyUsername(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	body := `{"username":"","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpsertCredential_UsernameTooLong(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	longUsername := strings.Repeat("a", maxUsernameLen+1)
	body := `{"username":"` + longUsername + `","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpsertCredential_PasswordTooShort(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	body := `{"username":"myuser","password":"abc"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpsertCredential_InvalidJSON(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpsertCredential_Conflict(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:          d,
		protocol:    "TestProto",
		errConflict: errTestConflict,
		upsert: func(_ context.Context, _, _, _ string) (credentialEntity, error) {
			return credentialEntity{}, errTestConflict
		},
	}

	body := `{"username":"taken","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestUpsertCredential_DBError(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:       d,
		protocol: "TestProto",
		upsert: func(_ context.Context, _, _, _ string) (credentialEntity, error) {
			return credentialEntity{}, errors.New("db write error")
		},
	}

	body := `{"username":"myuser","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestUpsertCredential_Success(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	body := `{"username":"myuser","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Username != "myuser" {
		t.Errorf("username = %q, want %q", resp.Username, "myuser")
	}
}

func TestUpsertCredential_UsernameNormalized(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	// Username with mixed case and surrounding whitespace should be normalized.
	body := `{"username":"  MyUser  ","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Username != "myuser" {
		t.Errorf("username = %q, want lowercase trimmed %q", resp.Username, "myuser")
	}
}

func TestUpsertCredential_WithDeriveKey(t *testing.T) {
	ops, userID := makeTestCredOps(t)
	var derivedKey string
	ops.deriveKey = func(password string) string {
		derivedKey = "derived:" + password
		return derivedKey
	}

	body := `{"username":"myuser","password":"validpassword"}`
	r := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	require.Equal(t, http.StatusOK, w.Code)
	if derivedKey != "derived:validpassword" {
		t.Errorf("deriveKey was not called with plaintext password; got %q", derivedKey)
	}
}

// ---- deleteCredential ----

func TestDeleteCredential_NotFound(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestDeleteCredential_FetchError(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:       d,
		protocol: "TestProto",
		getByUserID: func(_ context.Context, _ string) (credentialEntity, error) {
			return credentialEntity{}, errors.New("fetch error")
		},
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDeleteCredential_DeleteError(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:       d,
		protocol: "TestProto",
		getByUserID: func(_ context.Context, _ string) (credentialEntity, error) {
			return credentialEntity{ID: "cred-1", Username: "myuser"}, nil
		},
		del: func(_ context.Context, _ string) error {
			return errors.New("delete error")
		},
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDeleteCredential_DeleteNotFound(t *testing.T) {
	d := newTestDB(t)
	ops := credentialOps{
		db:       d,
		protocol: "TestProto",
		getByUserID: func(_ context.Context, _ string) (credentialEntity, error) {
			return credentialEntity{ID: "cred-1", Username: "myuser"}, nil
		},
		del: func(_ context.Context, _ string) error {
			return sql.ErrNoRows
		},
	}

	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteCredential_Success(t *testing.T) {
	ops, userID := makeTestCredOps(t)

	// Create a credential first.
	putBody := `{"username":"myuser","password":"validpassword"}`
	putR := httptest.NewRequest(http.MethodPut, "/api/credentials", strings.NewReader(putBody))
	putR = withUserID(putR, userID)
	putW := httptest.NewRecorder()
	handleCredentials(ops, putW, putR)
	require.Equal(t, http.StatusOK, putW.Code)

	// Now delete it.
	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}
