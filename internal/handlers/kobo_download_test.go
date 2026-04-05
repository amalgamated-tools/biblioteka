package handlers

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/kobo"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"

	"github.com/stretchr/testify/require"
)

func TestHandleCoverImage_DataURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	pngBytes := testutils.TinyPNG()
	cover := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(t.Context(), "Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &cover)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	got := w.Header().Get("Content-Type")
	require.Equal(t, "image/png", got)
	require.Equal(t, pngBytes, w.Body.Bytes())
}

// ---- HandleCoverImage edge cases ----

func TestHandleCoverImage_BookNotFound(t *testing.T) {
	h, _ := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/covers/nonexistent/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCoverImage_NoCover(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(t.Context(), "No Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCoverImage_ExternalURL(t *testing.T) {
	h, _ := setupKoboHandler(t)
	externalURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(t.Context(), "External Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &externalURL)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}
	if loc := w.Header().Get("Location"); loc != externalURL {
		t.Errorf("Location = %q, want %q", loc, externalURL)
	}
}

func TestHandleCoverImage_EmptyBookID(t *testing.T) {
	h, _ := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/covers/", nil)
	w := httptest.NewRecorder()

	h.HandleCoverImage(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ---- HandleDownload ----

func TestHandleDownload_MissingSegments(t *testing.T) {
	h, _ := setupKoboHandler(t)

	// Only one segment (no format)
	r := httptest.NewRequest(http.MethodGet, "/download/onlyone", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDownload_FormatNotFound(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(t.Context(), "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	// Book has no files at all, so format won't match.
	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDownload_FileNotFoundOnDisk(t *testing.T) {
	h, _ := setupKoboHandler(t)
	book, err := h.DB.CreateBook(t.Context(), "Test Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	// Register a file in the DB that doesn't exist on disk.
	_, err = h.DB.CreateBookFile(t.Context(), book.ID, "epub", "test.epub", 1024, nil, filepath.Join(t.TempDir(), "nonexistent-kobo-test-file.epub"))
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDownload_Success(t *testing.T) {
	h, _ := setupKoboHandler(t)

	// Write a temp file to serve.
	f, err := os.CreateTemp(t.TempDir(), "test-*.epub")
	require.NoError(t, err, "create temp file")
	content := []byte("fake epub content")
	if _, err := f.Write(content); err != nil {
		require.NoError(t, err, "write temp file")
	}
	require.NoError(t, f.Close(), "close temp file")

	book, err := h.DB.CreateBook(t.Context(), "Download Test", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")
	_, err = h.DB.CreateBookFile(t.Context(), book.ID, "epub", "test.epub", int64(len(content)), nil, f.Name())
	require.NoError(t, err, "create book file")

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()

	h.HandleDownload(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	if !bytes.Equal(w.Body.Bytes(), content) {
		t.Errorf("body = %q, want %q", w.Body.Bytes(), content)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "test.epub") {
		t.Errorf("Content-Disposition = %q, want filename=test.epub", cd)
	}
}

// ---- HandleDownload case-insensitive and edge cases ----

// TestHandleDownload_CaseInsensitiveFormat verifies that format matching is
// case-insensitive (e.g., "EPUB" matches an epub file).
func TestHandleDownload_CaseInsensitiveFormat(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	dir := t.TempDir()
	bookFile := filepath.Join(dir, "sample.epub")
	require.NoError(t, os.WriteFile(bookFile, []byte("epub content"), 0o644), "write file")

	book, _, err := d.CreateBookWithFile(
		t.Context(),
		"Case Format Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "sample.epub", int64(len("epub content")), nil, bookFile,
	)
	require.NoError(t, err, "create book")

	// Request with uppercase format.
	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/EPUB", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d for uppercase format", w.Code, http.StatusOK)
	}
}

// TestHandleDownload_ContentDispositionHeader verifies that the response
// includes a Content-Disposition header with the original file name.
func TestHandleDownload_ContentDispositionHeader(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	dir := t.TempDir()
	bookFile := filepath.Join(dir, "my-book.epub")
	require.NoError(t, os.WriteFile(bookFile, []byte("epub data"), 0o644), "write file")

	book, _, err := d.CreateBookWithFile(
		t.Context(),
		"CD Header Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "my-book.epub", int64(len("epub data")), nil, bookFile,
	)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "my-book.epub") {
		t.Errorf("Content-Disposition %q should contain filename my-book.epub", cd)
	}
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition %q should be attachment", cd)
	}
}

// TestHandleDownload_ReturnsFileContents verifies that the response body
// matches the file contents.
func TestHandleDownload_ReturnsFileContents(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	dir := t.TempDir()
	fileContent := []byte("epub file content here")
	bookFile := filepath.Join(dir, "content.epub")
	require.NoError(t, os.WriteFile(bookFile, fileContent, 0o644), "write file")

	book, _, err := d.CreateBookWithFile(
		t.Context(),
		"Content Verify Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "content.epub", int64(len(fileContent)), nil, bookFile,
	)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	if got := w.Body.Bytes(); string(got) != string(fileContent) {
		t.Errorf("body = %q, want %q", got, fileContent)
	}
}

// TestHandleDownload_EmptyFormat verifies that a URL with an empty format
// segment returns an error.
func TestHandleDownload_EmptyFormat(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	r := httptest.NewRequest(http.MethodGet, "/download/some-book-id/", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	// Should return bad request or not found for empty format.
	if w.Code == http.StatusOK {
		t.Errorf("status = 200, want non-200 for empty format")
	}
}

// ---- kobo.DownloadURLs helper ----

func TestKoboDownloadURLs_FiltersUnsupportedFormats(t *testing.T) {
	files := []db.BookFile{
		{ID: "1", FileType: "epub", FileName: "book.epub", FileSize: 100},
		{ID: "2", FileType: "txt", FileName: "book.txt", FileSize: 50},
		{ID: "3", FileType: "pdf", FileName: "book.pdf", FileSize: 200},
	}
	urls := kobo.DownloadURLs("http://localhost", "mytoken", "book-id", files)
	require.Len(t, urls, 2)
	formats := make(map[string]bool)
	for _, u := range urls {
		if u.Format == "" {
			t.Error("expected non-empty Format in download URL")
		}
		formats[u.Format] = true
		if u.URL == "" {
			t.Error("expected non-empty URL in download URL")
		}
	}
	if !formats["EPUB3"] {
		t.Error("expected EPUB3 format in download URLs")
	}
	if !formats["PDF"] {
		t.Error("expected PDF format in download URLs")
	}
}
