package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupConfigHandler creates a ConfigHandler with a test DB, an admin user, and
// a regular user. The IsOIDCConfigured callback defaults to always returning
// false; tests can override it.
func setupConfigHandler(t *testing.T) (*ConfigHandler, string, string) {
	t.Helper()
	d := newTestDB(t)

	admin, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
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
