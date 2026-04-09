package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleGetWatchFolderConfig_AdminNoSettings(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/watch-folder", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var resp watchFolderConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "", resp.Path)
	require.Equal(t, "", resp.LibraryID)
}

func TestHandleGetWatchFolderConfig_NonAdmin(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/watch-folder", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSetWatchFolderConfig_Success(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Create a library to use as target.
	lib, err := h.DB.CreateLibrary(t.Context(), "Test Library", `["/tmp"]`, "none", true)
	require.NoError(t, err)

	// Create a temp directory to use as watch folder.
	dir := t.TempDir()

	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      dir,
		LibraryID: lib.ID,
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp["message"], "saved successfully")

	// Verify settings were persisted.
	r2 := httptest.NewRequest(http.MethodGet, "/api/config/watch-folder", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleWatchFolderConfig(w2, r2)

	var getResp watchFolderConfigResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &getResp))
	require.Equal(t, dir, getResp.Path)
	require.Equal(t, lib.ID, getResp.LibraryID)
}

func TestHandleSetWatchFolderConfig_ClearConfig(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	// Set config first.
	lib, err := h.DB.CreateLibrary(t.Context(), "Test Library", `["/tmp"]`, "none", true)
	require.NoError(t, err)

	dir := t.TempDir()
	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      dir,
		LibraryID: lib.ID,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleWatchFolderConfig(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	// Now clear it.
	body2, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      "",
		LibraryID: "",
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body2))
	r2.Header.Set("Content-Type", "application/json")
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleWatchFolderConfig(w2, r2)
	require.Equal(t, http.StatusOK, w2.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	require.Contains(t, resp["message"], "cleared")

	// Verify it's cleared.
	r3 := httptest.NewRequest(http.MethodGet, "/api/config/watch-folder", nil)
	r3 = withUserID(r3, adminID)
	w3 := httptest.NewRecorder()
	h.HandleWatchFolderConfig(w3, r3)

	var getResp watchFolderConfigResponse
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &getResp))
	require.Equal(t, "", getResp.Path)
	require.Equal(t, "", getResp.LibraryID)
}

func TestHandleSetWatchFolderConfig_InvalidPath(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	lib, err := h.DB.CreateLibrary(t.Context(), "Test Library", `["/tmp"]`, "none", true)
	require.NoError(t, err)

	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      "/nonexistent/path/should/not/exist",
		LibraryID: lib.ID,
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetWatchFolderConfig_PathIsFile(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	lib, err := h.DB.CreateLibrary(t.Context(), "Test Library", `["/tmp"]`, "none", true)
	require.NoError(t, err)

	// Create a temp file (not directory).
	f := filepath.Join(t.TempDir(), "notadir.txt")
	require.NoError(t, os.WriteFile(f, []byte("test"), 0o644))

	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      f,
		LibraryID: lib.ID,
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetWatchFolderConfig_MissingLibraryID(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      dir,
		LibraryID: "",
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetWatchFolderConfig_LibraryNotFound(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      dir,
		LibraryID: "nonexistent-library-id",
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleSetWatchFolderConfig_NonAdmin(t *testing.T) {
	h, _, regularID := setupConfigHandler(t)

	body, _ := json.Marshal(setWatchFolderConfigRequest{
		Path:      "/tmp",
		LibraryID: "some-id",
	})

	r := httptest.NewRequest(http.MethodPut, "/api/config/watch-folder", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleWatchFolderConfig_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupConfigHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/watch-folder", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleWatchFolderConfig(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
