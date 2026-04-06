package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp), "decode")
	resources, ok := resp["Resources"].(map[string]any)
	require.True(t, ok)

	// device_auth URL should contain the token value to allow per-device routing.
	deviceAuth, _ := resources["device_auth"].(string)
	require.Contains(t, deviceAuth, tokenValue)
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
	require.NotEqual(t, "", imageHost)
	require.Contains(t, imageHost, "testhost.local")
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
	require.Contains(t, librarySync, tokenValue)
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

	require.NotNil(t, resources["configuration_data"])
	require.NotNil(t, resources["account_page"])
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
	require.True(t, ok)
	require.NotNil(t, blackstone["key"], "expected key in blackstone_header")
	require.NotNil(t, blackstone["value"], "expected value in blackstone_header")
}
