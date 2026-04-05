package sidecar

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/stretchr/testify/require"
)

func TestWriteSidecarFiles_BothFiles(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Test Book.epub")
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(imageData)

	meta := &exif.ExifToolOutput{
		CoverImageURL:   "data:image/jpeg;base64," + encoded,
		Description:     "A test book",
		Language:        "en",
		PublicationDate: "2024-01-01",
		Publisher:       "Test Publisher",
		ISBN10:          "1234567890",
	}

	WriteSidecarFiles(t.Context(), bookPath, meta, "Test Book", "Test Author", db.LibraryOrganizationBookPerFolder)

	_, err := os.Stat(filepath.Join(dir, "cover.jpg"))
	require.NoError(t, err, "cover.jpg not found")
	_, err = os.Stat(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "metadata.opf not found")
}

func TestWriteSidecarFiles_NoCover(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "No Cover Book.epub")
	meta := &exif.ExifToolOutput{
		Description: "No cover book",
		Language:    "en",
	}

	WriteSidecarFiles(t.Context(), bookPath, meta, "No Cover Book", "Author", db.LibraryOrganizationBookPerFolder)

	for _, ext := range []string{".jpg", ".png", ".webp", ".avif"} {
		name := "cover" + ext
		_, err := os.Stat(filepath.Join(dir, name))
		require.True(t, os.IsNotExist(err), "%s should not exist when no cover data URL", name)
	}
	_, err := os.Stat(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "metadata.opf not found")
}

func TestWriteSidecarFiles_NilMetadata(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Title Only.epub")

	WriteSidecarFiles(t.Context(), bookPath, nil, "Title Only", "", db.LibraryOrganizationBookPerFolder)

	for _, ext := range []string{".jpg", ".png", ".webp", ".avif"} {
		name := "cover" + ext
		_, err := os.Stat(filepath.Join(dir, name))
		require.True(t, os.IsNotExist(err), "%s should not exist with nil metadata", name)
	}
	_, err := os.Stat(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "metadata.opf not found")
}

func TestWriteSidecarFiles_BookPerFileUsesBookFilename(t *testing.T) {
	dir := t.TempDir()
	bookPath := filepath.Join(dir, "Alice's Adventures in Wonderland by Lewis Carroll.epub")
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(imageData)

	meta := &exif.ExifToolOutput{
		CoverImageURL: "data:image/jpeg;base64," + encoded,
		Language:      "en",
	}

	baseName := "Alice's Adventures in Wonderland by Lewis Carroll"
	WriteSidecarFiles(t.Context(), bookPath, meta, "Alice's Adventures in Wonderland", "Lewis Carroll", db.LibraryOrganizationBookPerFile)

	_, err := os.Stat(filepath.Join(dir, baseName+".jpg"))
	require.NoError(t, err, "expected %s.jpg", baseName)
	_, err = os.Stat(filepath.Join(dir, baseName+".opf"))
	require.NoError(t, err, "expected %s.opf", baseName)
	// Default names should NOT exist.
	_, err = os.Stat(filepath.Join(dir, "cover.jpg"))
	require.True(t, os.IsNotExist(err), "cover.jpg should not exist with custom baseName")
	_, err = os.Stat(filepath.Join(dir, "metadata.opf"))
	require.True(t, os.IsNotExist(err), "metadata.opf should not exist with custom baseName")
}
