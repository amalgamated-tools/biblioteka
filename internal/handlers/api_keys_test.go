package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func setupAPIKeyHandler(t *testing.T) (*APIKeyHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestCreateAPIKey_Success(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body := mustMarshal(t, apiKeyCreateRequest{Name: "CI Pipeline"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.NotEqual(t, "", resp.Key)
	require.True(t, strings.HasPrefix(resp.Key, auth.APIKeyPrefix))
	require.Equal(t, "CI Pipeline", resp.Name)
	require.NotEqual(t, "", resp.KeyPrefix)
	require.NotEqual(t, "", resp.ID)
}

func TestCreateAPIKey_EmptyName(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body := mustMarshal(t, apiKeyCreateRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAPIKey_NameTooLong(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	longName := strings.Repeat("a", maxTokenNameLength+1)
	body := mustMarshal(t, apiKeyCreateRequest{Name: longName})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListAPIKeys_Empty(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []apiKeyDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 0)
}

func TestListAPIKeys_AfterCreate(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create two keys.
	for _, name := range []string{"Key A", "Key B"} {
		body, err := json.Marshal(apiKeyCreateRequest{Name: name})
		require.NoError(t, err, "marshal create request for %q", name)
		r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
		r = withUserID(r, userID)
		w := httptest.NewRecorder()
		h.HandleAPIKeys(w, r)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List them.
	r := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []apiKeyDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal")
	require.Len(t, dtos, 2)
	// List should not expose the full key — bib_ should only appear in key_prefix fields.
	raw := w.Body.String()
	require.Equal(t, strings.Count(raw, auth.APIKeyPrefix), strings.Count(raw, "key_prefix"), "list response should not contain full API key values")
}

func TestListAPIKeys_UserScoped(t *testing.T) {
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}

	user1, err := d.CreateUser(t.Context(), "User 1", "u1@example.com", "password1")
	require.NoError(t, err, "create user1")
	user2, err := d.CreateUser(t.Context(), "User 2", "u2@example.com", "password2")
	require.NoError(t, err, "create user2")

	// Create a key for user1.
	body, err := json.Marshal(apiKeyCreateRequest{Name: "User1 Key"})
	require.NoError(t, err, "marshal create request")
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos), "unmarshal list response")
	require.Len(t, dtos, 0)
}

func TestDeleteAPIKey_Success(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create a key.
	body := mustMarshal(t, apiKeyCreateRequest{Name: "Ephemeral"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal create response; body: %s", w.Body.String())

	// Delete it.
	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's gone.
	r = httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var remaining []apiKeyDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &remaining), "unmarshal list response; body: %s", w.Body.String())
	require.Len(t, remaining, 0)
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/api-keys/nonexistent-id", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAPIKey_OtherUserCannotDelete(t *testing.T) {
	d := newTestDB(t)
	h := &APIKeyHandler{DB: d}

	user1, err := d.CreateUser(t.Context(), "User 1", "u1@example.com", "password1")
	require.NoError(t, err, "create user1")
	user2, err := d.CreateUser(t.Context(), "User 2", "u2@example.com", "password2")
	require.NoError(t, err, "create user2")

	// Create a key as user1.
	body := mustMarshal(t, apiKeyCreateRequest{Name: "User1 Key"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, user1.ID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "failed to unmarshal response body")

	// Try to delete as user2.
	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, user2.ID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateAPIKey_AuditLog(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	body := mustMarshal(t, apiKeyCreateRequest{Name: "Audited Key"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyCreated {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestDeleteAPIKey_AuditLog(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	// Create then delete.
	body := mustMarshal(t, apiKeyCreateRequest{Name: "ToDelete"})
	r := httptest.NewRequest(http.MethodPost, "/api/api-keys", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	var created apiKeyCreateResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal create response; body: %s", w.Body.String())

	r = httptest.NewRequest(http.MethodDelete, "/api/api-keys/"+created.ID, nil)
	r = withUserID(r, userID)
	w = httptest.NewRecorder()
	h.HandleAPIKey(w, r)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionAPIKeyDeleted {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestHandleAPIKeys_MethodNotAllowed(t *testing.T) {
	h, userID := setupAPIKeyHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/api-keys", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleAPIKeys(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
