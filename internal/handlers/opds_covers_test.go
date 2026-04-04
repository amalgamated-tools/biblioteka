package handlers

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
)

// --- Cover image MIME type ---

func TestCoverMIMEType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/cover.jpg", "image/jpeg"},
		{"https://example.com/cover.jpeg", "image/jpeg"},
		{"https://example.com/cover.png", "image/png"},
		{"https://example.com/cover.PNG", "image/png"},
		{"https://example.com/cover.webp", "image/webp"},
		{"https://example.com/cover.gif", "image/gif"},
		{"https://example.com/cover.svg", "image/svg+xml"},
		{"https://example.com/cover.avif", "image/avif"},
		{"data:image/png;base64,AAAA", "image/png"},
		{"https://example.com/cover", "image/jpeg"}, // no extension defaults to jpeg
	}

	for _, tt := range tests {
		got := coverMIMEType(tt.url)
		if got != tt.want {
			t.Errorf("coverMIMEType(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestCoverImageInFeed(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	coverURL := "https://example.com/cover.png"
	h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	imgLink := findLink(feed.Entries[0].Links, opdspkg.RelImage)
	if imgLink == nil {
		t.Fatal("missing image link")
	}
	if imgLink.Type != "image/png" {
		t.Errorf("image type = %q, want %q", imgLink.Type, "image/png")
	}
}

func TestCoverImageInFeed_DataURLRewritten(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(feed.Entries))
	}
	imgLink := findLink(feed.Entries[0].Links, opdspkg.RelImage)
	if imgLink == nil {
		t.Fatal("missing image link")
	}
	wantHref := "http://example.com/opds/covers/" + book.ID
	if imgLink.Href != wantHref {
		t.Errorf("image href = %q, want %q", imgLink.Href, wantHref)
	}
	if strings.HasPrefix(imgLink.Href, "data:") {
		t.Error("image href should not be a data URL")
	}
}

func TestServeCover_DataURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		t.Errorf("content-type = %q, want image/*", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), pngBytes) {
		t.Errorf("body length = %d, want %d", w.Body.Len(), len(pngBytes))
	}
}

// --- serveCover missing paths ---

func TestServeCover_MissingCover(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cover *string
	}{
		{"nil cover", nil},
		{"empty cover", ptr("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupOPDSHandler(t)
			ctx := t.Context()

			book, err := h.DB.CreateBook(ctx, "Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, tc.cover)
			if err != nil {
				t.Fatalf("create book: %v", err)
			}

			r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
			w := httptest.NewRecorder()
			h.HandleOPDS(w, r)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestServeCover_ExternalURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	coverURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(ctx, "External Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)
	if err != nil {
		t.Fatalf("create book: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}
	location := w.Header().Get("Location")
	if location != coverURL {
		t.Errorf("Location = %q, want %q", location, coverURL)
	}
}

func TestServeCover_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	if err := h.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/someid", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	// serveCover intentionally returns 404 for all DB errors (not 500) to avoid
	// leaking internal state to OPDS clients — a missing cover is indistinguishable
	// from a DB failure from the client's perspective.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
