package sidecar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/coverutil"
)

// WriteCover decodes a base64 data URL and writes the image as cover.jpg in dir.
func WriteCover(dir string, coverDataURL string) error {
	mimeType, data, err := coverutil.DecodeDataURL(coverDataURL)
	if err != nil {
		return fmt.Errorf("decode cover data URL: %w", err)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return fmt.Errorf("unsupported cover MIME type: %s", mimeType)
	}

	ext := extensionForMIME(mimeType) // e.g. ".jpg", ".png", ".webp"
	filename := "cover" + ext
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}

	return nil
}

// extensionForMIME returns the file extension for a given MIME type.
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
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}
