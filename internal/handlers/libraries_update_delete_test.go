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

func TestUpdateLibrary_NonexistentPath(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library with a valid path first.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")

	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal created")

	// Update with an invalid path.
	updateBody := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{"/nonexistent/update/path"},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusBadRequest, w2.Body.String())
	}
}

func TestUpdateLibrary_ValidPath(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create with dir1.
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir1},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")

	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal")

	// Update to dir2.
	updateBody := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir2},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestUpdateLibrary_NonAdminForbidden(t *testing.T) {
	h, adminID, regularID := setupLibraryHandler(t)

	// Create a library as admin first.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "Books", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")
	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal")

	// Attempt update as regular user.
	updateBody := mustMarshal(t, libraryRequest{Name: "Updated", Paths: []string{dir}})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusForbidden, w2.Body.String())
	}
}

func TestDeleteLibrary_NonAdminForbidden(t *testing.T) {
	h, adminID, regularID := setupLibraryHandler(t)

	// Create a library as admin first.
	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{Name: "Books", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")
	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal")

	// Attempt delete as regular user.
	r2 := httptest.NewRequest(http.MethodDelete, "/api/libraries/"+created.ID, nil)
	r2 = withUserID(r2, regularID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusForbidden, w2.Body.String())
	}
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/libraries/nonexistent-id", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibrary(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if resp.Error != "library not found" {
		t.Errorf("error = %q, want %q", resp.Error, "library not found")
	}
}

func TestUpdateLibrary_InvalidOrganizationType(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")

	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal created")

	updateBody := mustMarshal(t, libraryRequest{
		Name:             "Books",
		Paths:            []string{dir},
		OrganizationType: "flat_organize",
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusBadRequest, w2.Body.String())
	}
}

func TestUpdateLibrary_EmptyOrganizationTypePreservesExistingValue(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:             "Books",
		Paths:            []string{dir},
		OrganizationType: db.LibraryOrganizationNone,
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "setup: create library")

	var created libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created), "unmarshal created")

	updateBody := mustMarshal(t, libraryRequest{
		Name:  "Books Updated",
		Paths: []string{dir},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	require.Equal(t, http.StatusOK, w2.Code, "update library status")

	var updated libraryDTO
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &updated), "unmarshal updated")
	if updated.OrganizationType != db.LibraryOrganizationNone {
		t.Errorf("organization_type = %q, want %q", updated.OrganizationType, db.LibraryOrganizationNone)
	}
}
