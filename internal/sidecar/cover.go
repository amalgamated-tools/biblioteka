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

	// Write atomically: write to a temp file, rename into place, then clean stale formats.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", filename, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", "", fmt.Errorf("rename %s: %w", filename, err)
	}

	// Remove previously written cover files of other formats only after the new file is in place.
	for _, oldExt := range []string{".jpg", ".png", ".webp", ".avif"} {
		oldPath := filepath.Join(dir, "cover"+oldExt)
		if oldPath != path {
			_ = os.Remove(oldPath) // best-effort cleanup
		}
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
