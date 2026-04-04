package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["AccessToken"] == nil || resp["AccessToken"] == "" {
		t.Error("expected non-empty AccessToken in refresh response")
	}
	if resp["RefreshToken"] == nil || resp["RefreshToken"] == "" {
		t.Error("expected non-empty RefreshToken in refresh response")
	}
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
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

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// When no body is provided, UserKey should be the zero value.
	if _, ok := resp["UserKey"]; !ok {
		t.Error("expected UserKey field in response even when body is nil")
	}
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
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["TokenType"] != "Bearer" {
		t.Errorf("TokenType = %v, want Bearer", resp["TokenType"])
	}
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
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	trackingID, _ := resp["TrackingId"].(string)
	if len(trackingID) == 0 {
		t.Error("expected non-empty TrackingId")
	}
	// UUID format has 4 hyphens: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	hyphenCount := 0
	for _, c := range trackingID {
		if c == '-' {
			hyphenCount++
		}
	}
	if hyphenCount != 4 {
		t.Errorf("TrackingId %q does not look like a UUID (expected 4 hyphens, got %d)", trackingID, hyphenCount)
	}
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
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestKoboRandomUUID_IsUnique verifies that successive calls to koboRandomUUID
// produce different values.
func TestKoboRandomUUID_IsUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 10 {
		id, err := koboRandomUUID()
		if err != nil {
			t.Fatalf("koboRandomUUID() error: %v", err)
		}
		if seen[id] {
			t.Errorf("koboRandomUUID() returned duplicate value %q", id)
		}
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
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_ = user

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/device", bytes.NewBufferString(`{"UserKey":"direct-key"}`))
	w := httptest.NewRecorder()
	h.HandleAuth(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["UserKey"] != "direct-key" {
		t.Errorf("UserKey = %v, want direct-key", resp["UserKey"])
	}
}
