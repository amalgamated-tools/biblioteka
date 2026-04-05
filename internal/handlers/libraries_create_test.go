package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"

	"github.com/stretchr/testify/require"
)

func TestCreateLibrary_ValidPath(t *testing.T) {
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

	require.Equal(t, http.StatusCreated, w.Code)

	var dto libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Books", dto.Name)
	require.Len(t, dto.Paths, 1)
	require.Equal(t, dir, dto.Paths[0])
}

func TestCreateLibrary_NonexistentPath(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{"/nonexistent/path/that/does/not/exist"},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	msg := resp["error"]
	require.NotEqual(t, "", msg)
}

func TestCreateLibrary_PathIsFile(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o644), "write file")

	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{filePath},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	msg := resp["error"]
	require.Equal(t, "path is not a folder: "+filePath, msg)
}

func TestCreateLibrary_MixedValidAndInvalidPaths(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	validDir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{validDir, "/nonexistent/path"},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateLibrary_EnqueuesScanJobs(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)
	mock := &mockEnqueuer{}
	h.Enqueuer = mock

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		require.Failf(t, "unexpected status", "status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	require.Len(t, mock.jobs, 1, "enqueued jobs")
	require.Equal(t, jobs.JobScanLibrary, mock.jobs[0].Name)
	var p jobs.ScanLibraryPayload
	require.NoError(t, json.Unmarshal(mock.jobs[0].Payload, &p), "unmarshal payload")
	require.Len(t, p.Paths, 1)
	require.Equal(t, dir, p.Paths[0])
}

func TestCreateLibrary_EnqueuesScanJobsForMultiplePaths(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)
	mock := &mockEnqueuer{}
	h.Enqueuer = mock

	dir1 := t.TempDir()
	dir2 := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir1, dir2},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "status; body: %s", w.Body.String())

	require.Len(t, mock.jobs, 1, "enqueued jobs")
	require.Equal(t, jobs.JobScanLibrary, mock.jobs[0].Name)
	var p jobs.ScanLibraryPayload
	require.NoError(t, json.Unmarshal(mock.jobs[0].Payload, &p), "unmarshal payload")
	require.Len(t, p.Paths, 2, "job paths count")
	for i, dir := range []string{dir1, dir2} {
		require.Equal(t, dir, p.Paths[i])
	}
}

func TestCreateLibrary_EnqueueErrorDoesNotFailRequest(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)
	mock := &mockEnqueuer{err: fmt.Errorf("redis unavailable")}
	h.Enqueuer = mock

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateLibrary_NilEnqueuerDoesNotPanic(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)
	// h.Enqueuer is nil by default from setupLibraryHandler

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateLibrary_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupLibraryHandler(t)

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateLibrary_InvalidOrganizationType(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:             "Books",
		Paths:            []string{dir},
		OrganizationType: "invalid_type",
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	msg := resp["error"]
	require.NotEqual(t, "", msg)
}

func TestCreateLibrary_ValidOrganizationTypes(t *testing.T) {
	for _, orgType := range db.LibraryOrganizationTypeNames() {
		t.Run(orgType, func(t *testing.T) {
			h, adminID, _ := setupLibraryHandler(t)

			dir := t.TempDir()
			body := mustMarshal(t, libraryRequest{
				Name:             "Books-" + orgType,
				Paths:            []string{dir},
				OrganizationType: orgType,
			})

			r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
			r = withUserID(r, adminID)
			w := httptest.NewRecorder()

			h.HandleLibraries(w, r)

			require.Equal(t, http.StatusCreated, w.Code)

			var dto libraryDTO
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
			require.Equal(t, orgType, dto.OrganizationType)
		})
	}
}

func TestCreateLibrary_EmptyOrganizationTypeDefaultsToBookPerFolder(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body := mustMarshal(t, libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
		// OrganizationType intentionally omitted (empty string)
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, db.LibraryOrganizationBookPerFolder, dto.OrganizationType)
}
