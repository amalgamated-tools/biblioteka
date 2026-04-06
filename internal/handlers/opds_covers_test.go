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

	"github.com/stretchr/testify/require"
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
		require.Equal(t, tt.want, got)
	}
}

func TestCoverImageInFeed(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	coverURL := "https://example.com/cover.png"
	_, err := h.DB.CreateBook(ctx, "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)
	imgLink := findLink(feed.Entries[0].Links, opdspkg.RelImage)
	require.NotNil(t, imgLink)
	require.Equal(t, "image/png", imgLink.Type)
}

func TestCoverImageInFeed_DataURLRewritten(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Len(t, feed.Entries, 1)
	imgLink := findLink(feed.Entries[0].Links, opdspkg.RelImage)
	require.NotNil(t, imgLink)
	wantHref := "http://example.com/opds/covers/" + book.ID
	require.Equal(t, wantHref, imgLink.Href)
	require.False(t, strings.HasPrefix(imgLink.Href, "data:"))
}

func TestServeCover_DataURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	pngBytes := testutils.TinyPNG()
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	book, err := h.DB.CreateBook(ctx, "Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &dataURL)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.True(t, strings.HasPrefix(ct, "image/"), "content-type = %q, want image/*", ct)
	require.True(t, bytes.Equal(w.Body.Bytes(), pngBytes))
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
			require.NoError(t, err, "create book")

			r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
			w := httptest.NewRecorder()
			h.HandleOPDS(w, r)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestServeCover_ExternalURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	coverURL := "https://example.com/cover.jpg"
	book, err := h.DB.CreateBook(ctx, "External Cover", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &coverURL)
	require.NoError(t, err, "create book")

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	require.Equal(t, http.StatusTemporaryRedirect, w.Code)
	location := w.Header().Get("Location")
	require.Equal(t, coverURL, location)
}

// TestServeCover_UnsafeExternalURL verifies that non-https URLs are rejected
// with 404 to prevent open redirect attacks.
func TestServeCover_UnsafeExternalURL(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	for _, unsafeURL := range []string{
		"http://evil.com/cover.jpg",
		"//evil.com/cover.jpg",
		"javascript:alert(1)",
		"ftp://example.com/cover.jpg",
	} {
		unsafeURL := unsafeURL
		t.Run(unsafeURL, func(t *testing.T) {
			book, err := h.DB.CreateBook(ctx, "Unsafe Cover Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &unsafeURL)
			require.NoError(t, err, "create book")

			r := httptest.NewRequest(http.MethodGet, "/opds/covers/"+book.ID, nil)
			w := httptest.NewRecorder()
			h.HandleOPDS(w, r)

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestServeCover_DBError(t *testing.T) {
	h := setupOPDSHandler(t)
	require.NoError(t, h.DB.Close(), "close db")

	r := httptest.NewRequest(http.MethodGet, "/opds/covers/someid", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	// serveCover intentionally returns 404 for all DB errors (not 500) to avoid
	// leaking internal state to OPDS clients — a missing cover is indistinguishable
	// from a DB failure from the client's perspective.
	require.Equal(t, http.StatusNotFound, w.Code)
}
