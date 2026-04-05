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

// --- HandleGetOIDCConfig ---

func TestHandleGetOIDCConfig_AdminNoSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp oidcConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.IssuerURL != "" {
		t.Errorf("IssuerURL = %q, want empty", resp.IssuerURL)
	}
	if resp.ClientSecretSet {
		t.Error("ClientSecretSet should be false when no secret is stored")
	}
}

func TestHandleGetOIDCConfig_AdminWithSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Pre-populate settings
	_ = h.DB.SetSetting(t.Context(), settingOIDCIssuerURL, "https://auth.example.com")
	_ = h.DB.SetSetting(t.Context(), settingOIDCClientID, "my-client-id")
	_ = h.DB.SetSetting(t.Context(), settingOIDCClientSecret, "my-secret")
	_ = h.DB.SetSetting(t.Context(), settingOIDCRedirectURI, "https://app.example.com/callback")

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp oidcConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.IssuerURL != "https://auth.example.com" {
		t.Errorf("IssuerURL = %q, want %q", resp.IssuerURL, "https://auth.example.com")
	}
	if resp.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, want %q", resp.ClientID, "my-client-id")
	}
	if !resp.ClientSecretSet {
		t.Error("ClientSecretSet should be true when secret is stored")
	}
	if resp.RedirectURI != "https://app.example.com/callback" {
		t.Errorf("RedirectURI = %q, want %q", resp.RedirectURI, "https://app.example.com/callback")
	}
}

func TestHandleGetOIDCConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

// --- HandleSetOIDCConfig ---

func TestHandleSetOIDCConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_MissingIssuerURL(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_MissingClientID(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_MissingRedirectURI(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"secret","redirect_uri":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_MissingSecretNoExisting(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// No existing secret, and empty secret in request → should fail
	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_InvalidJSON(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_ProviderDiscoveryFailure(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Use an unreachable/invalid issuer URL; oidc.NewProvider will fail
	body := `{"issuer_url":"https://invalid.example.invalid","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetOIDCConfig_ValidProviderSavesSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Serve a minimal OIDC discovery document from a test HTTP server.
	oidcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"issuer": "` + "http://" + r.Host + `",
				"authorization_endpoint": "http://` + r.Host + `/authorize",
				"token_endpoint": "http://` + r.Host + `/token",
				"jwks_uri": "http://` + r.Host + `/jwks"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer oidcServer.Close()

	var callbackCalled bool
	h.OnOIDCConfigSet = func(ctx context.Context, issuerURL, clientID, clientSecret, redirectURI string) error {
		callbackCalled = true
		return nil
	}

	body := `{"issuer_url":"` + oidcServer.URL + `","client_id":"my-client","client_secret":"my-secret","redirect_uri":"https://app.example.com/callback"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify settings were persisted
	issuerURL, err := h.DB.GetSetting(t.Context(), settingOIDCIssuerURL)
	require.NoError(t, err, "GetSetting(issuer_url) error")
	if issuerURL != oidcServer.URL {
		t.Errorf("saved issuer_url = %q, want %q", issuerURL, oidcServer.URL)
	}

	clientID, err := h.DB.GetSetting(t.Context(), settingOIDCClientID)
	require.NoError(t, err, "GetSetting(client_id) error")
	if clientID != "my-client" {
		t.Errorf("saved client_id = %q, want %q", clientID, "my-client")
	}

	if !callbackCalled {
		t.Error("expected OnOIDCConfigSet callback to be called")
	}
}

func TestHandleSetOIDCConfig_PreservesExistingSecret(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Pre-store a secret
	_ = h.DB.SetSetting(t.Context(), settingOIDCClientSecret, "existing-secret")

	// Serve a minimal OIDC discovery document.
	oidcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"issuer": "` + "http://" + r.Host + `",
				"authorization_endpoint": "http://` + r.Host + `/authorize",
				"token_endpoint": "http://` + r.Host + `/token",
				"jwks_uri": "http://` + r.Host + `/jwks"
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer oidcServer.Close()

	// Send request with empty client_secret — should reuse the existing one
	body := `{"issuer_url":"` + oidcServer.URL + `","client_id":"my-client","client_secret":"","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// The stored secret should still be the original one
	secret, err := h.DB.GetSetting(t.Context(), settingOIDCClientSecret)
	require.NoError(t, err, "GetSetting(client_secret)")
	if secret != "existing-secret" {
		t.Errorf("client_secret = %q, want %q", secret, "existing-secret")
	}
}

// --- HandleOIDCConfig (dispatch) ---

func TestHandleOIDCConfig_DispatchGet(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleOIDCConfig(w, r)

	// GET dispatches to HandleGetOIDCConfig; admin user → 200
	if w.Code != http.StatusOK {
		t.Errorf("GET dispatch: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleOIDCConfig_DispatchMethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleOIDCConfig(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
