package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func TestHandleRegistrationConfig_GetDefault(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))

	r := httptest.NewRequest(http.MethodGet, "/api/config/registration", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp registrationConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.RegistrationDisabled)
}

func TestHandleRegistrationConfig_GetAfterSet(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))
	require.NoError(t, h.DB.SetSetting(t.Context(), db.SettingRegistrationDisabled, "true"))

	r := httptest.NewRequest(http.MethodGet, "/api/config/registration", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp registrationConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.RegistrationDisabled)
}

func TestHandleRegistrationConfig_GetForbiddenForNonAdmin(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/registration", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleRegistrationConfig_PutDisable(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))

	body := mustMarshal(t, setRegistrationConfigRequest{RegistrationDisabled: true})
	r := httptest.NewRequest(http.MethodPut, "/api/config/registration", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp registrationConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.RegistrationDisabled)

	// Verify persisted.
	val, err := h.DB.GetSetting(t.Context(), db.SettingRegistrationDisabled)
	require.NoError(t, err)
	require.Equal(t, "true", val)
}

func TestHandleRegistrationConfig_PutEnable(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))
	require.NoError(t, h.DB.SetSetting(t.Context(), db.SettingRegistrationDisabled, "true"))

	body := mustMarshal(t, setRegistrationConfigRequest{RegistrationDisabled: false})
	r := httptest.NewRequest(http.MethodPut, "/api/config/registration", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp registrationConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.False(t, resp.RegistrationDisabled)

	// Verify persisted.
	val, err := h.DB.GetSetting(t.Context(), db.SettingRegistrationDisabled)
	require.NoError(t, err)
	require.Equal(t, "false", val)
}

func TestHandleRegistrationConfig_PutForbiddenForNonAdmin(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body := mustMarshal(t, setRegistrationConfigRequest{RegistrationDisabled: true})
	r := httptest.NewRequest(http.MethodPut, "/api/config/registration", bytes.NewReader(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleRegistrationConfig_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))

	r := httptest.NewRequest(http.MethodDelete, "/api/config/registration", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleRegistrationConfig(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
