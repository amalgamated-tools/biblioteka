package exif

import (
	"testing"
)

// TestIsASIN verifies that isASIN correctly identifies alphanumeric strings.
func TestIsASIN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "all uppercase letters", input: "ABCDEFGHIJ", want: true},
		{name: "all digits", input: "0123456789", want: true},
		{name: "mixed alphanumeric", input: "B08FHBV4ZX", want: true},
		{name: "lowercase letters", input: "abcdefghij", want: true},
		{name: "empty string", input: "", want: true}, // vacuously true (no invalid chars)
		{name: "contains hyphen", input: "B08-HBV4ZX", want: false},
		{name: "contains space", input: "B08FHBV 4Z", want: false},
		{name: "contains dot", input: "B08FHBV.4Z", want: false},
		{name: "single valid char", input: "A", want: true},
		{name: "single invalid char", input: "-", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isASIN(tt.input)
			if got != tt.want {
				t.Errorf("isASIN(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsLikelyImage verifies that isLikelyImage recognizes images by MIME type
// and file extension.
func TestIsLikelyImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		href     string
		mimeType string
		want     bool
	}{
		// MIME type based detection
		{name: "image/jpeg MIME", href: "file.xyz", mimeType: "image/jpeg", want: true},
		{name: "image/png MIME", href: "file.xyz", mimeType: "image/png", want: true},
		{name: "image/ prefix", href: "file.xyz", mimeType: "image/webp", want: true},
		{name: "IMAGE/PNG uppercase MIME", href: "file.xyz", mimeType: "IMAGE/PNG", want: true},
		{name: "non-image MIME", href: "file.xhtml", mimeType: "application/xhtml+xml", want: false},
		// Extension-based detection (when MIME is empty or non-image)
		{name: ".jpg extension", href: "images/cover.jpg", mimeType: "", want: true},
		{name: ".jpeg extension", href: "images/cover.jpeg", mimeType: "", want: true},
		{name: ".png extension", href: "images/cover.png", mimeType: "", want: true},
		{name: ".gif extension", href: "images/icon.gif", mimeType: "", want: true},
		{name: ".webp extension", href: "images/cover.webp", mimeType: "", want: true},
		{name: ".avif extension", href: "images/cover.avif", mimeType: "", want: true},
		{name: ".svg extension", href: "images/logo.svg", mimeType: "", want: true},
		{name: ".PNG uppercase extension", href: "images/cover.PNG", mimeType: "", want: true},
		{name: ".xhtml extension not image", href: "text/ch1.xhtml", mimeType: "", want: false},
		{name: ".xml extension not image", href: "toc.xml", mimeType: "", want: false},
		{name: "no extension", href: "somefile", mimeType: "", want: false},
		{name: "empty href and MIME", href: "", mimeType: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isLikelyImage(tt.href, tt.mimeType)
			if got != tt.want {
				t.Errorf("isLikelyImage(%q, %q) = %v, want %v", tt.href, tt.mimeType, got, tt.want)
			}
		})
	}
}
