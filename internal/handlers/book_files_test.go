package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupBookFileHandler(t *testing.T) (*BookFileHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &BookFileHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test User", "test@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func TestGetBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dto bookFileDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "gunslinger.epub", dto.FileName)
}

func TestGetBookFile_NotFound(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/nonexistent", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBookFile_Handler(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodDelete, "/api/book-files/"+bf.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestDownloadBookFile_NotFound(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/nonexistent/download", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadBookFile_MethodNotAllowed(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/api/book-files/some-id/download", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestDownloadBookFile_FileNotOnDisk(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	// Create a library with a path that contains the book file path.
	libDir := t.TempDir()
	_, err := h.DB.CreateLibrary(t.Context(), "test-lib", `["`+libDir+`"]`, "none", false)
	require.NoError(t, err, "create library")

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	filePath := filepath.Join(libDir, "gunslinger.epub")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID+"/download", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	// File doesn't exist on disk, so we get a 404.
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDownloadBookFile_PathForbidden(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	// No libraries defined, so any path is outside allowed directories.
	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "create book")
	outsideDir := t.TempDir()
	filePath := filepath.Join(outsideDir, "gunslinger.epub")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, filePath)
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID+"/download", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDownloadBookFile_Success(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	// Create a temp directory and file.
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-book.epub")
	err := os.WriteFile(filePath, []byte("fake epub content"), 0o644)
	require.NoError(t, err, "write test file")

	// Create a library with the temp dir as root.
	_, err = h.DB.CreateLibrary(t.Context(), "test-lib", `["`+tmpDir+`"]`, "none", false)
	require.NoError(t, err, "create library")

	book, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err, "create book")
	bf, err := h.DB.CreateBookFile(t.Context(), book.ID, "epub", "test-book.epub", 17, nil, filePath)
	require.NoError(t, err, "create book file")

	// Verify initial download count is 0.
	require.Equal(t, int64(0), bf.DownloadCount)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/"+bf.ID+"/download", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "fake epub content", w.Body.String())
	require.Contains(t, w.Header().Get("Content-Disposition"), "test-book.epub")
	require.Equal(t, "application/epub+zip", w.Header().Get("Content-Type"))

	// Verify download count was incremented.
	updated, err := h.DB.GetBookFile(t.Context(), bf.ID)
	require.NoError(t, err, "get book file after download")
	require.Equal(t, int64(1), updated.DownloadCount)
}

func TestDownloadBookFile_UnknownSubResource(t *testing.T) {
	h, userID := setupBookFileHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/book-files/some-id/unknown", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleBookFile(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}
