package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupKOSyncHandler(t *testing.T) (*KOSyncHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &KOSyncHandler{DB: d}

	user, err := d.CreateUser(t.Context(), "KOUser", "ko@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

// ---- Credential management (JWT-protected) ----

func TestKOSyncCredentials_MethodNotAllowed(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/kosync/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKOSyncCredentials_GetNotFound(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/kosync/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestKOSyncCredentials_Put_Success(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"username":"myreader","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "decode response")
	if resp.Username != "myreader" {
		t.Errorf("Username = %q, want %q", resp.Username, "myreader")
	}
}

func TestKOSyncCredentials_Put_LowercasesUsername(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"username":"MyReader","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "decode response")
	if resp.Username != "myreader" {
		t.Errorf("Username = %q, want %q (should be lowercase)", resp.Username, "myreader")
	}
}

func TestKOSyncCredentials_Put_EmptyUsername(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"username":"","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncCredentials_Put_ShortPassword(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"username":"reader","password":"abc"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncCredentials_Put_UsernameTooLong(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	longUsername := strings.Repeat("a", maxUsernameLen+1)
	body := mustMarshal(t, credentialRequest{Username: longUsername, Password: "secretpass"})
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestKOSyncCredentials_Get_AfterPut(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	// Create
	body := `{"username":"myreader","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Fetch
	r2 := httptest.NewRequest(http.MethodGet, "/api/kosync/credentials", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w2, r2)
	if w2.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp credentialResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp), "decode GET response")
	if resp.Username != "myreader" {
		t.Errorf("Username = %q, want %q", resp.Username, "myreader")
	}
}

func TestKOSyncCredentials_Delete(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	// Create
	body := `{"username":"myreader","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Delete
	r2 := httptest.NewRequest(http.MethodDelete, "/api/kosync/credentials", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("DELETE status = %d, want %d; body: %s", w2.Code, http.StatusNoContent, w2.Body.String())
	}

	// Confirm gone
	r3 := httptest.NewRequest(http.MethodGet, "/api/kosync/credentials", nil)
	r3 = withUserID(r3, userID)
	w3 := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w3, r3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want %d", w3.Code, http.StatusNotFound)
	}
}

func TestKOSyncCredentials_Delete_NotFound(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/kosync/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncCredentials(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestKOSyncCredentials_UsernameConflict(t *testing.T) {
	d := newTestDB(t)
	h := &KOSyncHandler{DB: d}

	user1, err := d.CreateUser(t.Context(), "User1", "u1@example.com", "pw")
	require.NoError(t, err, "create user1")
	user2, err := d.CreateUser(t.Context(), "User2", "u2@example.com", "pw")
	require.NoError(t, err, "create user2")

	// user1 claims "shared"
	body := `{"username":"shared","password":"secretpass"}`
	r := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r = withUserID(r, user1.ID)
	w := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// user2 tries to claim "shared"
	r2 := httptest.NewRequest(http.MethodPut, "/api/kosync/credentials", bytes.NewBufferString(body))
	r2 = withUserID(r2, user2.ID)
	w2 := httptest.NewRecorder()
	h.HandleKOSyncCredentials(w2, r2)
	if w2.Code != http.StatusConflict {
		t.Errorf("user2 PUT status = %d, want %d", w2.Code, http.StatusConflict)
	}
}

// ---- KOReader kosync protocol endpoints ----

func TestKOSyncUserCreate_AlwaysReturnsConflict(t *testing.T) {
	h, _ := setupKOSyncHandler(t)

	body := `{"username":"newuser","password":"somemd5hash"}`
	r := httptest.NewRequest(http.MethodPost, "/api/user/create", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.HandleKOSyncUserCreate(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (409 tells KOReader to proceed to auth)", w.Code, http.StatusConflict)
	}
}

func TestKOSyncUserCreate_MethodNotAllowed(t *testing.T) {
	h, _ := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/user/create", nil)
	w := httptest.NewRecorder()
	h.HandleKOSyncUserCreate(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKOSyncUserAuth_Success(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	// Auth endpoint simply returns 200 when the middleware has authenticated
	r := httptest.NewRequest(http.MethodGet, "/api/user/auth", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncUserAuth(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "decode response")
	if resp["authorized"] != "OK" {
		t.Errorf("authorized = %q, want %q", resp["authorized"], "OK")
	}
}

func TestKOSyncUserAuth_MethodNotAllowed(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/user/auth", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKOSyncUserAuth(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKOSyncProgress_Put_Success(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"document":"abc123","progress":"/body/p[5]","percentage":0.42,"device":"MyKindle","device_id":"dev-001"}`
	r := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp kosyncProgressResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "decode response")
	if resp.Document != "abc123" {
		t.Errorf("Document = %q, want %q", resp.Document, "abc123")
	}
	if resp.Progress != "/body/p[5]" {
		t.Errorf("Progress = %q, want %q", resp.Progress, "/body/p[5]")
	}
	if resp.Percentage != 0.42 {
		t.Errorf("Percentage = %v, want 0.42", resp.Percentage)
	}
	if resp.Device != "MyKindle" {
		t.Errorf("Device = %q, want %q", resp.Device, "MyKindle")
	}
	if resp.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
}

func TestKOSyncProgress_Put_MissingDocument(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"progress":"/body/p[1]","percentage":0.1}`
	r := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncProgress_Put_MissingProgress(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"document":"abc123","percentage":0.1}`
	r := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncProgress_Put_DocumentContainsSlash(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	body := `{"document":"path/to/doc","progress":"/body/p[1]","percentage":0.1}`
	r := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncProgress_Get_Success(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	// Insert progress first
	putBody := `{"document":"doc42","progress":"/body/p[3]","percentage":0.3}`
	rPut := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(putBody))
	rPut = withUserID(rPut, userID)
	wPut := httptest.NewRecorder()
	h.HandleKOSyncProgress(wPut, rPut)
	require.Equal(t, http.StatusOK, wPut.Code)

	// Now GET
	rGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/syncs/progress/%s", "doc42"), nil)
	rGet = withUserID(rGet, userID)
	wGet := httptest.NewRecorder()
	h.HandleKOSyncProgress(wGet, rGet)

	if wGet.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d; body: %s", wGet.Code, http.StatusOK, wGet.Body.String())
	}

	var resp kosyncProgressResponse
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &resp), "decode GET response")
	if resp.Document != "doc42" {
		t.Errorf("Document = %q, want %q", resp.Document, "doc42")
	}
	if resp.Progress != "/body/p[3]" {
		t.Errorf("Progress = %q, want %q", resp.Progress, "/body/p[3]")
	}
}

func TestKOSyncProgress_Get_NotFound(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/syncs/progress/missing-doc", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestKOSyncProgress_Get_MissingDocumentInPath(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/syncs/progress/", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestKOSyncProgress_MethodNotAllowed(t *testing.T) {
	h, userID := setupKOSyncHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/syncs/progress", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleKOSyncProgress(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestKOSyncProgress_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	h := &KOSyncHandler{DB: d}

	user1, err := d.CreateUser(t.Context(), "User1", "u1@example.com", "pw")
	require.NoError(t, err, "create user1")
	user2, err := d.CreateUser(t.Context(), "User2", "u2@example.com", "pw")
	require.NoError(t, err, "create user2")

	// user1 writes progress
	putBody := `{"document":"shared-doc","progress":"/body/p[1]","percentage":0.1}`
	rPut := httptest.NewRequest(http.MethodPut, "/api/syncs/progress", bytes.NewBufferString(putBody))
	rPut = withUserID(rPut, user1.ID)
	wPut := httptest.NewRecorder()
	h.HandleKOSyncProgress(wPut, rPut)
	require.Equal(t, http.StatusOK, wPut.Code)

	// user2 cannot see user1's progress
	rGet := httptest.NewRequest(http.MethodGet, "/api/syncs/progress/shared-doc", nil)
	rGet = withUserID(rGet, user2.ID)
	wGet := httptest.NewRecorder()
	h.HandleKOSyncProgress(wGet, rGet)
	if wGet.Code != http.StatusNotFound {
		t.Errorf("user2 GET status = %d, want %d (progress must be isolated)", wGet.Code, http.StatusNotFound)
	}
}
