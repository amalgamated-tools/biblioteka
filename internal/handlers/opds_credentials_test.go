package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupOPDSCredentialHandler(t *testing.T) (*OPDSCredentialHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &OPDSCredentialHandler{DB: d}

	user, err := d.CreateUser(context.Background(), "TestUser", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestOPDSCredentials_MethodNotAllowed(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOPDSCredentials_GetNotFound(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOPDSCredentials_PutSuccess(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	body, _ := json.Marshal(opdsCredentialRequest{Username: "myreader", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp opdsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "myreader" {
		t.Errorf("username = %q, want %q", resp.Username, "myreader")
	}
}

func TestOPDSCredentials_GetAfterPut(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	// Create credentials first.
	body, _ := json.Marshal(opdsCredentialRequest{Username: "myreader", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Now GET them.
	r = httptest.NewRequest(http.MethodGet, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp opdsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "myreader" {
		t.Errorf("username = %q, want %q", resp.Username, "myreader")
	}
}

func TestOPDSCredentials_PutUpdateExisting(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	// Create initial credentials.
	body, _ := json.Marshal(opdsCredentialRequest{Username: "oldname", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("first PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	// Update with new username.
	body, _ = json.Marshal(opdsCredentialRequest{Username: "newname", Password: "secret456"})
	r = httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("second PUT status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp opdsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "newname" {
		t.Errorf("username = %q, want %q", resp.Username, "newname")
	}
}

func TestOPDSCredentials_PutEmptyUsername(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	body, _ := json.Marshal(opdsCredentialRequest{Username: "", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOPDSCredentials_PutShortPassword(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	body, _ := json.Marshal(opdsCredentialRequest{Username: "myreader", Password: "abc"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOPDSCredentials_PutInvalidJSON(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader([]byte("not json")))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOPDSCredentials_PutUsernameLowercased(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	body, _ := json.Marshal(opdsCredentialRequest{Username: "MyReader", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp opdsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "myreader" {
		t.Errorf("username = %q, want %q (should be lowercased)", resp.Username, "myreader")
	}
}

func TestOPDSCredentials_PutDuplicateUsername(t *testing.T) {
	d := newTestDB(t)
	h := &OPDSCredentialHandler{DB: d}

	ctx := context.Background()
	user1, err := d.CreateUser(ctx, "User1", "user1@example.com", "password1")
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	user2, err := d.CreateUser(ctx, "User2", "user2@example.com", "password1")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	// User1 creates credentials with username "reader".
	body, _ := json.Marshal(opdsCredentialRequest{Username: "reader", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, user1.ID)
	w := httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("user1 PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	// User2 tries to use the same username.
	body, _ = json.Marshal(opdsCredentialRequest{Username: "reader", Password: "secret456"})
	r = httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, user2.ID)
	w = httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("user2 PUT status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOPDSCredentials_DeleteSuccess(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	// Create credentials first.
	body, _ := json.Marshal(opdsCredentialRequest{Username: "myreader", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d; body: %s", w.Code, w.Body.String())
	}

	// Delete them.
	r = httptest.NewRequest(http.MethodDelete, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("DELETE status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify they're gone.
	r = httptest.NewRequest(http.MethodGet, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOPDSCredentials_DeleteNotFound(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/opds/credentials", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOPDSCredentials_PutUsernameTrimmed(t *testing.T) {
	h, userID := setupOPDSCredentialHandler(t)

	body, _ := json.Marshal(opdsCredentialRequest{Username: "  spacey  ", Password: "secret123"})
	r := httptest.NewRequest(http.MethodPut, "/api/opds/credentials", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleOPDSCredentials(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp opdsCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Username != "spacey" {
		t.Errorf("username = %q, want %q (should be trimmed and lowercased)", resp.Username, "spacey")
	}
}
