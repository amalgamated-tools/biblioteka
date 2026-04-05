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
	require.False(t, resp.SMTPConfigured)

	// Set from → now configured
	_ = h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "noreply@example.com")

	r = httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w = httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.True(t, resp.SMTPConfigured)
}

func TestHandleConfigStatus_SMTPConfiguredWithDisplayName(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Set both host and a From address with a display name.
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyHost, "smtp.example.com"))
	require.NoError(t, h.DB.SetSetting(t.Context(), smtp.SettingKeyFrom, "\"My App\" <noreply@example.com>"))

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleConfigStatus(w, r)

	var resp configStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.True(t, resp.SMTPConfigured, "expected SMTPConfigured=true when From has a display name")
}

func TestHandleGetSMTPConfig_AdminNoSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp smtpConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "", resp.Host)
	require.False(t, resp.PasswordSet)
	require.False(t, resp.EnvOverride)
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

	require.Equal(t, http.StatusOK, w.Code)
	var resp smtpConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "smtp.example.com", resp.Host)
	require.Equal(t, "465", resp.Port)
	require.Equal(t, "user@example.com", resp.Username)
	require.True(t, resp.PasswordSet)
	require.Equal(t, "noreply@example.com", resp.From)
	require.Equal(t, "tls", resp.TLS)
}

func TestHandleGetSMTPConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleGetSMTPConfig_EnvOverride(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	t.Setenv("SMTP_HOST", "env-smtp.example.com")
	t.Setenv("SMTP_FROM", "env@example.com")

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp smtpConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.Equal(t, "env-smtp.example.com", resp.Host)
	require.Equal(t, "env@example.com", resp.From)
	require.True(t, resp.EnvOverride)
}

// --- HandleSMTPConfig (PUT) ---

func TestHandleSetSMTPConfig_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"587","from":"noreply@example.com","password":"secret","tls":"starttls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSetSMTPConfig_MissingHost(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_MissingFrom(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_InvalidPort(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"99999","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_InvalidTLS(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"noreply@example.com","password":"secret","tls":"invalid"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_InvalidJSON(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_Success(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","port":"465","username":"user@example.com","password":"secret","from":"noreply@example.com","tls":"tls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify settings were persisted
	host, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyHost)
	require.NoError(t, err, "GetSetting(smtp_host) error")
	require.Equal(t, "smtp.example.com", host)

	from, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyFrom)
	require.NoError(t, err, "GetSetting(smtp_from) error")
	require.Equal(t, "noreply@example.com", from)

	port, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPort)
	require.NoError(t, err, "GetSetting(smtp_port) error")
	require.Equal(t, "465", port)

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password) error")
	require.Equal(t, "secret", pw)
}

func TestHandleSetSMTPConfig_DefaultsPortAndTLS(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"noreply@example.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	port, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPort)
	require.NoError(t, err, "get setting")
	require.Equal(t, "587", port)

	tlsMode, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyTLS)
	require.NoError(t, err, "get setting")
	require.Equal(t, "starttls", tlsMode)
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
		_, err := h.DB.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS test_settings_fail_smtp_username_update`)
		require.NoError(t, err, "drop trigger")
	})

	body := `{"host":"smtp.example.com","port":"465","username":"user@example.com","password":"secret","from":"noreply@example.com","tls":"tls"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	for _, setting := range existing {
		value, err := h.DB.GetSetting(ctx, setting.Key)
		require.NoError(t, err, "GetSetting(%s)", setting.Key)
		require.Equal(t, setting.Value, value)
	}
}

// --- HandleSMTPConfig (dispatch) ---

func TestHandleSMTPConfig_DispatchMethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/smtp", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- HandleSMTPTest ---

func TestHandleSMTPTest_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSMTPTest_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleSMTPTest_NotConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/smtp/test", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPTest(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password)")
	require.Equal(t, "existing-pw", pw)
}

func TestHandleSetSMTPConfig_UnauthenticatedSMTP(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// No username, no password — should succeed (unauthenticated SMTP)
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)

	pw, err := h.DB.GetSetting(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err, "GetSetting(smtp_password)")
	require.Equal(t, "", pw)
}

func TestHandleSetSMTPConfig_UsernameWithoutPasswordFails(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Username set but no password anywhere — should fail
	body := `{"host":"smtp.example.com","from":"noreply@example.com","username":"user","password":""}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_InvalidFromAddress(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	body := `{"host":"smtp.example.com","from":"not-an-email","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetSMTPConfig_FromWithDisplayName(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// "Display Name <addr>" should be rejected — only bare email addresses allowed
	body := `{"host":"smtp.example.com","from":"App <noreply@example.com>","password":"secret"}`
	r := httptest.NewRequest(http.MethodPut, "/api/config/smtp", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSMTPConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "noreply@example.com", calledFrom)
	require.Equal(t, "admin@example.com", calledTo)
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

	require.Equal(t, http.StatusBadGateway, w.Code)
}
