package sidecar

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteCover_ValidDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	require.Equal(t, "cover.jpg", filename, "filename")
	require.Equal(t, "image/jpeg", mimeType, "mimeType")

	written, err := os.ReadFile(filepath.Join(dir, "cover.jpg"))
	require.NoError(t, err, "read cover.jpg")

	require.Len(t, written, len(imageData), "cover.jpg size")
	for i := range imageData {
		require.Equal(t, imageData[i], written[i])
	}
}

func TestWriteCover_PNGDataURL_WritesAsPNG(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x89, 0x50, 0x4E, 0x47}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/png;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	require.Equal(t, "cover.png", filename, "filename")
	require.Equal(t, "image/png", mimeType, "mimeType")

	_, err = os.Stat(filepath.Join(dir, "cover.png"))
	require.NoError(t, err, "cover.png not found")
}

func TestWriteCover_WebPDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x52, 0x49, 0x46, 0x46}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/webp;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	require.Equal(t, "cover.webp", filename, "filename")
	require.Equal(t, "image/webp", mimeType, "mimeType")
}

func TestWriteCover_AVIFDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x00, 0x00, 0x00, 0x1C}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/avif;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	require.Equal(t, "cover.avif", filename, "filename")
	require.Equal(t, "image/avif", mimeType, "mimeType")
}

func TestWriteCover_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x47, 0x49, 0x46, 0x38}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/gif;base64," + encoded

	_, _, err := WriteCover(dir, dataURL, "")
	require.Error(t, err, "expected error for unsupported image format")
}

func TestWriteCover_InvalidDataURL(t *testing.T) {
	dir := t.TempDir()
	_, _, err := WriteCover(dir, "https://example.com/image.jpg", "")
	require.Error(t, err, "expected error for non-data URL")
}

func TestWriteCover_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")

	// Write an initial file.
	require.NoError(t, os.WriteFile(coverPath, []byte("old"), 0o644), "setup")

	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	_, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, "")
	require.NoError(t, err, "WriteCover")

	written, err := os.ReadFile(coverPath)
	require.NoError(t, err, "read cover.jpg")
	require.NotEqual(t, "old", string(written), "cover.jpg was not overwritten")
}

func TestWriteCover_RemovesStaleFormats(t *testing.T) {
	dir := t.TempDir()

	for _, ext := range []string{".jpg", ".png"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover"+ext), []byte("old"), 0o644), "setup %s", ext)
	}

	imageData := []byte{0x52, 0x49, 0x46, 0x46}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	_, _, err := WriteCover(dir, "data:image/webp;base64,"+encoded, "")
	require.NoError(t, err, "WriteCover")

	for _, ext := range []string{".jpg", ".png"} {
		_, err = os.Stat(filepath.Join(dir, "cover"+ext))
		require.True(t, os.IsNotExist(err), "cover%s should have been removed", ext)
	}
	_, err = os.Stat(filepath.Join(dir, "cover.webp"))
	require.NoError(t, err, "cover.webp not found")
}

func TestWriteCover_CustomBaseName(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	filename, _, err := WriteCover(dir, dataURL, "Alice's Adventures in Wonderland by Lewis Carroll")
	require.NoError(t, err, "WriteCover")
	expected := "Alice's Adventures in Wonderland by Lewis Carroll.jpg"
	require.Equal(t, expected, filename, "filename")
	_, err = os.Stat(filepath.Join(dir, expected))
	require.NoError(t, err, "expected file not found")
	// Default "cover.jpg" should NOT exist.
	_, err = os.Stat(filepath.Join(dir, "cover.jpg"))
	require.True(t, os.IsNotExist(err), "cover.jpg should not exist when using custom baseName")
}

func TestWriteCover_CustomBaseName_RemovesStaleFormats(t *testing.T) {
	dir := t.TempDir()
	stem := "My Book"

	// Pre-create stale files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, stem+".png"), []byte("old"), 0o644), "setup")

	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	_, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, stem)
	require.NoError(t, err, "WriteCover")

	_, err = os.Stat(filepath.Join(dir, stem+".png"))
	require.True(t, os.IsNotExist(err), "%s.png should have been removed", stem)
	_, err = os.Stat(filepath.Join(dir, stem+".jpg"))
	require.NoError(t, err, "%s.jpg not found", stem)
}

func TestWriteCover_InvalidBaseName(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)

	_, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, "../escape")
	require.Error(t, err, "expected error for invalid base name")
}
