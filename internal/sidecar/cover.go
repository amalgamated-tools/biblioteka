package sidecar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/coverutil"
)

// WriteCover decodes a base64 data URL and writes the image in dir.
// The output filename is determined by the decoded MIME type (e.g. cover.png,
// cover.webp). It returns the filename and MIME type on success.
func WriteCover(dir string, coverDataURL string) (filename, mimeType string, err error) {
	mimeType, data, err := coverutil.DecodeDataURL(coverDataURL)
	if err != nil {
		return "", "", fmt.Errorf("decode cover data URL: %w", err)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return "", "", fmt.Errorf("unsupported cover MIME type: %s", mimeType)
	}

	ext := extensionForMIME(mimeType)
	if ext == "" {
		return "", "", fmt.Errorf("unsupported cover image format: %s", mimeType)
	}
	filename = "cover" + ext
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", filename, err)
	}

	return filename, mimeType, nil
}

// extensionForMIME returns the file extension for supported cover image MIME types.
func extensionForMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/avif":
		return ".avif"
	default:
		return ""
	}
}
