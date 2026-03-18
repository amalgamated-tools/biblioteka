package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setupAdminHandler creates a DB with an admin user (first user) and a regular user,
// and returns a handler, the admin ID, and the regular user ID.
func setupAdminHandler(t *testing.T) (*AdminHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &AdminHandler{DB: d}

	admin, err := d.CreateUser(context.Background(), "Admin", "admin@example.com", "password1")
	if err != nil {
		failNowf(t, "create admin: %v", err)
	}
	regular, err := d.CreateUser(context.Background(), "Regular", "regular@example.com", "password1")
	if err != nil {
		failNowf(t, "create regular user: %v", err)
	}
	return h, admin.ID, regular.ID
}

// --- HandleListUsers ---

func TestHandleListUsers_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var users []adminUserDTO
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}
	if len(users) != 2 {
		failf(t, "expected 2 users, got %d", len(users))
	}
}

func TestHandleListUsers_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	if w.Code != http.StatusForbidden {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleListUsers_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		failf(t, "status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleListUsers_ResponseContainsAdminFlag(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	var users []adminUserDTO
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}

	var foundAdmin, foundRegular bool
	for _, u := range users {
		if u.Email == "admin@example.com" {
			foundAdmin = true
			if !u.IsAdmin {
				fail(t, "admin user should have is_admin=true")
			}
		}
		if u.Email == "regular@example.com" {
			foundRegular = true
			if u.IsAdmin {
				fail(t, "regular user should have is_admin=false")
			}
		}
	}
	if !foundAdmin {
		fail(t, "admin user not found in response")
	}
	if !foundRegular {
		fail(t, "regular user not found in response")
	}
}

func TestHandleListUsers_OIDCLinkedField(t *testing.T) {
	d := newTestDB(t)
	h := &AdminHandler{DB: d}

	local, err := d.CreateUser(context.Background(), "Local User", "local@example.com", "password1")
	if err != nil {
		failNowf(t, "create local user: %v", err)
	}
	oidcUser, err := d.CreateOIDCUser(context.Background(), "OIDC User", "oidc@example.com", "oidc-subject-123")
	if err != nil {
		failNowf(t, "create OIDC user: %v", err)
	}
	// First user is auto-admin; promote the local user so we can query.
	_ = d.SetAdmin(context.Background(), local.ID, true)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, local.ID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	if w.Code != http.StatusOK {
		failNowf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var users []adminUserDTO
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		failNowf(t, "unmarshal: %v", err)
	}

	for _, u := range users {
		switch u.ID {
		case local.ID:
			if u.OIDCLinked {
				fail(t, "local user should have oidc_linked=false")
			}
		case oidcUser.ID:
			if !u.OIDCLinked {
				fail(t, "OIDC user should have oidc_linked=true")
			}
		}
	}
}

// --- HandleSetAdmin ---

func TestHandleSetAdmin_AdminPromotesUser(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	isAdmin, err := h.DB.IsAdmin(context.Background(), regularID)
	if err != nil {
		failNowf(t, "IsAdmin() error: %v", err)
	}
	if !isAdmin {
		fail(t, "regular user should now be admin")
	}
}

func TestHandleSetAdmin_AdminDemotesUser(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	// Promote first
	_ = h.DB.SetAdmin(context.Background(), regularID, true)

	body := `{"is_admin":false}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusOK {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	isAdmin, _ := h.DB.IsAdmin(context.Background(), regularID)
	if isAdmin {
		fail(t, "user should no longer be admin")
	}
}

func TestHandleSetAdmin_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAdminHandler(t)

	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusForbidden {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleSetAdmin_CannotChangeSelf(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	body := `{"is_admin":false}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+adminID, bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusBadRequest {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetAdmin_UserNotFound(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/nonexistent-id", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusNotFound {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestHandleSetAdmin_InvalidBody(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusBadRequest {
		failf(t, "status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleSetAdmin_MethodNotAllowed(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+regularID, nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		failf(t, "status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleSetAdmin_InvalidPath(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	// Path with no ID segment
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	if w.Code != http.StatusBadRequest {
		failf(t, "status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
