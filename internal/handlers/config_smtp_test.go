package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	netsmtp "net/smtp"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"

	"github.com/stretchr/testify/require"
)

// --- HandleConfigStatus SMTP ---

func TestHandleConfigStatus_SMTPConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Only host set, no from → should not be configured
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com")

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	var resp configStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.SMTPConfigured {
		t.Error("expected SMTPConfigured=false when only host is set")
	}

	// Set from → now configured
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com")

	r = httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w = httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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

	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPort, "465")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyUsername, "user@example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPassword, "secret")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyTLS, "tls")

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp smtpConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
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
	host, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyHost)
	require.NoError(t, err, "GetSetting(smtp_host) error")
	if host != "smtp.example.com" {
		t.Errorf("saved smtp_host = %q, want %q", host, "smtp.example.com")
	}

	from, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyFrom)
	require.NoError(t, err, "GetSetting(smtp_from) error")
	if from != "noreply@example.com" {
		t.Errorf("saved smtp_from = %q, want %q", from, "noreply@example.com")
	}

	port, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPort)
	require.NoError(t, err, "GetSetting(smtp_port) error")
	if port != "465" {
		t.Errorf("saved smtp_port = %q, want %q", port, "465")
	}

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password) error")
	if pw != "secret" {
		t.Errorf("saved smtp_password = %q, want %q", pw, "secret")
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

	port, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPort)
	require.NoError(t, err, "get setting")
	if port != "587" {
		t.Errorf("default port = %q, want %q", port, "587")
	}

	tlsMode, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyTLS)
	require.NoError(t, err, "get setting")
	if tlsMode != "starttls" {
		t.Errorf("default tls = %q, want %q", tlsMode, "starttls")
	}
}

func TestHandleSetSMTPConfig_RollsBackOnSaveError(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	ctx := t.Context()
	existing := []db.Setting{
		{Key: smtp.SettingKeyHost, Value: "old.example.com"},
		{Key: smtp.SettingKeyPort, Value: "587"},
		{Key: smtp.SettingKeyUsername, Value: "old-user"},
		{Key: smtp.SettingKeyPassword, Value: "old-secret"},
		{Key: smtp.SettingKeyFrom, Value: "old@example.com"},
		{Key: smtp.SettingKeyTLS, Value: "starttls"},
	}
	for _, setting := range existing {
		require.NoError(t, h.DB.SetSetting(ctx, setting.Key, setting.Value), "SetSetting(%s)", setting.Key)
	}

	if _, err := h.DB.ExecContext(ctx, fmt.Sprintf(`
		CREATE TRIGGER test_settings_fail_smtp_username_update
		BEFORE UPDATE ON settings
		WHEN NEW.key = '%s'
		BEGIN
			SELECT RAISE(FAIL, 'forced smtp save failure');
		END;
	`, smtp.SettingKeyUsername)); err != nil {
		require.NoError(t, err, "create trigger")
	}
	t.Cleanup(func() {
		if _, err := h.DB.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS test_settings_fail_smtp_username_update`); err != nil {
			require.NoError(t, err, "drop trigger")
		}
	})

	body := `{"host":"smtp.example.com","port":"465","username":"user@example.com","password":"secret","from":"noreply@example.com","tls":"tls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusInternalServerError {
		require.Failf(t, "failed", "status = %d, want %d; body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}

	for _, setting := range existing {
		value, err := h.DB.GetSetting(ctx, setting.Key)
		require.NoError(t, err, "GetSetting(%s)", setting.Key)
		if value != setting.Value {
			t.Errorf("setting %s = %q, want %q after rollback", setting.Key, value, setting.Value)
		}
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

	// Pre-store a username and password
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyUsername, "user")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPassword, "existing-pw")

	// Send request with empty password — should reuse the existing one
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"user","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password)")
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

func TestHandleSetSMTPConfig_UnauthenticatedSMTP_ClearsExistingPassword(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Pre-store credentials
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPassword, "old-secret")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyUsername, "old-user")

	// Switch to unauthenticated — both fields intentionally empty
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password)")
	if pw != "" {
		t.Errorf("smtp_password = %q, want empty after switching to unauthenticated SMTP", pw)
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
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPort, "587")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyTLS, "starttls")

	var calledFrom, calledTo string
	h.SendMailFunc = func(_ context.Context, addr string, a netsmtp.Auth, from, to string, msg []byte, tlsMode string) error {
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

	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyPort, "587")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com")
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyTLS, "starttls")

	h.SendMailFunc = func(_ context.Context, addr string, a netsmtp.Auth, from, to string, msg []byte, tlsMode string) error {
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
