package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleInit_DeviceAuthKey verifies that HandleInit response includes
// the device_auth resource key containing the token base URL.
func TestHandleInit_DeviceAuthKey(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		require.Failf(t, "failed", "status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, ok := resp["Resources"].(map[string]any)
	if !ok {
		require.Fail(t, "expected Resources object in response")
	}

	// device_auth URL should contain the token value to allow per-device routing.
	deviceAuth, _ := resources["device_auth"].(string)
	if !strings.Contains(deviceAuth, tokenValue) {
		t.Errorf("device_auth %q should contain token value %q", deviceAuth, tokenValue)
	}
}

// TestHandleInit_ImageHostKey verifies that HandleInit includes an image_host
// pointing to the server base URL.
func TestHandleInit_ImageHostKey(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "testhost.local"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, _ := resp["Resources"].(map[string]any)
	imageHost, _ := resources["image_host"].(string)
	if imageHost == "" {
		t.Error("expected non-empty image_host in Resources")
	}
	if !strings.Contains(imageHost, "testhost.local") {
		t.Errorf("image_host %q should contain request host testhost.local", imageHost)
	}
}

// TestHandleInit_LibrarySyncURL verifies that the library_sync URL includes
// the token value for per-device routing.
func TestHandleInit_LibrarySyncURL(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, _ := resp["Resources"].(map[string]any)
	librarySync, _ := resources["library_sync"].(string)
	if !strings.Contains(librarySync, tokenValue) {
		t.Errorf("library_sync %q should contain token value %q", librarySync, tokenValue)
	}
}

// TestHandleInit_ConfigurationDataKey verifies that the configuration_data
// resource key is present in the init response.
func TestHandleInit_ConfigurationDataKey(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, _ := resp["Resources"].(map[string]any)

	if resources["configuration_data"] == nil {
		t.Error("expected configuration_data in Resources")
	}
	if resources["account_page"] == nil {
		t.Error("expected account_page in Resources")
	}
}

// TestHandleInit_BlackstoneHeaderKey verifies that the blackstone_header field
// is an object in the init response Resources.
func TestHandleInit_BlackstoneHeaderKey(t *testing.T) {
	t.Parallel()

	h, userID := setupKoboHandler(t)
	handler := koboDeviceHandler(h)
	tokenValue := createTestKoboToken(t, h, userID)

	r := httptest.NewRequest(http.MethodGet, "/kobo/"+tokenValue+"/v1/initialization", nil)
	r.Host = "localhost:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, _ := resp["Resources"].(map[string]any)
	blackstone, ok := resources["blackstone_header"].(map[string]any)
	if !ok {
		require.Fail(t, "expected blackstone_header to be an object in Resources")
	}
	if blackstone["key"] == nil || blackstone["value"] == nil {
		t.Error("expected key and value in blackstone_header")
	}
}
