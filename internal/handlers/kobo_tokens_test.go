package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestKoboTokenList_WithTokens verifies that the list endpoint returns all
// tokens belonging to the authenticated user.
func TestKoboTokenList_WithTokens(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)

	// Create two tokens.
	for _, name := range []string{"Device A", "Device B"} {
		body := mustMarshal(t, koboTokenCreateRequest{Name: name})
		r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
		r = withUserID(r, userID)
		w := httptest.NewRecorder()
		h.HandleKoboTokens(w, r)
		if w.Code != http.StatusCreated {
			require.Failf(t, "failed", "create token %q: status = %d, want %d", name, w.Code, http.StatusCreated)
		}
	}

	// List tokens.
	r := httptest.NewRequest(http.MethodGet, "/api/kobo/tokens", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "list tokens: status = %d, want %d", w.Code, http.StatusOK)
	}

	var tokens []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tokens), "decode list response")
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
}

// TestKoboTokenCreate_ResponseContainsToken verifies that the creation response
// includes the raw token string which cannot be retrieved again.
func TestKoboTokenCreate_ResponseContainsToken(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: "New Device"})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusCreated {
		require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusCreated)
	}
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode create response")
	token, _ := resp["token"].(string)
	if len(token) == 0 {
		t.Error("expected non-empty token in creation response")
	}
}

// TestKoboTokenCreate_NameTooLong verifies that names longer than the max token
// name length are rejected.
func TestKoboTokenCreate_NameTooLong(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)

	longName := make([]byte, 200)
	for i := range longName {
		longName[i] = 'A'
	}
	body := mustMarshal(t, koboTokenCreateRequest{Name: string(longName)})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for name too long", w.Code, http.StatusBadRequest)
	}
}

// TestKoboTokenCreate_NameTrimmed verifies that leading/trailing whitespace
// in the token name is trimmed.
func TestKoboTokenCreate_NameTrimmed(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)

	body := mustMarshal(t, koboTokenCreateRequest{Name: "  My Device  "})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleKoboTokens(w, r)

	if w.Code != http.StatusCreated {
		require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusCreated)
	}
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode create response")
	name, _ := resp["name"].(string)
	if name != "My Device" {
		t.Errorf("name = %q, want %q (should be trimmed)", name, "My Device")
	}
}

// TestKoboTokenDelete_UserIsolation verifies that a user cannot delete another
// user's Kobo token.
func TestKoboTokenDelete_UserIsolation(t *testing.T) {
	t.Parallel()

	h, userID1 := setupKoboHandler(t)
	user2, err := h.DB.CreateUser(context.Background(), "User2", "user2@example.com", "password")
	require.NoError(t, err, "create user2")

	// Create a token for user1.
	createBody := mustMarshal(t, koboTokenCreateRequest{Name: "User1 Device"})
	r := httptest.NewRequest(http.MethodPost, "/api/kobo/tokens", bytes.NewReader(createBody))
	r = withUserID(r, userID1)
	w := httptest.NewRecorder()
	h.HandleKoboTokens(w, r)
	if w.Code != http.StatusCreated {
		require.Failf(t, "failed", "create token: status = %d, want %d", w.Code, http.StatusCreated)
	}
	var created map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created), "decode create response")
	tokenID, _ := created["id"].(string)

	// Attempt to delete user1's token as user2.
	r2 := httptest.NewRequest(http.MethodDelete, "/api/kobo/tokens/"+tokenID, nil)
	r2 = withUserID(r2, user2.ID)
	w2 := httptest.NewRecorder()
	h.HandleKoboToken(w2, r2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (user isolation)", w2.Code, http.StatusNotFound)
	}
}
