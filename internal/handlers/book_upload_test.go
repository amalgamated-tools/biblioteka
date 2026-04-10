package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/stretchr/testify/require"
)

// --- helpers ---------------------------------------------------------------

// makeUploadRequest builds a multipart POST request for HandleUpload.
// fields is a map of additional form fields (e.g. "library_id", "title").
func makeUploadRequest(
	t *testing.T,
	url string,
	filename string,
	fileContent []byte,
	fields map[string]string,
) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if filename != "" {
		fw, err := mw.CreateFormFile("file", filename)
		require.NoError(t, err, "create form file")
		_, err = fw.Write(fileContent)
		require.NoError(t, err, "write file content")
	}

	for key, value := range fields {
		require.NoError(t, mw.WriteField(key, value), "write field %q", key)
	}

	require.NoError(t, mw.Close())

	r := httptest.NewRequest(http.MethodPost, url, &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

// setupUploadHandler creates a BookHandler backed by an in-memory test DB, a
// test library rooted at a temp directory, and a mockEnqueuer. It returns the
// handler, the user ID, and the library ID.
func setupUploadHandler(t *testing.T) (h *BookHandler, userID, libraryID string) {
	t.Helper()

	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Upload User", "upload@example.com", "password1")
	require.NoError(t, err, "create user")

	libDir := t.TempDir()
	pathsJSON, err := json.Marshal([]string{libDir})
	require.NoError(t, err, "marshal library paths")
	lib, err := d.CreateLibrary(t.Context(), "Upload Library", string(pathsJSON), db.LibraryOrganizationNone, false)
	require.NoError(t, err, "create library")

	h = &BookHandler{
		DB:       d,
		Enqueuer: &mockEnqueuer{},
	}

	return h, user.ID, lib.ID
}

// --- tests -----------------------------------------------------------------

func TestHandleUpload_Success(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "test.epub", []byte("epub content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	var resp uploadAcceptedResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "unmarshal response")
	require.Equal(t, "test.epub", resp.FileName)
	require.Equal(t, "epub", resp.FileType)
	require.Equal(t, libraryID, resp.LibraryID)
	require.NotEmpty(t, resp.Message)
}

func TestHandleUpload_EnqueuesProcessFileJob(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	r := makeUploadRequest(t, "/api/books/upload", "dune.epub", []byte("epub content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1, "expected exactly one enqueued job")
	require.Equal(t, jobs.JobProcessFile, enq.jobs[0].Name)

	var payload jobs.ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.Equal(t, "dune.epub", payload.FileName)
	require.Equal(t, "epub", payload.FileType)
	require.Equal(t, libraryID, payload.LibraryID)
	require.Equal(t, userID, payload.UserID)
	require.NotEmpty(t, payload.Path, "staging path should be set")
}

func TestHandleUpload_StagedFileExists(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	fileContent := []byte("epub file bytes")
	r := makeUploadRequest(t, "/api/books/upload", "book.epub", fileContent, map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)

	var payload jobs.ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))

	// The staged file should exist on disk at the path in the payload.
	_, statErr := os.Stat(payload.Path)
	require.NoError(t, statErr, "staged file should exist on disk")
	require.Equal(t, int64(len(fileContent)), payload.FileSize)
}

func TestHandleUpload_MetadataOverrides(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)
	enq := &mockEnqueuer{}
	h.Enqueuer = enq

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id":  libraryID,
		"title":       "My Custom Title",
		"author":      "Jane Doe",
		"description": "A great book",
		"isbn":        "9780000000001",
		"language":    "en",
		"publisher":   "Acme Press",
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	enq.mu.Lock()
	defer enq.mu.Unlock()
	require.Len(t, enq.jobs, 1)

	var payload jobs.ProcessFilePayload
	require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
	require.Equal(t, "My Custom Title", payload.OverrideTitle)
	require.Equal(t, "Jane Doe", payload.OverrideAuthor)
	require.Equal(t, "A great book", payload.OverrideDescription)
	require.Equal(t, "9780000000001", payload.OverrideISBN)
	require.Equal(t, "en", payload.OverrideLanguage)
	require.Equal(t, "Acme Press", payload.OverridePublisher)
}

func TestHandleUpload_WrongMethod(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/api/books/upload", nil)
	r = withUserID(r, userID)
	// Unused but keeps the test realistic.
	_ = libraryID
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleUpload_MissingFile(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	// Send a multipart form without a "file" field.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("library_id", libraryID))
	require.NoError(t, mw.Close())

	r := httptest.NewRequest(http.MethodPost, "/api/books/upload", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleUpload_UnsupportedFileType(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "document.txt", []byte("text content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "unsupported file type")
}

func TestHandleUpload_MissingLibraryID(t *testing.T) {
	h, userID, _ := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		// library_id intentionally omitted
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "library_id")
}

func TestHandleUpload_LibraryNotFound(t *testing.T) {
	h, userID, _ := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id": "nonexistent-library-id",
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleUpload_NoEnqueuer(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)
	h.Enqueuer = nil

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandleUpload_EnqueueFailureCleansStagedFile(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)
	enq := &mockEnqueuer{err: fmt.Errorf("redis unavailable")}
	h.Enqueuer = enq

	// Track staged files by intercepting the enqueue — since enqueueing fails
	// before the response is written, we instead check via the library's
	// .uploads directory after the request.
	// Use the same library dir from the main handler.
	libs, err := h.DB.ListLibraries(t.Context())
	require.NoError(t, err)
	require.Len(t, libs, 1)

	var libPaths []string
	require.NoError(t, json.Unmarshal([]byte(libs[0].Paths), &libPaths))
	stagingDir := filepath.Join(libPaths[0], uploadStagingDir)

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	// The staging directory may not have been created at all, or it was created
	// and the file was cleaned up. Either way, no .epub file should remain.
	entries, readErr := os.ReadDir(stagingDir)
	if readErr == nil {
		for _, e := range entries {
			require.False(t, strings.HasSuffix(e.Name(), ".epub"), "staged epub should be removed after enqueue failure")
		}
	}
}

func TestHandleUpload_InvalidISBN(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id": libraryID,
		"isbn":       "not-a-valid-isbn",
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Contains(t, resp.Error, "invalid isbn")
}

func TestHandleUpload_AllSupportedExtensions(t *testing.T) {
	extensions := []struct {
		filename string
		fileType string
	}{
		{"book.epub", "epub"},
		{"book.mobi", "mobi"},
		{"book.azw3", "azw3"},
		{"book.pdf", "pdf"},
	}

	for _, tc := range extensions {
		t.Run(tc.filename, func(t *testing.T) {
			h, userID, libraryID := setupUploadHandler(t)
			enq := &mockEnqueuer{}
			h.Enqueuer = enq

			r := makeUploadRequest(t, "/api/books/upload", tc.filename, []byte("content"), map[string]string{
				"library_id": libraryID,
			})
			r = withUserID(r, userID)
			w := httptest.NewRecorder()

			h.HandleUpload(w, r)

			require.Equal(t, http.StatusAccepted, w.Code, "filename=%s", tc.filename)

			enq.mu.Lock()
			defer enq.mu.Unlock()
			require.Len(t, enq.jobs, 1)

			var payload jobs.ProcessFilePayload
			require.NoError(t, json.Unmarshal(enq.jobs[0].Payload, &payload))
			require.Equal(t, tc.fileType, payload.FileType)
		})
	}
}

func TestHandleUpload_AuditLog(t *testing.T) {
	h, userID, libraryID := setupUploadHandler(t)

	r := makeUploadRequest(t, "/api/books/upload", "book.epub", []byte("content"), map[string]string{
		"library_id": libraryID,
	})
	r = withUserID(r, userID)
	w := httptest.NewRecorder()

	h.HandleUpload(w, r)

	require.Equal(t, http.StatusAccepted, w.Code)

	logs, _, err := h.DB.ListAuditLogs(t.Context(), 10, 0)
	require.NoError(t, err, "list audit logs")

	found := false
	for _, l := range logs {
		if l.Action == db.AuditActionBookUploaded {
			found = true
			require.NotNil(t, l.UserID)
			require.Equal(t, userID, *l.UserID)
			break
		}
	}
	require.True(t, found, "expected book.uploaded audit log entry")
}

// --- unit tests for package-level helpers ----------------------------------

func TestDetectUploadFileType(t *testing.T) {
	tests := []struct {
		filename string
		wantType string
		wantOK   bool
	}{
		{"book.epub", "epub", true},
		{"BOOK.EPUB", "epub", true},
		{"book.mobi", "mobi", true},
		{"book.azw3", "azw3", true},
		{"book.pdf", "pdf", true},
		{"book.txt", "", false},
		{"book.doc", "", false},
		{"noextension", "", false},
		{"book.epub.bak", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			gotType, gotOK := detectUploadFileType(tc.filename)
			require.Equal(t, tc.wantOK, gotOK)
			require.Equal(t, tc.wantType, gotType)
		})
	}
}

func TestParseFirstLibraryPath(t *testing.T) {
	t.Run("single path", func(t *testing.T) {
		p, err := parseFirstLibraryPath(`["/books/library"]`)
		require.NoError(t, err)
		require.Equal(t, "/books/library", p)
	})

	t.Run("multiple paths returns first", func(t *testing.T) {
		p, err := parseFirstLibraryPath(`["/books/a", "/books/b"]`)
		require.NoError(t, err)
		require.Equal(t, "/books/a", p)
	})

	t.Run("empty list errors", func(t *testing.T) {
		_, err := parseFirstLibraryPath(`[]`)
		require.Error(t, err)
	})

	t.Run("invalid JSON errors", func(t *testing.T) {
		_, err := parseFirstLibraryPath(`not-json`)
		require.Error(t, err)
	})
}

func TestSaveUploadedFile(t *testing.T) {
	t.Run("writes content and returns size", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.epub")
		content := []byte("hello world")

		n, err := saveUploadedFile(bytes.NewReader(content), dest)
		require.NoError(t, err)
		require.Equal(t, int64(len(content)), n)

		got, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, content, got)
	})

	t.Run("cleans up on write error", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.epub")

		// A reader that errors partway through.
		r := &errAfterNReader{data: []byte("partial"), errAfter: 3}
		_, err := saveUploadedFile(r, dest)
		require.Error(t, err)

		// File should have been cleaned up.
		_, statErr := os.Stat(dest)
		require.True(t, os.IsNotExist(statErr), "staged file should be removed after write error")
	})
}

// errAfterNReader returns data for the first errAfter bytes, then returns an error.
type errAfterNReader struct {
	data     []byte
	errAfter int
	pos      int
}

func (r *errAfterNReader) Read(p []byte) (int, error) {
	if r.pos >= r.errAfter {
		return 0, fmt.Errorf("simulated read error")
	}
	remaining := r.errAfter - r.pos
	src := r.data[r.pos:]
	if len(src) > remaining {
		src = src[:remaining]
	}
	n := copy(p, src)
	r.pos += n
	if r.pos >= r.errAfter {
		return n, fmt.Errorf("simulated read error")
	}
	return n, nil
}
