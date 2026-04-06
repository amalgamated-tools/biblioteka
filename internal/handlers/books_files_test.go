package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// createTestLibrary creates a temporary directory, registers it as a library in
// the test DB, and returns the absolute path to that directory. The library is
// cleaned up automatically when the test ends.
func createTestLibrary(t *testing.T, d *db.DB) string {
	t.Helper()
	dir := t.TempDir()
	registerTestLibrary(t, d, dir)
	return dir
}

// registerTestLibrary registers an existing directory as a library in the test DB.
func registerTestLibrary(t *testing.T, d *db.DB, dir string) {
	t.Helper()
	pathsJSON, err := json.Marshal([]string{dir})
	require.NoError(t, err, "marshal library paths")
	_, err = d.CreateLibrary(t.Context(), "Test Library", string(pathsJSON), db.LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "create test library")
}

func TestGetBookFiles_Empty(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 0)
}

func TestGetBookFiles_WithFiles(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), b.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var files []bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &files), "unmarshal")
	require.Len(t, files, 1)
	require.Equal(t, "gunslinger.epub", files[0].FileName)
}

func TestPostBookFiles_Success(t *testing.T) {
	h, userID := setupBookHandler(t)

	dir := createTestLibrary(t, h.DB)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FileSize: 2048,
		FilePath: filepath.Join(dir, "gunslinger.epub"),
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	var dto bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "gunslinger.epub", dto.FileName)
	require.Equal(t, "epub", dto.FileType)
	require.Equal(t, int64(2048), dto.FileSize)
	require.Equal(t, b.ID, dto.BookID)
}

func TestPostBookFiles_MissingFileType(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileName: "gunslinger.epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_MissingFileName(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_MissingFilePath(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_InvalidJSON(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", strings.NewReader("not-json"))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBookFiles_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodPut, "/api/books/"+b.ID+"/files", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPostBookFiles_AuditLog(t *testing.T) {
	h, userID := setupBookHandler(t)

	dir := createTestLibrary(t, h.DB)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FilePath: filepath.Join(dir, "gunslinger.epub"),
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == "book_file.created" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestPostBookFiles_PathOutsideLibrary(t *testing.T) {
	h, userID := setupBookHandler(t)
	createTestLibrary(t, h.DB)

	// Create a second temp dir that is NOT registered as a library root.
	outsideDir := t.TempDir()

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FilePath: filepath.Join(outsideDir, "gunslinger.epub"),
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_PathTraversal(t *testing.T) {
	h, userID := setupBookHandler(t)
	dir := createTestLibrary(t, h.DB)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// Create a temp dir outside the library root to use as the traversal target.
	outsideDir := t.TempDir()

	// Attempt path traversal from within the library root to the outside dir.
	// Build a relative path that escapes the library directory.
	relPath, err := filepath.Rel(dir, outsideDir)
	require.NoError(t, err, "compute relative path")

	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "passwd",
		FilePath: filepath.Join(dir, relPath, "evil.epub"),
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostBookFiles_NoLibraries(t *testing.T) {
	h, userID := setupBookHandler(t)

	b, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	// No libraries are configured; any path should be rejected.
	body := mustMarshal(t, createBookFileRequest{
		FileType: "epub",
		FileName: "gunslinger.epub",
		FilePath: "/books/gunslinger.epub",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/books/"+b.ID+"/files", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
