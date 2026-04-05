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

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	if dto.Name != "Books" {
		t.Errorf("name = %q, want %q", dto.Name, "Books")
	}
	if len(dto.Paths) != 1 || dto.Paths[0] != dir {
		t.Errorf("paths = %v, want [%s]", dto.Paths, dir)
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if msg := resp["error"]; msg == "" {
		t.Error("expected error message in response")
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if msg := resp["error"]; msg != "path is not a folder: "+filePath {
		t.Errorf("error = %q, want 'path is not a folder: %s'", msg, filePath)
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
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
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	if len(mock.jobs) != 1 {
		t.Fatalf("enqueued jobs = %d, want 1", len(mock.jobs))
	}
	if mock.jobs[0].Name != jobs.JobScanLibrary {
		t.Errorf("job name = %q, want %q", mock.jobs[0].Name, jobs.JobScanLibrary)
	}
	var p jobs.ScanLibraryPayload
	require.NoError(t, json.Unmarshal(mock.jobs[0].Payload, &p), "unmarshal payload")
	if len(p.Paths) != 1 || p.Paths[0] != dir {
		t.Errorf("job paths = %v, want [%s]", p.Paths, dir)
	}
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

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	if len(mock.jobs) != 1 {
		t.Fatalf("enqueued jobs = %d, want 1", len(mock.jobs))
	}
	if mock.jobs[0].Name != jobs.JobScanLibrary {
		t.Errorf("job name = %q, want %q", mock.jobs[0].Name, jobs.JobScanLibrary)
	}
	var p jobs.ScanLibraryPayload
	require.NoError(t, json.Unmarshal(mock.jobs[0].Payload, &p), "unmarshal payload")
	if len(p.Paths) != 2 {
		t.Fatalf("job paths count = %d, want 2", len(p.Paths))
	}
	for i, dir := range []string{dir1, dir2} {
		if p.Paths[i] != dir {
			t.Errorf("job paths[%d] = %q, want %q", i, p.Paths[i], dir)
		}
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

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
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

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
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

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal")
	if msg := resp["error"]; msg == "" {
		t.Error("expected error message about organization_type")
	}
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

			if w.Code != http.StatusCreated {
				t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
			}

			var dto libraryDTO
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
			if dto.OrganizationType != orgType {
				t.Errorf("organization_type = %q, want %q", dto.OrganizationType, orgType)
			}
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

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto libraryDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	if dto.OrganizationType != db.LibraryOrganizationBookPerFolder {
		t.Errorf("organization_type = %q, want %q", dto.OrganizationType, db.LibraryOrganizationBookPerFolder)
	}
}
