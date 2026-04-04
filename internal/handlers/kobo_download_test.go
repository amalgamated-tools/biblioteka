package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHandleDownload_CaseInsensitiveFormat verifies that format matching is
// case-insensitive (e.g., "EPUB" matches an epub file).
func TestHandleDownload_CaseInsensitiveFormat(t *testing.T) {
	t.Parallel()

	d := newTestDB(t)
	h := &KoboHandler{DB: d}

	dir := t.TempDir()
	bookFile := filepath.Join(dir, "sample.epub")
	if err := os.WriteFile(bookFile, []byte("epub content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	book, _, err := d.CreateBookWithFile(
		context.Background(),
		"Case Format Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "sample.epub", int64(len("epub content")), nil, bookFile,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

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
	if err := os.WriteFile(bookFile, []byte("epub data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	book, _, err := d.CreateBookWithFile(
		context.Background(),
		"CD Header Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "my-book.epub", int64(len("epub data")), nil, bookFile,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
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
	if err := os.WriteFile(bookFile, fileContent, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	book, _, err := d.CreateBookWithFile(
		context.Background(),
		"Content Verify Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub", "content.epub", int64(len(fileContent)), nil, bookFile,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/download/"+book.ID+"/epub", nil)
	w := httptest.NewRecorder()
	h.HandleDownload(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
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
