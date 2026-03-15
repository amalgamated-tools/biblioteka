package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func setupAPIKeyHandler(t *testing.T) (*APIKeyHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}
	user, err := d.CreateUser(context.Background(), "Test User", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestCreateAPIKey_Success(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body, _ := json.Marshal(apiKeyCreateRequest{Name: "CI Pipeline"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp apiKeyCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Key == "" {
		t.Error("expected non-empty key in response")
	}
	if !strings.HasPrefix(resp.Key, auth.APIKeyPrefix) {
		t.Errorf("key %q should start with %q", resp.Key, auth.APIKeyPrefix)
	}
	if resp.Name != "CI Pipeline" {
		t.Errorf("name = %q, want %q", resp.Name, "CI Pipeline")
	}
	if resp.KeyPrefix == "" {
		t.Error("expected non-empty key_prefix")
	}
	if resp.ID == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreateAPIKey_EmptyName(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body, _ := json.Marshal(apiKeyCreateRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateAPIKey_NameTooLong(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	longName := strings.Repeat("a", maxAPIKeyNameLength+1)
	body, _ := json.Marshal(apiKeyCreateRequest{Name: longName})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestListAPIKeys_Empty(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []apiKeyDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dtos) != 0 {
		t.Errorf("len = %d, want 0", len(dtos))
	}
}

func TestListAPIKeys_AfterCreate(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create two keys.
	for _, name := range []string{"Key A", "Key B"} {
		body, err := json.Marshal(apiKeyCreateRequest{Name: name})
		if err != nil {
			t.Fatalf("marshal create request for %q: %v", name, err)
		}
		r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
		r = withUserID(r, userID)
		w := httptest.NewRecorder()
		h.HandleAPIKeys(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %q: status = %d", name, w.Code)
		}
	}

	// List them.
	r := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var dtos []apiKeyDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
	// List should not expose the full key.
	raw := w.Body.String()
	if strings.Contains(raw, auth.APIKeyPrefix) && strings.Count(raw, auth.APIKeyPrefix) != strings.Count(raw, "key_prefix") {
		t.Error("list response should not contain full API key values")
	}
}

func TestListAPIKeys_UserScoped(t *testing.T) {
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}

	user1, err := d.CreateUser(context.Background(), "User 1", "u1@example.com", "password1")
	if err != nil {
		t.Fatalf("create user1: %v", err)
	}
	user2, err := d.CreateUser(context.Background(), "User 2", "u2@example.com", "password2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	// Create a key for user1.
	body, err := json.Marshal(apiKeyCreateRequest{Name: "User1 Key"})
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, user1.ID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	// User2 should see no keys.
	r = httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, user2.ID)
	w = httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var dtos []apiKeyDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dtos); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(dtos) != 0 {
		t.Errorf("user2 should see 0 keys, got %d", len(dtos))
	}
}

func TestDeleteAPIKey_Success(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create a key.
	body, _ := json.Marshal(apiKeyCreateRequest{Name: "Ephemeral"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v; body: %s", err, w.Body.String())
	}

	// Delete it.
	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify it's gone.
	r = httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var remaining []apiKeyDTO
	if err := json.Unmarshal(w.Body.Bytes(), &remaining); err != nil {
		t.Fatalf("unmarshal list response: %v; body: %s", err, w.Body.String())
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 keys after delete, got %d", len(remaining))
	}
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/api-keys/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteAPIKey_OtherUserCannotDelete(t *testing.T) {
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}

	user1, _ := d.CreateUser(context.Background(), "User 1", "u1@example.com", "password1")
	user2, _ := d.CreateUser(context.Background(), "User 2", "u2@example.com", "password2")

	// Create a key as user1.
	body, _ := json.Marshal(apiKeyCreateRequest{Name: "User1 Key"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, user1.ID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Try to delete as user2.
	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, user2.ID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (other user should not see the key)", w.Code, http.StatusNotFound)
	}
}

func TestCreateAPIKey_AuditLog(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body, _ := json.Marshal(apiKeyCreateRequest{Name: "Audited Key"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	logs, _, err := h.DB.ListAuditLogs(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyCreated {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log entry with action api_key.created")
	}
}

func TestDeleteAPIKey_AuditLog(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create then delete.
	body, _ := json.Marshal(apiKeyCreateRequest{Name: "ToDelete"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	logs, _, err := h.DB.ListAuditLogs(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyDeleted {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit log entry with action api_key.deleted")
	}
}

func TestHandleAPIKeys_MethodNotAllowed(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
