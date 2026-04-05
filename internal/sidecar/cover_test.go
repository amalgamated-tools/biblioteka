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
	if filename != "cover.jpg" {
		t.Errorf("filename = %q, want %q", filename, "cover.jpg")
	}
	if mimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/jpeg")
	}

	written, err := os.ReadFile(filepath.Join(dir, "cover.jpg"))
	require.NoError(t, err, "read cover.jpg")

	if len(written) != len(imageData) {
		t.Errorf("cover.jpg size = %d, want %d", len(written), len(imageData))
	}
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
	if filename != "cover.png" {
		t.Errorf("filename = %q, want %q", filename, "cover.png")
	}
	if mimeType != "image/png" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/png")
	}

	if _, err := os.Stat(filepath.Join(dir, "cover.png")); err != nil {
		t.Errorf("cover.png not found: %v", err)
	}
}

func TestWriteCover_WebPDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x52, 0x49, 0x46, 0x46}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/webp;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	if filename != "cover.webp" {
		t.Errorf("filename = %q, want %q", filename, "cover.webp")
	}
	if mimeType != "image/webp" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/webp")
	}
}

func TestWriteCover_AVIFDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x00, 0x00, 0x00, 0x1C}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/avif;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL, "")
	require.NoError(t, err, "WriteCover")
	if filename != "cover.avif" {
		t.Errorf("filename = %q, want %q", filename, "cover.avif")
	}
	if mimeType != "image/avif" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/avif")
	}
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
	if _, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, ""); err != nil {
		require.NoError(t, err, "WriteCover")
	}

	written, err := os.ReadFile(coverPath)
	require.NoError(t, err, "read cover.jpg")
	if string(written) == "old" {
		t.Error("cover.jpg was not overwritten")
	}
}

func TestWriteCover_RemovesStaleFormats(t *testing.T) {
	dir := t.TempDir()

	for _, ext := range []string{".jpg", ".png"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "cover"+ext), []byte("old"), 0o644), "setup %s", ext)
	}

	imageData := []byte{0x52, 0x49, 0x46, 0x46}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	if _, _, err := WriteCover(dir, "data:image/webp;base64,"+encoded, ""); err != nil {
		require.NoError(t, err, "WriteCover")
	}

	for _, ext := range []string{".jpg", ".png"} {
		if _, err := os.Stat(filepath.Join(dir, "cover"+ext)); !os.IsNotExist(err) {
			t.Errorf("cover%s should have been removed, err=%v", ext, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "cover.webp")); err != nil {
		t.Errorf("cover.webp not found: %v", err)
	}
}

func TestWriteCover_CustomBaseName(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	filename, _, err := WriteCover(dir, dataURL, "Alice's Adventures in Wonderland by Lewis Carroll")
	require.NoError(t, err, "WriteCover")
	expected := "Alice's Adventures in Wonderland by Lewis Carroll.jpg"
	if filename != expected {
		t.Errorf("filename = %q, want %q", filename, expected)
	}
	if _, err := os.Stat(filepath.Join(dir, expected)); err != nil {
		t.Errorf("expected file not found: %v", err)
	}
	// Default "cover.jpg" should NOT exist.
	if _, err := os.Stat(filepath.Join(dir, "cover.jpg")); !os.IsNotExist(err) {
		t.Errorf("cover.jpg should not exist when using custom baseName")
	}
}

func TestWriteCover_CustomBaseName_RemovesStaleFormats(t *testing.T) {
	dir := t.TempDir()
	stem := "My Book"

	// Pre-create stale files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, stem+".png"), []byte("old"), 0o644), "setup")

	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	if _, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, stem); err != nil {
		require.NoError(t, err, "WriteCover")
	}

	if _, err := os.Stat(filepath.Join(dir, stem+".png")); !os.IsNotExist(err) {
		t.Errorf("%s.png should have been removed", stem)
	}
	if _, err := os.Stat(filepath.Join(dir, stem+".jpg")); err != nil {
		t.Errorf("%s.jpg not found: %v", stem, err)
	}
}

func TestWriteCover_InvalidBaseName(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)

	_, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded, "../escape")
	require.Error(t, err, "expected error for invalid base name")
}
