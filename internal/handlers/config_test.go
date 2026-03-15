package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"testing"
)

// setupConfigHandler creates a ConfigHandler with a test DB, an admin user, and
// a regular user. The IsOIDCConfigured callback defaults to always returning
// false; tests can override it.
func setupConfigHandler(t *testing.T) (*ConfigHandler, string, string) {
	t.Helper()
	d := newTestDB(t)

	admin, err := d.CreateUser(context.Background(), "Admin", "admin@example.com", "password1")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	regular, err := d.CreateUser(context.Background(), "Regular", "regular@example.com", "password1")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}

	h := &ConfigHandler{
		DB:               d,
		IsOIDCConfigured: func() bool { return false },
	}
	return h, admin.ID, regular.ID
}

// --- HandleConfigStatus ---

func TestHandleConfigStatus_Success(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp configStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.OIDCConfigured {
		t.Error("expected OIDCConfigured=false")
	}
	if !resp.IsAdmin {
		t.Error("expected IsAdmin=true for admin user")
	}
}

func TestHandleConfigStatus_RegularUser(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp configStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.IsAdmin {
		t.Error("expected IsAdmin=false for regular user")
	}
}

func TestHandleConfigStatus_WhenConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	h.IsOIDCConfigured = func() bool { return true }

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	var resp configStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.OIDCConfigured {
		t.Error("expected OIDCConfigured=true")
	}
}

func TestHandleConfigStatus_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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
	_ = h.DB.SetSetting(context.Background(), settingOIDCIssuerURL, "https://auth.example.com")
	_ = h.DB.SetSetting(context.Background(), settingOIDCClientID, "my-client-id")
	_ = h.DB.SetSetting(context.Background(), settingOIDCClientSecret, "my-secret")
	_ = h.DB.SetSetting(context.Background(), settingOIDCRedirectURI, "https://app.example.com/callback")

	r := httptest.NewRequest(http.MethodGet, "/api/config/oidc", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleGetOIDCConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp oidcConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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
	issuerURL, err := h.DB.GetSetting(context.Background(), settingOIDCIssuerURL)
	if err != nil {
		t.Fatalf("GetSetting(issuer_url) error: %v", err)
	}
	if issuerURL != oidcServer.URL {
		t.Errorf("saved issuer_url = %q, want %q", issuerURL, oidcServer.URL)
	}

	clientID, err := h.DB.GetSetting(context.Background(), settingOIDCClientID)
	if err != nil {
		t.Fatalf("GetSetting(client_id) error: %v", err)
	}
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
	_ = h.DB.SetSetting(context.Background(), settingOIDCClientSecret, "existing-secret")

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
	secret, err := h.DB.GetSetting(context.Background(), settingOIDCClientSecret)
	if err != nil {
		t.Fatalf("GetSetting(client_secret): %v", err)
	}
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

// --- HandleConfigStatus SMTP ---

func TestHandleConfigStatus_SMTPConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Only host set, no from → should not be configured
	_ = h.DB.SetSetting(context.Background(), settingSMTPHost, "smtp.example.com")

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	var resp configStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.SMTPConfigured {
		t.Error("expected SMTPConfigured=false when only host is set")
	}

	// Set from → now configured
	_ = h.DB.SetSetting(context.Background(), settingSMTPFrom, "noreply@example.com")

	r = httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w = httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.SMTPConfigured {
		t.Error("expected SMTPConfigured=true when host and from are set")
	}
}

// --- HandleSMTPConfig (GET) ---

func TestHandleGetSMTPConfig_AdminNoSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp smtpConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Host != "" {
		t.Errorf("Host = %q, want empty", resp.Host)
	}
	if resp.PasswordSet {
		t.Error("PasswordSet should be false when no password is stored")
	}
	if resp.EnvOverride {
		t.Error("EnvOverride should be false when no env vars are set")
	}
}

func TestHandleGetSMTPConfig_AdminWithSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	_ = h.DB.SetSetting(context.Background(), settingSMTPHost, "smtp.example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPPort, "465")
	_ = h.DB.SetSetting(context.Background(), settingSMTPUsername, "user@example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPPassword, "secret")
	_ = h.DB.SetSetting(context.Background(), settingSMTPFrom, "noreply@example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPTLS, "tls")

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp smtpConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Host != "smtp.example.com" {
		t.Errorf("Host = %q, want %q", resp.Host, "smtp.example.com")
	}
	if resp.Port != "465" {
		t.Errorf("Port = %q, want %q", resp.Port, "465")
	}
	if resp.Username != "user@example.com" {
		t.Errorf("Username = %q, want %q", resp.Username, "user@example.com")
	}
	if !resp.PasswordSet {
		t.Error("PasswordSet should be true when password is stored")
	}
	if resp.From != "noreply@example.com" {
		t.Errorf("From = %q, want %q", resp.From, "noreply@example.com")
	}
	if resp.TLS != "tls" {
		t.Errorf("TLS = %q, want %q", resp.TLS, "tls")
	}
}

func TestHandleGetSMTPConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleGetSMTPConfig_EnvOverride(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	t.Setenv("SMTP_HOST", "env-smtp.example.com")
	t.Setenv("SMTP_FROM", "env@example.com")

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp smtpConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Host != "env-smtp.example.com" {
		t.Errorf("Host = %q, want %q", resp.Host, "env-smtp.example.com")
	}
	if resp.From != "env@example.com" {
		t.Errorf("From = %q, want %q", resp.From, "env@example.com")
	}
	if !resp.EnvOverride {
		t.Error("EnvOverride should be true when SMTP_HOST env var is set")
	}
}

// --- HandleSMTPConfig (PUT) ---

func TestHandleSetSMTPConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"587","from":"noreply@example.com","password":"secret","tls":"starttls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_MissingHost(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_MissingFrom(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_InvalidPort(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"99999","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_InvalidTLS(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"noreply@example.com","password":"secret","tls":"invalid"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_InvalidJSON(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_Success(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"465","username":"user@example.com","password":"secret","from":"noreply@example.com","tls":"tls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify settings were persisted
	host, err := h.DB.GetSetting(context.Background(), settingSMTPHost)
	if err != nil {
		t.Fatalf("GetSetting(smtp_host) error: %v", err)
	}
	if host != "smtp.example.com" {
		t.Errorf("saved smtp_host = %q, want %q", host, "smtp.example.com")
	}

	from, err := h.DB.GetSetting(context.Background(), settingSMTPFrom)
	if err != nil {
		t.Fatalf("GetSetting(smtp_from) error: %v", err)
	}
	if from != "noreply@example.com" {
		t.Errorf("saved smtp_from = %q, want %q", from, "noreply@example.com")
	}

	port, err := h.DB.GetSetting(context.Background(), settingSMTPPort)
	if err != nil {
		t.Fatalf("GetSetting(smtp_port) error: %v", err)
	}
	if port != "465" {
		t.Errorf("saved smtp_port = %q, want %q", port, "465")
	}
}

func TestHandleSetSMTPConfig_DefaultsPortAndTLS(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	port, _ := h.DB.GetSetting(context.Background(), settingSMTPPort)
	if port != "587" {
		t.Errorf("default port = %q, want %q", port, "587")
	}

	tlsMode, _ := h.DB.GetSetting(context.Background(), settingSMTPTLS)
	if tlsMode != "starttls" {
		t.Errorf("default tls = %q, want %q", tlsMode, "starttls")
	}
}

// --- HandleSMTPConfig (dispatch) ---

func TestHandleSMTPConfig_DispatchMethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- HandleSMTPTest ---

func TestHandleSMTPTest_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleSMTPTest_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleSMTPTest_NotConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- Password handling ---

func TestHandleSetSMTPConfig_PreservesExistingPassword(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Pre-store a password
	_ = h.DB.SetSetting(context.Background(), settingSMTPPassword, "existing-pw")

	// Send request with empty password — should reuse the existing one
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"user","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	pw, err := h.DB.GetSetting(context.Background(), settingSMTPPassword)
	if err != nil {
		t.Fatalf("GetSetting(smtp_password): %v", err)
	}
	if pw != "existing-pw" {
		t.Errorf("smtp_password = %q, want %q", pw, "existing-pw")
	}
}

func TestHandleSetSMTPConfig_UnauthenticatedSMTP(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// No username, no password — should succeed (unauthenticated SMTP)
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_UsernameWithoutPasswordFails(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Username set but no password anywhere — should fail
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"user","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_InvalidFromAddress(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"not-an-email","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetSMTPConfig_FromWithDisplayName(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// "Display Name <addr>" should be rejected — only bare email addresses allowed
	body := `{"host":"smtp.example.com","from":"App <noreply@example.com>","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- HandleSMTPTest success path ---

func TestHandleSMTPTest_Success(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Configure SMTP in DB
	_ = h.DB.SetSetting(context.Background(), settingSMTPHost, "smtp.example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPPort, "587")
	_ = h.DB.SetSetting(context.Background(), settingSMTPFrom, "noreply@example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPTLS, "starttls")

	var calledFrom, calledTo string
	h.SendMailFunc = func(addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error {
		calledFrom = from
		calledTo = to
		return nil
	}

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if calledFrom != "noreply@example.com" {
		t.Errorf("sendMail from = %q, want %q", calledFrom, "noreply@example.com")
	}
	if calledTo != "admin@example.com" {
		t.Errorf("sendMail to = %q, want %q", calledTo, "admin@example.com")
	}
}

func TestHandleSMTPTest_SendMailFailure(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	_ = h.DB.SetSetting(context.Background(), settingSMTPHost, "smtp.example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPPort, "587")
	_ = h.DB.SetSetting(context.Background(), settingSMTPFrom, "noreply@example.com")
	_ = h.DB.SetSetting(context.Background(), settingSMTPTLS, "starttls")

	h.SendMailFunc = func(addr string, a smtp.Auth, from, to string, msg []byte, tlsMode string) error {
		return fmt.Errorf("connection refused")
	}

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadGateway, w.Body.String())
	}
}
