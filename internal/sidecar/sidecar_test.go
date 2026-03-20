package sidecar

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
)

func TestWriteSidecarFiles_BothFiles(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(imageData)

	meta := &metadata.BookMetadata{
		CoverImageURL:   "data:image/jpeg;base64," + encoded,
		Description:     "A test book",
		Language:        "en",
		PublicationDate: "2024-01-01",
		Publisher:       "Test Publisher",
		ISBN:            "1234567890",
	}

	WriteSidecarFiles(context.Background(), dir, meta, "Test Book", "Test Author")

	if _, err := os.Stat(filepath.Join(dir, "cover.jpg")); err != nil {
		t.Errorf("cover.jpg not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "metadata.opf")); err != nil {
		t.Errorf("metadata.opf not found: %v", err)
	}
}

func TestWriteSidecarFiles_NoCover(t *testing.T) {
	dir := t.TempDir()
	meta := &metadata.BookMetadata{
		Description: "No cover book",
		Language:    "en",
	}

	WriteSidecarFiles(context.Background(), dir, meta, "No Cover Book", "Author")

	for _, ext := range []string{".jpg", ".png", ".webp", ".avif"} {
		name := "cover" + ext
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should not exist when no cover data URL", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "metadata.opf")); err != nil {
		t.Errorf("metadata.opf not found: %v", err)
	}
}

func TestWriteSidecarFiles_NilMetadata(t *testing.T) {
	dir := t.TempDir()

	WriteSidecarFiles(context.Background(), dir, nil, "Title Only", "")

	for _, ext := range []string{".jpg", ".png", ".webp", ".avif"} {
		name := "cover" + ext
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should not exist with nil metadata", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "metadata.opf")); err != nil {
		t.Errorf("metadata.opf not found: %v", err)
	}
}
