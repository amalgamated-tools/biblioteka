package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// setupAdminHandler creates a DB with an admin user (first user) and a regular user,
// and returns a handler, the admin ID, and the regular user ID.
func setupAdminHandler(t *testing.T) (*AdminHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &AdminHandler{DB: d}

	admin, err := d.CreateUser(t.Context(), "Admin", "admin@example.com", "password1")
	require.NoError(t, err, "create admin")
	regular, err := d.CreateUser(t.Context(), "Regular", "regular@example.com", "password1")
	require.NoError(t, err, "create regular user")
	return h, admin.ID, regular.ID
}

// --- HandleListUsers ---

func TestHandleListUsers_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var users []adminUserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users), "unmarshal")
	require.Len(t, users, 2)
}

func TestHandleListUsers_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleListUsers_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleListUsers_ResponseContainsAdminFlag(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	var users []adminUserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users), "unmarshal")

	var foundAdmin, foundRegular bool
	for _, u := range users {
		if u.Email == "admin@example.com" {
			foundAdmin = true
			require.True(t, u.IsAdmin)
		}
		if u.Email == "regular@example.com" {
			foundRegular = true
			require.False(t, u.IsAdmin)
		}
	}
	require.True(t, foundAdmin)
	require.True(t, foundRegular)
}

func TestHandleListUsers_OIDCLinkedField(t *testing.T) {
	d := newTestDB(t)
	h := &AdminHandler{DB: d}

	local, err := d.CreateUser(t.Context(), "Local User", "local@example.com", "password1")
	require.NoError(t, err, "create local user")
	oidcUser, err := d.CreateOIDCUser(t.Context(), "OIDC User", "oidc@example.com", "oidc-subject-123")
	require.NoError(t, err, "create OIDC user")
	// First user is auto-admin; promote the local user so we can query.
	_ = d.SetAdmin(t.Context(), local.ID, true)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	r = withUserID(r, local.ID)
	w := httptest.NewRecorder()

	h.HandleListUsers(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var users []adminUserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users), "unmarshal")

	for _, u := range users {
		switch u.ID {
		case local.ID:
			require.False(t, u.OIDCLinked)
		case oidcUser.ID:
			require.True(t, u.OIDCLinked)
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

	require.Equal(t, http.StatusOK, w.Code)

	isAdmin, err := h.DB.IsAdmin(t.Context(), regularID)
	require.NoError(t, err, "IsAdmin() error")
	require.True(t, isAdmin)
}

func TestHandleSetAdmin_AdminDemotesUser(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	// Promote first
	_ = h.DB.SetAdmin(t.Context(), regularID, true)

	body := `{"is_admin":false}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	isAdmin, err := h.DB.IsAdmin(t.Context(), regularID)
	require.NoError(t, err, "check admin")
	require.False(t, isAdmin)
}

func TestHandleSetAdmin_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAdminHandler(t)

	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSetAdmin_CannotChangeSelf(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	body := `{"is_admin":false}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+adminID, bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetAdmin_UserNotFound(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	body := `{"is_admin":true}`
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/nonexistent-id", bytes.NewBufferString(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSetAdmin_InvalidBody(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+regularID, bytes.NewBufferString("not json"))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetAdmin_MethodNotAllowed(t *testing.T) {
	h, adminID, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+regularID, nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleSetAdmin_InvalidPath(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	// Path with no ID segment
	r := httptest.NewRequest(http.MethodPut, "/api/admin/users/", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleSetAdmin(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// --- HandleFTSRebuild ---

func TestHandleFTSRebuild_AdminSuccess(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/search/reindex", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleFTSRebuild(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "search index rebuilt", resp["message"])
}

func TestHandleFTSRebuild_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/admin/search/reindex", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleFTSRebuild(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleFTSRebuild_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/admin/search/reindex", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleFTSRebuild(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleFTSRebuild_RebuildPreservesSearchResults(t *testing.T) {
	h, adminID, _ := setupAdminHandler(t)

	_, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook()")

	r := httptest.NewRequest(http.MethodPost, "/api/admin/search/reindex", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleFTSRebuild(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	// Verify search still returns results after rebuild.
	books, total, err := h.DB.SearchBooks(t.Context(), "Foundation", 10, 0)
	require.NoError(t, err, "SearchBooks() after rebuild")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
}
