package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupConfigHandler creates a ConfigHandler with a test DB, an admin user, and
// a regular user. The IsOIDCConfigured callback defaults to always returning
// false; tests can override it.
func setupConfigHandler(t *testing.T) (*ConfigHandler, string, string) {
	t.Helper()
	d := newTestDB(t)

	admin, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	require.NoError(t, err, "create admin")
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
	require.NoError(t, err, "create regular user")

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

	require.Equal(t, http.StatusOK, w.Code)
	var resp configStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.False(t, resp.OIDCConfigured)
	require.True(t, resp.IsAdmin)
}

func TestHandleConfigStatus_RegularUser(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp configStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.False(t, resp.IsAdmin)
}

func TestHandleConfigStatus_WhenConfigured(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)
	h.IsOIDCConfigured = func() bool { return true }

	r := httptest.NewRequest(http.MethodGet, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	var resp configStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	require.True(t, resp.OIDCConfigured)
}

func TestHandleConfigStatus_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/config/status", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleConfigStatus(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
