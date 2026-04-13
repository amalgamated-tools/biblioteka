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

// noopIssuerURLValidator disables SSRF URL validation for tests that focus on
// other aspects of HandleSetOIDCConfig (e.g., settings persistence). Tests for
// SSRF validation itself call validateOIDCIssuerURL directly.
func noopIssuerURLValidator(_ context.Context, _ string) error { return nil }

// --- validateOIDCIssuerURL ---

func TestValidateOIDCIssuerURL_HttpSchemeRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "http://auth.example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "https")
}

func TestValidateOIDCIssuerURL_FileSchemeRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "file:///etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "https")
}

func TestValidateOIDCIssuerURL_GopherSchemeRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "gopher://internal.host/")
	require.Error(t, err)
	require.Contains(t, err.Error(), "https")
}

func TestValidateOIDCIssuerURL_LoopbackIPv4Rejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://127.0.0.1")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_AWSMetadataIPRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://169.254.169.254")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_RFC1918ClassARejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://10.0.0.1")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_RFC1918ClassBRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://172.16.0.1")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_RFC1918ClassCRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://192.168.1.1")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_IPv6LoopbackRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://[::1]")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_IPv6LinkLocalRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://[fe80::1]")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_LocalhostRejectedViaDNS(t *testing.T) {
	// "localhost" is a DNS name (not a literal IP), but it resolves to the
	// loopback address; the DNS-resolution check must catch it.
	err := validateOIDCIssuerURL(t.Context(), "https://localhost")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_ValidPublicURL(t *testing.T) {
	// A syntactically valid https URL with a non-private host should pass.
	// validateOIDCIssuerURL does not call oidc.NewProvider here, so this avoids
	// the OIDC HTTP discovery request, but hostname validation may still perform
	// DNS resolution.
	err := validateOIDCIssuerURL(t.Context(), "https://auth.example.com")
	require.NoError(t, err)
}

func TestValidateOIDCIssuerURL_MissingHost(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://")
	require.Error(t, err)
}

func TestValidateOIDCIssuerURL_UserinfoRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://user:pass@issuer.example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "userinfo")
}

func TestValidateOIDCIssuerURL_UserinfoUsernameOnlyRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://admin@issuer.example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "userinfo")
}

func TestValidateOIDCIssuerURL_IPv6ZoneIDRejected(t *testing.T) {
	err := validateOIDCIssuerURL(t.Context(), "https://[fe80::1%25lo0]")
	require.Error(t, err)
	require.Contains(t, err.Error(), "zone")
}

// --- HandleGetOIDCConfig ---

func TestHandleGetOIDCConfig_AdminNoSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp oidcConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "", resp.IssuerURL)
	require.False(t, resp.ClientSecretSet)
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

	require.Equal(t, http.StatusOK, w.Code)
	var resp oidcConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "https://auth.example.com", resp.IssuerURL)
	require.Equal(t, "my-client-id", resp.ClientID)
	require.True(t, resp.ClientSecretSet)
	require.Equal(t, "https://app.example.com/callback", resp.RedirectURI)
}

func TestHandleGetOIDCConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

// --- HandleSetOIDCConfig ---

func TestHandleSetOIDCConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSetOIDCConfig_MissingIssuerURL(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_MissingClientID(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_MissingRedirectURI(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"secret","redirect_uri":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_MissingSecretNoExisting(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// No existing secret, and empty secret in request → should fail
	body := `{"issuer_url":"https://auth.example.com","client_id":"id","client_secret":"","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_InvalidJSON(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// --- SSRF rejection tests (handler level) ---

func TestHandleSetOIDCConfig_SSRFHttpSchemeRejected(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"http://169.254.169.254","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_SSRFPrivateIPRejected(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://192.168.1.1","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_SSRFLoopbackIPRejected(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://127.0.0.1/oidc","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_SSRFAWSMetadataIPRejected(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"issuer_url":"https://169.254.169.254","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_ProviderDiscoveryFailure(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	// Bypass URL validation so we reach the oidc.NewProvider call.
	h.IssuerURLValidator = noopIssuerURLValidator
	h.OIDCHTTPClient = http.DefaultClient

	// Use an unreachable/invalid issuer URL; oidc.NewProvider will fail
	body := `{"issuer_url":"https://invalid.example.invalid","client_id":"id","client_secret":"secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetOIDCConfig_ValidProviderSavesSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	// Bypass URL validation and SSRF-safe client: this test exercises settings
	// persistence and the OnOIDCConfigSet callback, not SSRF protection.
	h.IssuerURLValidator = noopIssuerURLValidator
	h.OIDCHTTPClient = http.DefaultClient

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

	require.Equal(t, http.StatusOK, w.Code)

	// Verify settings were persisted
	issuerURL, err := h.DB.GetSetting(t.Context(), settingOIDCIssuerURL)
	require.NoError(t, err, "GetSetting(issuer_url) error")
	require.Equal(t, oidcServer.URL, issuerURL)

	clientID, err := h.DB.GetSetting(t.Context(), settingOIDCClientID)
	require.NoError(t, err, "GetSetting(client_id) error")
	require.Equal(t, "my-client", clientID)

	require.True(t, callbackCalled)
}

func TestHandleSetOIDCConfig_PreservesExistingSecret(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	// Bypass URL validation and SSRF-safe client: this test exercises
	// secret-preservation logic, not SSRF protection.
	h.IssuerURLValidator = noopIssuerURLValidator
	h.OIDCHTTPClient = http.DefaultClient

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

	require.Equal(t, http.StatusOK, w.Code)

	// The stored secret should still be the original one
	secret, err := h.DB.GetSetting(t.Context(), settingOIDCClientSecret)
	require.NoError(t, err, "GetSetting(client_secret)")
	require.Equal(t, "existing-secret", secret)
}

// --- HandleOIDCConfig (dispatch) ---

func TestHandleOIDCConfig_DispatchGet(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleOIDCConfig(w, r)

	// GET dispatches to HandleGetOIDCConfig; admin user → 200
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleOIDCConfig_DispatchMethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleOIDCConfig(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- Encryption tests ---

func TestHandleSetOIDCConfig_EncryptsClientSecret(t *testing.T) {
	h, adminID, _ := setupConfigHandlerWithSecrets(t)
	h.IssuerURLValidator = noopIssuerURLValidator
	h.OIDCHTTPClient = http.DefaultClient

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

	body := `{"issuer_url":"` + oidcServer.URL + `","client_id":"my-client","client_secret":"my-secret","redirect_uri":"https://app.example.com/callback"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetOIDCConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	stored, err := h.DB.GetSetting(t.Context(), settingOIDCClientSecret)
	require.NoError(t, err)
	require.NotEqual(t, "my-secret", stored, "secret must not be stored as plaintext")
	require.Contains(t, stored, "enc:v1:", "stored secret must have encryption prefix")

	// GET must still report that the secret is set.
	r2 := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleGetOIDCConfig(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)
	var resp oidcConfigResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	require.True(t, resp.ClientSecretSet)
}

func TestHandleSetOIDCConfig_PreservesExistingEncryptedSecret(t *testing.T) {
	h, adminID, _ := setupConfigHandlerWithSecrets(t)
	h.IssuerURLValidator = noopIssuerURLValidator
	h.OIDCHTTPClient = http.DefaultClient

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

	// First PUT: store an encrypted secret.
	body := `{"issuer_url":"` + oidcServer.URL + `","client_id":"my-client","client_secret":"original-secret","redirect_uri":"https://app/cb"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleSetOIDCConfig(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Capture what the callback receives on second PUT (empty secret).
	var callbackSecret string
	h.OnOIDCConfigSet = func(_ context.Context, _, _, secret, _ string) error {
		callbackSecret = secret
		return nil
	}

	// Second PUT with empty secret — should reuse the stored (encrypted) one.
	body2 := `{"issuer_url":"` + oidcServer.URL + `","client_id":"my-client","client_secret":"","redirect_uri":"https://app/cb"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/config/oidc", bytes.NewBufferString(body2))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleSetOIDCConfig(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)

	// The callback must receive the plaintext secret (not the encrypted form).
	require.Equal(t, "original-secret", callbackSecret)
}

