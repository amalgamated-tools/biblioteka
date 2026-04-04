package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
)

// requireExifToolExtractor creates a metadata.Extractor and skips the test if
// exiftool is not installed. Cover extraction requires exiftool, so any test that
// needs end-to-end cover import must call this helper.
func requireExifToolExtractor(t *testing.T) *metadata.Extractor {
	t.Helper()
	// NewExtractor always returns nil error; skip check and use probe instead.
	ext, _ := metadata.NewExtractor(t.Context())
	// Probe availability: ExtractMetadata returns ErrExifToolUnavailable immediately
	// (no file I/O) when the underlying exiftool process could not be started.
	_, probeErr := ext.ExtractMetadata(t.Context(), "/dev/null")
	if errors.Is(probeErr, metadata.ErrExifToolUnavailable) {
		ext.Close(t.Context())
		t.Skip("exiftool not available, skipping EPUB3 cover import test")
	}
	t.Cleanup(func() { ext.Close(context.Background()) })
	return ext
}

// TestEPUB3CoverImport_EndToEnd verifies the complete EPUB3 cover import flow:
//  1. A fixture EPUB3 file with a known cover image is processed via ProcessBookFile.
//  2. The cover is stored as a base64 data URL on the imported book record.
//  3. The decoded cover bytes match the original image.
//  4. The cover is served correctly by KoboHandler.HandleCoverImage.
func TestEPUB3CoverImport_EndToEnd(t *testing.T) {
	ext := requireExifToolExtractor(t)

	d := newTestDB(t)
	koboH := &KoboHandler{DB: d}

	// Create a fixture EPUB3 file with a known 1×1 PNG cover image using the
	// EPUB3-specific properties="cover-image" manifest attribute.
	coverBytes := testutils.TinyPNG()
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3-with-cover.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB3 Cover Book", "Test Author",
		"urn:isbn:9780000099999", testutils.EPUBOptions{
			Version:        "3.0",
			EPUB3Cover:     true,
			CoverImageData: coverBytes,
			CoverImageHref: "images/cover.png",
			CoverMediaType: "image/png",
		})

	// Import the EPUB3 file; this triggers metadata extraction (including cover).
	epubInfo, err := os.Stat(epubPath)
	if err != nil {
		t.Fatalf("stat epub: %v", err)
	}
	if err := jobs.ProcessBookFile(t.Context(), d, ext, jobs.ProcessFilePayload{
		Path:     epubPath,
		FileName: "epub3-with-cover.epub",
		FileType: "epub",
		FileSize: epubInfo.Size(),
	}); err != nil {
		t.Fatalf("ProcessBookFile: %v", err)
	}

	// Verify the cover is associated with the imported book.
	books, err := d.ListBooks(t.Context())
	if err != nil {
		t.Fatalf("list books: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	book := books[0]
	if book.CoverImageURL == nil {
		t.Fatal("expected cover image URL to be set, got nil")
	}
	if !strings.HasPrefix(*book.CoverImageURL, "data:image/png;base64,") {
		t.Fatalf("expected PNG data URL cover, got %q", *book.CoverImageURL)
	}

	// Verify the decoded cover bytes match the original image.
	b64 := strings.TrimPrefix(*book.CoverImageURL, "data:image/png;base64,")
	gotCover, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode cover base64: %v", err)
	}
	if !bytes.Equal(gotCover, coverBytes) {
		t.Errorf("cover bytes mismatch: got %d bytes, want %d bytes", len(gotCover), len(coverBytes))
	}

	// Verify the cover is served and accessible via HandleCoverImage.
	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil).WithContext(t.Context())
	w := httptest.NewRecorder()
	koboH.HandleCoverImage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleCoverImage status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/png") {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), coverBytes) {
		t.Errorf("served cover bytes mismatch: got %d bytes, want %d bytes", len(w.Body.Bytes()), len(coverBytes))
	}
}
