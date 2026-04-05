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

// TestHandleAuth_RefreshEndpoint verifies that HandleAuth handles the
// /v1/auth/refresh path and returns the expected response shape.
func TestHandleAuth_RefreshEndpoint(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/refresh", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode response")
	require.NotEmpty(t, resp["AccessToken"], "expected non-empty AccessToken in refresh response")
	require.NotEmpty(t, resp["RefreshToken"], "expected non-empty RefreshToken in refresh response")
}

// TestHandleAuth_ExchangeEndpoint verifies that HandleAuth handles the
// /v1/auth/exchange path.
func TestHandleAuth_ExchangeEndpoint(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/exchange", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

// TestHandleAuth_NilBody verifies that HandleAuth works correctly when no
// request body is provided (UserKey should be empty string in response).
func TestHandleAuth_NilBody(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/auth/device", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode response")
	// When no body is provided, UserKey should be the zero value.
	_, ok := resp["UserKey"]
	require.True(t, ok, "expected UserKey field in response even when body is nil")
}

// TestHandleAuth_TokenType verifies that the response always contains the
// expected TokenType value.
func TestHandleAuth_TokenType(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/device",
		bytes.NewBufferString(`{}`))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode response")
	require.Equal(t, "Bearer", resp["TokenType"])
}

// TestHandleAuth_TrackingIDFormat verifies that TrackingId is a UUID-like string.
func TestHandleAuth_TrackingIDFormat(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/device", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode response")
	trackingID, _ := resp["TrackingId"].(string)
	require.NotEmpty(t, trackingID)
	// UUID format has 4 hyphens: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	hyphenCount := 0
	for _, c := range trackingID {
		if c == '-' {
			hyphenCount++
		}
	}
	require.Equal(t, 4, hyphenCount)
}

// TestHandleAuth_InvalidJSONBody verifies that HandleAuth tolerates an invalid
// JSON body (UserKey defaults to empty).
func TestHandleAuth_InvalidJSONBody(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodPost, "/kobo/"+tokenValue+"/v1/auth/device",
		bytes.NewBufferString(`not valid json`))
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// HandleAuth tolerates decode errors; should still return 200.
	require.Equal(t, http.StatusOK, w.Code)
}

// TestKoboRandomUUID_IsUnique verifies that successive calls to koboRandomUUID
// produce different values.
func TestKoboRandomUUID_IsUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 10 {
		id, err := koboRandomUUID()
		require.NoError(t, err, "koboRandomUUID() error")
		require.False(t, seen[id], "koboRandomUUID() returned duplicate value %q", id)
		seen[id] = true
	}
}

// TestHandleAuth_DirectHandler verifies HandleAuth directly without the full
// Kobo device middleware.
func TestHandleAuth_DirectHandler(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	user, err := d.CreateUser(context.Background(), "Test User", "test@example.com", "password")
	require.NoError(t, err, "create user")
	_ = user

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/device", bytes.NewBufferString(`{"UserKey":"direct-key"}`))
	w := httptest.NewRecorder()
	h.HandleAuth(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	require.Equal(t, "direct-key", resp["UserKey"])
}
