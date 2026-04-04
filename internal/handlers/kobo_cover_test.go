package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/testutils"
)

// TestHandleCoverImage_JPEGDataURL verifies that a JPEG data URL cover
// is served with a recognized image Content-Type.
func TestHandleCoverImage_JPEGDataURL(t *testing.T) {
	t.Parallel()

	h, _ := setupKoboHandler(t)

	// Create a PNG-encoded image wrapped as JPEG in the data URL prefix.
	// The server detects the actual MIME type from the image bytes,
	// so the response will be image/png regardless of the declared JPEG prefix.
	pngBytes := testutils.TinyPNG()
	jpegDataURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(
		context.Background(),
		"JPEG Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		&jpegDataURL,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()
	h.HandleCoverImage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		t.Errorf("Content-Type = %q, want image/* type", ct)
	}
}

// TestHandleCoverImage_PNGDataURL verifies that a PNG data URL cover is
// served with the correct Content-Type.
func TestHandleCoverImage_PNGDataURL(t *testing.T) {
	t.Parallel()

	h, _ := setupKoboHandler(t)

	pngBytes := testutils.TinyPNG()
	pngDataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(
		context.Background(),
		"PNG Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		&pngDataURL,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()
	h.HandleCoverImage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/png") {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
}

// TestHandleCoverImage_InvalidDataURL verifies that a malformed data URL
// results in a 400 or 500 error response.
func TestHandleCoverImage_InvalidDataURL(t *testing.T) {
	t.Parallel()

	h, _ := setupKoboHandler(t)

	// Malformed data URL: not valid base64.
	badURL := "data:image/png;base64,!notvalidbase64!"
	book, err := h.DB.CreateBook(
		context.Background(),
		"Bad Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		&badURL,
	)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/covers/"+book.ID+"/600/800/false/image.jpg", nil)
	w := httptest.NewRecorder()
	h.HandleCoverImage(w, r)

	// Should return an error status for invalid base64.
	if w.Code == http.StatusOK {
		t.Errorf("status = 200, want non-200 for invalid data URL")
	}
}

// TestHandleCoverImage_MissingPathSegments verifies that an incomplete path
// returns an error.
func TestHandleCoverImage_MissingPathSegments(t *testing.T) {
	t.Parallel()

	h, _ := setupKoboHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/covers/", nil)
	w := httptest.NewRecorder()
	h.HandleCoverImage(w, r)

	if w.Code == http.StatusOK {
		t.Errorf("status = 200, want non-200 for missing path segments")
	}
}
