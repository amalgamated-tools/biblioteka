package handlers

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSafeCoverRedirectURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"https is allowed", "https://example.com/cover.jpg", true},
		{"https with path and query", "https://cdn.example.com/img.png?v=1", true},
		{"http is rejected", "http://example.com/cover.jpg", false},
		{"protocol-relative is rejected", "//example.com/cover.jpg", false},
		{"javascript is rejected", "javascript:alert(1)", false},
		{"data URL is rejected", "data:image/png;base64,abc", false},
		{"empty string is rejected", "", false},
		{"relative path is rejected", "/cover.jpg", false},
		{"ftp is rejected", "ftp://example.com/cover.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isSafeCoverRedirectURL(tt.url))
		})
	}
}

func TestDataURLMIMEType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMIME string
		wantOK   bool
	}{
		{
			name:     "image/png data URL",
			input:    "data:image/png;base64,abc123",
			wantMIME: "image/png",
			wantOK:   true,
		},
		{
			name:     "image/webp data URL",
			input:    "data:image/webp;base64,abc123",
			wantMIME: "image/webp",
			wantOK:   true,
		},
		{
			name:     "empty mime type defaults to text/plain",
			input:    "data:;base64,abc123",
			wantMIME: "text/plain;charset=US-ASCII",
			wantOK:   true,
		},
		{
			name:     "mime with semicolon params strips after semicolon",
			input:    "data:image/png;base64,xyz",
			wantMIME: "image/png",
			wantOK:   true,
		},
		{
			name:     "no comma is not a data URL",
			input:    "data:image/png;base64",
			wantMIME: "",
			wantOK:   false,
		},
		{
			name:     "not a data URL (http URL)",
			input:    "https://example.com/image.png",
			wantMIME: "",
			wantOK:   false,
		},
		{
			name:     "empty string",
			input:    "",
			wantMIME: "",
			wantOK:   false,
		},
		{
			name:     "application/octet-stream",
			input:    "data:application/octet-stream;base64,abc",
			wantMIME: "application/octet-stream",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMIME, gotOK := dataURLMIMEType(tt.input)
			require.Equal(t, tt.wantOK, gotOK)
			require.Equal(t, tt.wantMIME, gotMIME)
		})
	}
}

func TestCoverMIMETypeExtended(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "data URL with image/png",
			input: "data:image/png;base64,abc",
			want:  "image/png",
		},
		{
			name:  "data URL with image/webp",
			input: "data:image/webp;base64,abc",
			want:  "image/webp",
		},
		{
			name:  "data URL with non-image mime falls through to extension default",
			input: "data:application/json;base64,abc",
			want:  "image/jpeg",
		},
		{
			name:  ".png extension",
			input: "/path/to/cover.png",
			want:  "image/png",
		},
		{
			name:  ".webp extension",
			input: "/path/to/cover.webp",
			want:  "image/webp",
		},
		{
			name:  ".avif extension",
			input: "/path/to/cover.avif",
			want:  "image/avif",
		},
		{
			name:  ".gif extension",
			input: "/path/to/cover.gif",
			want:  "image/gif",
		},
		{
			name:  ".svg extension",
			input: "/path/to/cover.svg",
			want:  "image/svg+xml",
		},
		{
			name:  ".jpg extension",
			input: "/path/to/cover.jpg",
			want:  "image/jpeg",
		},
		{
			name:  ".jpeg extension",
			input: "/path/to/cover.jpeg",
			want:  "image/jpeg",
		},
		{
			name:  "unknown extension defaults to image/jpeg",
			input: "/path/to/cover.bmp",
			want:  "image/jpeg",
		},
		{
			name:  "no extension defaults to image/jpeg",
			input: "/path/to/cover",
			want:  "image/jpeg",
		},
		{
			name:  "uppercase extension is case-insensitive",
			input: "/path/to/cover.PNG",
			want:  "image/png",
		},
		{
			name:  "mixed case extension",
			input: "/path/to/cover.WebP",
			want:  "image/webp",
		},
		{
			name:  "URL with query string strips query before ext lookup",
			input: "https://example.com/cover.png?size=200",
			want:  "image/png",
		},
		{
			name:  "URL with path and query for webp",
			input: "https://cdn.example.com/images/cover.webp?v=2",
			want:  "image/webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coverMIMEType(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeDataURL(t *testing.T) {
	t.Run("valid base64 PNG data URL", func(t *testing.T) {
		payload := []byte("hello world image bytes")
		encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(payload)

		mime, got, err := decodeDataURL(encoded)
		require.NoError(t, err)
		require.Equal(t, "image/png", mime)
		require.Equal(t, string(payload), string(got))
	})

	t.Run("not a data URL returns ErrNotDataURL", func(t *testing.T) {
		_, _, err := decodeDataURL("https://example.com/image.png")
		require.Error(t, err, "expected error for non-data URL, got nil")
		require.ErrorIs(t, err, errNotDataURL)
	})

	t.Run("empty string returns ErrNotDataURL", func(t *testing.T) {
		_, _, err := decodeDataURL("")
		require.Error(t, err, "expected error for empty string, got nil")
		require.ErrorIs(t, err, errNotDataURL)
	})

	t.Run("valid base64 JPEG data URL", func(t *testing.T) {
		payload := []byte{0xFF, 0xD8, 0xFF} // fake JPEG header
		encoded := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(payload)

		mime, got, err := decodeDataURL(encoded)
		require.NoError(t, err)
		require.Equal(t, "image/jpeg", mime)
		require.Len(t, got, len(payload))
	})

	t.Run("invalid base64 payload returns error", func(t *testing.T) {
		_, _, err := decodeDataURL("data:image/png;base64,!!!invalid!!!")
		require.Error(t, err, "expected error for invalid base64, got nil")
	})
}
