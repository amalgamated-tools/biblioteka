package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper: creates an admin (first user) and a non-admin (second user) in the given DB.
func setupUsers(t *testing.T) (*ArrServiceHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &ArrServiceHandler{DB: d}

	// First user is auto-promoted to admin
	admin, err := d.CreateUser("Admin", "admin@example.com", "password1")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	// Second user is a regular user
	regular, err := d.CreateUser("Regular", "regular@example.com", "password1")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}

	return h, admin.ID, regular.ID
}

var validServiceBody = `{"name":"My Radarr","type":"radarr","url":"http://radarr:7878","api_key":"testkey123"}`

// --- List (GET /api/arr-services) ---

func TestListArrServices_AnyUserCanList(t *testing.T) {
	h, _, regularID := setupUsers(t)

	r := httptest.NewRequest(http.MethodGet, "/api/arr-services", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleArrServices(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var services []arrServiceDTO
	if err := json.Unmarshal(w.Body.Bytes(), &services); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}

func TestListArrServices_ReturnsServicesCreatedByAdmin(t *testing.T) {
	h, adminID, regularID := setupUsers(t)

	// Admin creates a service
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create: status = %d; body: %s", w.Code, w.Body.String())
	}

	// Regular user can see it
	r2 := httptest.NewRequest(http.MethodGet, "/api/arr-services", nil)
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()
	h.HandleArrServices(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusOK)
	}
	var services []arrServiceDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &services); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
}

// --- Create (POST /api/arr-services) ---

func TestCreateArrService_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupUsers(t)

	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleArrServices(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var svc arrServiceDTO
	if err := json.Unmarshal(w.Body.Bytes(), &svc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if svc.Name != "My Radarr" {
		t.Errorf("name = %q, want %q", svc.Name, "My Radarr")
	}
	if svc.Type != "radarr" {
		t.Errorf("type = %q, want %q", svc.Type, "radarr")
	}
}

func TestCreateArrService_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupUsers(t)

	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleArrServices(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

// --- Get (GET /api/arr-services/{id}) ---

func TestGetArrService_AnyUserCanGet(t *testing.T) {
	h, adminID, regularID := setupUsers(t)

	// Admin creates a service
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	var created arrServiceDTO
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Regular user can get it by ID
	r2 := httptest.NewRequest(http.MethodGet, "/api/arr-services/"+created.ID, nil)
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()
	h.HandleArrService(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var svc arrServiceDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &svc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if svc.ID != created.ID {
		t.Errorf("id = %q, want %q", svc.ID, created.ID)
	}
}

// --- Update (PUT /api/arr-services/{id}) ---

func TestUpdateArrService_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupUsers(t)

	// Create
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	var created arrServiceDTO
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Update
	updateBody := `{"name":"Updated Radarr","type":"radarr","url":"http://radarr:7878","api_key":"newkey"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/arr-services/"+created.ID, bytes.NewBufferString(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleArrService(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
	var updated arrServiceDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updated.Name != "Updated Radarr" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated Radarr")
	}
}

func TestUpdateArrService_NonAdminForbidden(t *testing.T) {
	h, adminID, regularID := setupUsers(t)

	// Admin creates
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	var created arrServiceDTO
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Non-admin tries to update
	updateBody := `{"name":"Hacked","type":"radarr","url":"http://evil","api_key":"key"}`
	r2 := httptest.NewRequest(http.MethodPut, "/api/arr-services/"+created.ID, bytes.NewBufferString(updateBody))
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()
	h.HandleArrService(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusForbidden, w2.Body.String())
	}
}

// --- Delete (DELETE /api/arr-services/{id}) ---

func TestDeleteArrService_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupUsers(t)

	// Create
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	var created arrServiceDTO
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Delete
	r2 := httptest.NewRequest(http.MethodDelete, "/api/arr-services/"+created.ID, nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleArrService(w2, r2)

	if w2.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusNoContent, w2.Body.String())
	}

	// Verify it's gone
	r3 := httptest.NewRequest(http.MethodGet, "/api/arr-services/"+created.ID, nil)
	r3 = withUserID(r3, adminID)
	w3 := httptest.NewRecorder()
	h.HandleArrService(w3, r3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", w3.Code, http.StatusNotFound)
	}
}

func TestDeleteArrService_NonAdminForbidden(t *testing.T) {
	h, adminID, regularID := setupUsers(t)

	// Admin creates
	r := httptest.NewRequest(http.MethodPost, "/api/arr-services", bytes.NewBufferString(validServiceBody))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleArrServices(w, r)
	var created arrServiceDTO
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Non-admin tries to delete
	r2 := httptest.NewRequest(http.MethodDelete, "/api/arr-services/"+created.ID, nil)
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()
	h.HandleArrService(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusForbidden, w2.Body.String())
	}

	// Verify service still exists
	r3 := httptest.NewRequest(http.MethodGet, "/api/arr-services/"+created.ID, nil)
	r3 = withUserID(r3, adminID)
	w3 := httptest.NewRecorder()
	h.HandleArrService(w3, r3)

	if w3.Code != http.StatusOK {
		t.Errorf("service should still exist: status = %d, want %d", w3.Code, http.StatusOK)
	}
}
