package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// makeTestCredOps builds a minimal credentialOps backed by a real DB and
// simple in-memory state via closures. It is sufficient to exercise the
// handleCredentials/getCredential/upsertCredential/deleteCredential paths
// without committing to any particular protocol schema.
func makeTestCredOps(t *testing.T) (credentialOps, string) {
	t.Helper()
	d := newTestDB(t)
	user, err := d.CreateUser(t.Context(), "CredUser", "cred@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	ops := credentialOps{
		db:              d,
		protocol:        "TestProto",
		auditEntityType: "testproto_credential",
		auditUpsert:     db.AuditActionKOSyncCredentialUpdated,
		auditDelete:     db.AuditActionKOSyncCredentialDeleted,
		errConflict:     db.ErrKOSyncUsernameExists,
		getByUserID: func(ctx context.Context, userID string) (credentialEntity, error) {
			c, err := d.GetKOSyncCredentialByUserID(ctx, userID)
			if err != nil {
				return credentialEntity{}, err
			}
			return credentialEntity{
				ID:        c.ID,
				Username:  c.Username,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
			}, nil
		},
		upsert: func(ctx context.Context, userID, username, hash string) (credentialEntity, error) {
			c, err := d.UpsertKOSyncCredential(ctx, userID, username, hash)
			if err != nil {
				return credentialEntity{}, err
			}
			return credentialEntity{
				ID:        c.ID,
				Username:  c.Username,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
			}, nil
		},
		del: d.DeleteKOSyncCredential,
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
	if putW.Code != http.StatusOK {
		t.Fatalf("PUT setup failed: status=%d body=%s", putW.Code, putW.Body.String())
	}

	// Now GET it.
	r := httptest.NewRequest(http.MethodGet, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp credentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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
	errConflict := errors.New("username conflict")
	ops := credentialOps{
		db:          d,
		protocol:    "TestProto",
		errConflict: errConflict,
		upsert: func(_ context.Context, _, _, _ string) (credentialEntity, error) {
			return credentialEntity{}, errConflict
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp credentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp credentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
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
	if putW.Code != http.StatusOK {
		t.Fatalf("PUT setup failed: status=%d body=%s", putW.Code, putW.Body.String())
	}

	// Now delete it.
	r := httptest.NewRequest(http.MethodDelete, "/api/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	handleCredentials(ops, w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}
