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

	"github.com/amalgamated-tools/biblioteka/internal/db"
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

func setupLibraryHandler(t *testing.T) (*LibraryHandler, string, string) {
	t.Helper()
	d := newTestDB(t)
	h := &LibraryHandler{DB: d}

	admin, err := d.CreateUser(context.Background(), "Admin", "admin@example.com", "password1")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := d.SetAdmin(context.Background(), admin.ID, true); err != nil {
		t.Fatalf("set admin role: %v", err)
	}
	regular, err := d.CreateUser(context.Background(), "Regular", "regular@example.com", "password1")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}
	return h, admin.ID, regular.ID
}

func TestCreateLibrary_ValidPath(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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
	h, adminID, _ := setupLibraryHandler(t)

	body, _ := json.Marshal(libraryRequest{
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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg := resp["error"]; msg == "" {
		t.Error("expected error message in response")
	}
}

func TestCreateLibrary_PathIsFile(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	body, _ := json.Marshal(libraryRequest{
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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg := resp["error"]; msg != "path is not a folder: "+filePath {
		t.Errorf("error = %q, want 'path is not a folder: %s'", msg, filePath)
	}
}

func TestCreateLibrary_MixedValidAndInvalidPaths(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	validDir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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
	body, _ := json.Marshal(libraryRequest{
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
	if err := json.Unmarshal(mock.jobs[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
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
	body, _ := json.Marshal(libraryRequest{
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
	h, adminID, _ := setupLibraryHandler(t)
	mock := &mockEnqueuer{err: fmt.Errorf("redis unavailable")}
	h.Enqueuer = mock

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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
	body, _ := json.Marshal(libraryRequest{
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

func TestListLibraryBooks_Success(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "Fiction", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create library: status = %d; body: %s", w.Code, w.Body.String())
	}
	var lib libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &lib); err != nil {
		t.Fatalf("unmarshal library: %v", err)
	}

	// Create a book and link it to the library.
	book, err := h.DB.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}
	if err := h.DB.AddBookToLibrary(context.Background(), lib.ID, book.ID); err != nil {
		t.Fatalf("add book to library: %v", err)
	}

	// List books for the library.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal books: %v", err)
	}
	if len(resp.Books) != 1 {
		t.Fatalf("books count = %d, want 1", len(resp.Books))
	}
	if resp.Books[0].Title != "The Gunslinger" {
		t.Errorf("title = %q, want %q", resp.Books[0].Title, "The Gunslinger")
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
}

func TestListLibraryBooks_PaginationValid(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "Paginated", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create library: status = %d; body: %s", w.Code, w.Body.String())
	}
	var lib libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &lib); err != nil {
		t.Fatalf("unmarshal library: %v", err)
	}

	// Create multiple books and link them to the library.
	const totalBooks = 3
	for i := range totalBooks {
		title := fmt.Sprintf("Book %d", i+1)
		book, err := h.DB.CreateBook(context.Background(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("create book %d: %v", i+1, err)
		}
		if err := h.DB.AddBookToLibrary(context.Background(), lib.ID, book.ID); err != nil {
			t.Fatalf("add book %d to library: %v", i+1, err)
		}
	}

	// Request a paginated slice of books.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=2&offset=1", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal books: %v", err)
	}

	if resp.Total != totalBooks {
		t.Fatalf("total = %d, want %d", resp.Total, totalBooks)
	}
	if len(resp.Books) == 0 || len(resp.Books) > 2 {
		t.Fatalf("books count = %d, want between 1 and 2", len(resp.Books))
	}
}

func TestListLibraryBooks_PaginationInvalidValues(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "InvalidPagination", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create library: status = %d; body: %s", w.Code, w.Body.String())
	}
	var lib libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &lib); err != nil {
		t.Fatalf("unmarshal library: %v", err)
	}

	// Create multiple books and link them to the library.
	const totalBooks = 3
	for i := range totalBooks {
		title := fmt.Sprintf("Invalid Book %d", i+1)
		book, err := h.DB.CreateBook(context.Background(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("create book %d: %v", i+1, err)
		}
		if err := h.DB.AddBookToLibrary(context.Background(), lib.ID, book.ID); err != nil {
			t.Fatalf("add book %d to library: %v", i+1, err)
		}
	}

	// Use negative values that should be validated/clamped by the handler.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=-5&offset=-10", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal books: %v", err)
	}

	if resp.Total != totalBooks {
		t.Fatalf("total = %d, want %d", resp.Total, totalBooks)
	}
	if len(resp.Books) == 0 {
		t.Fatalf("books count = %d, want > 0", len(resp.Books))
	}
}

func TestListLibraryBooks_PaginationMaxLimitClamping(t *testing.T) {
	h, userID, _ := setupLibraryHandler(t)

	// Create a library.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "MaxLimit", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create library: status = %d; body: %s", w.Code, w.Body.String())
	}
	var lib libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &lib); err != nil {
		t.Fatalf("unmarshal library: %v", err)
	}

	// Create several books and link them to the library.
	const totalBooks = 10
	for i := range totalBooks {
		title := fmt.Sprintf("Clamped Book %d", i+1)
		book, err := h.DB.CreateBook(context.Background(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("create book %d: %v", i+1, err)
		}
		if err := h.DB.AddBookToLibrary(context.Background(), lib.ID, book.ID); err != nil {
			t.Fatalf("add book %d to library: %v", i+1, err)
		}
	}

	// Request with an excessively large limit to ensure it is clamped internally.
	r2 := httptest.NewRequest(http.MethodGet, "/api/libraries/"+lib.ID+"/books?limit=999999&offset=0", nil)
	r2 = withUserID(r2, userID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var resp bookListDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal books: %v", err)
	}

	if resp.Total != totalBooks {
		t.Fatalf("total = %d, want %d", resp.Total, totalBooks)
	}
	if len(resp.Books) != totalBooks {
		t.Fatalf("books count = %d, want %d", len(resp.Books), totalBooks)
	}
}

func TestListLibraryBooks_NotFound(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/libraries/nonexistent-id/books", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibrary(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestListLibraryBooks_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library first.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "Fiction", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create library: status = %d; body: %s", w.Code, w.Body.String())
	}
	var lib libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &lib); err != nil {
		t.Fatalf("unmarshal library: %v", err)
	}

	// POST to /books sub-resource should be method not allowed.
	r2 := httptest.NewRequest(http.MethodPost, "/api/libraries/"+lib.ID+"/books", nil)
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()
	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusMethodNotAllowed, w2.Body.String())
	}
}

func TestUpdateLibrary_NonexistentPath(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	// Create a library with a valid path first.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
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
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir1},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
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
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}
}

func TestCreateLibrary_NonAdminForbidden(t *testing.T) {
	h, _, regularID := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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

func TestUpdateLibrary_NonAdminForbidden(t *testing.T) {
	h, adminID, regularID := setupLibraryHandler(t)

	// Create a library as admin first.
	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{Name: "Books", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d; body: %s", w.Code, w.Body.String())
	}
	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Attempt update as regular user.
	updateBody, _ := json.Marshal(libraryRequest{Name: "Updated", Paths: []string{dir}})
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
	body, _ := json.Marshal(libraryRequest{Name: "Books", Paths: []string{dir}})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d; body: %s", w.Code, w.Body.String())
	}
	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != "library not found" {
		t.Errorf("error = %q, want %q", resp.Error, "library not found")
	}
}

func TestListLibraries_NonAdminAllowed(t *testing.T) {
	h, _, regularID := setupLibraryHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/libraries", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleLibraries(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestCreateLibrary_InvalidOrganizationType(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg := resp["error"]; msg == "" {
		t.Error("expected error message about organization_type")
	}
}

func TestCreateLibrary_ValidOrganizationTypes(t *testing.T) {
	for _, orgType := range db.LibraryOrganizationTypeNames() {
		t.Run(orgType, func(t *testing.T) {
			h, adminID, _ := setupLibraryHandler(t)

			dir := t.TempDir()
			body, _ := json.Marshal(libraryRequest{
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
			if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if dto.OrganizationType != orgType {
				t.Errorf("organization_type = %q, want %q", dto.OrganizationType, orgType)
			}
		})
	}
}

func TestCreateLibrary_EmptyOrganizationTypeDefaultsToBookPerFolder(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
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
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.OrganizationType != db.LibraryOrganizationBookPerFolder {
		t.Errorf("organization_type = %q, want %q", dto.OrganizationType, db.LibraryOrganizationBookPerFolder)
	}
}

func TestUpdateLibrary_InvalidOrganizationType(t *testing.T) {
	h, adminID, _ := setupLibraryHandler(t)

	dir := t.TempDir()
	body, _ := json.Marshal(libraryRequest{
		Name:  "Books",
		Paths: []string{dir},
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	updateBody, _ := json.Marshal(libraryRequest{
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
	body, _ := json.Marshal(libraryRequest{
		Name:             "Books",
		Paths:            []string{dir},
		OrganizationType: db.LibraryOrganizationNone,
	})
	r := httptest.NewRequest(http.MethodPost, "/api/libraries", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()
	h.HandleLibraries(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup: status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created libraryDTO
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}

	updateBody, _ := json.Marshal(libraryRequest{
		Name:  "Books Updated",
		Paths: []string{dir},
	})
	r2 := httptest.NewRequest(http.MethodPut, "/api/libraries/"+created.ID, bytes.NewReader(updateBody))
	r2 = withUserID(r2, adminID)
	w2 := httptest.NewRecorder()

	h.HandleLibrary(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var updated libraryDTO
	if err := json.Unmarshal(w2.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal updated: %v", err)
	}
	if updated.OrganizationType != db.LibraryOrganizationNone {
		t.Errorf("organization_type = %q, want %q", updated.OrganizationType, db.LibraryOrganizationNone)
	}
}
