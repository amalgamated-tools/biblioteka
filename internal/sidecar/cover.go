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

	path := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write cover.jpg: %w", err)
	}

	return nil
}
