package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/jobs"
)

// mockEnqueuer records all enqueued jobs for test assertions.
type mockEnqueuer struct {
	mu   sync.Mutex
	jobs []enqueued
	err  error // if set, Enqueue returns this error
}

type enqueued struct {
	Name    string
	Payload json.RawMessage
}

func (m *mockEnqueuer) Enqueue(_ context.Context, name string, payload any) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs = append(m.jobs, enqueued{Name: name, Payload: json.RawMessage(data)})
	return "mock-job-id", nil
}

func setupLibraryHandler(t *testing.T) (*LibraryHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &LibraryHandler{DB: d}

	user, err := d.CreateUser("Test User", "test@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return h, user.ID
}

func TestCreateLibrary_ValidPath(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "Books" {
		t.Errorf("name = %q, want %q", dto.Name, "Books")
	}
	if len(dto.Paths) != 1 || dto.Paths[0] != dir {
		t.Errorf("paths = %v, want [%s]", dto.Paths, dir)
	}
}

func TestCreateLibrary_NonexistentPath(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{"/nonexistent/path/that/does/not/exist"},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg := resp["error"]; msg == "" {
		t.Error("expected error message in response")
	}
}

func TestCreateLibrary_PathIsFile(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{filePath},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg := resp["error"]; msg != "path is not a folder: "+filePath {
		t.Errorf("error = %q, want 'path is not a folder: %s'", msg, filePath)
	}
}

func TestCreateLibrary_MixedValidAndInvalidPaths(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	validDir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{validDir, "/nonexistent/path"},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateLibrary_EnqueuesScanJobs(t *testing.T) {
	h, userID := setupLibraryHandler(t)
	mock := &mockEnqueuer{}
	h.Enqueuer = mock

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
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
	if err := json.Unmarshal(mock.jobs[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(p.Paths) != 1 || p.Paths[0] != dir {
		t.Errorf("job paths = %v, want [%s]", p.Paths, dir)
	}
}

func TestCreateLibrary_EnqueuesScanJobsForMultiplePaths(t *testing.T) {
	h, userID := setupLibraryHandler(t)
	mock := &mockEnqueuer{}
	h.Enqueuer = mock

	dir1 := t.TempDir()
	dir2 := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir1, dir2},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
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
	if err := json.Unmarshal(mock.jobs[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
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
	h, userID := setupLibraryHandler(t)
	mock := &mockEnqueuer{err: fmt.Errorf("redis unavailable")}
	h.Enqueuer = mock

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestCreateLibrary_NilEnqueuerDoesNotPanic(t *testing.T) {
	h, userID := setupLibraryHandler(t)
	// h.Enqueuer is nil by default from setupLibraryHandler

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})

	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestUpdateLibrary_NonexistentPath(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	// Create a library with a valid path first.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	// Update with an invalid path.
	updateBody, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{"/nonexistent/update/path"},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusBadRequest, w2.Body.String())
	}
}

func TestUpdateLibrary_ValidPath(t *testing.T) {
	h, userID := setupLibraryHandler(t)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create with dir1.
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir1},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Update to dir2.
	updateBody, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir2},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}
