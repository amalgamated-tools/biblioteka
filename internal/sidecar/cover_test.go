package sidecar

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCover_ValidDataURL(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/jpeg;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL)
	if err != nil {
		t.Fatalf("WriteCover: %v", err)
	}
	if filename != "cover.jpg" {
		t.Errorf("filename = %q, want %q", filename, "cover.jpg")
	}
	if mimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/jpeg")
	}

	written, err := os.ReadFile(filepath.Join(dir, "cover.jpg"))
	if err != nil {
		t.Fatalf("read cover.jpg: %v", err)
	}

	if len(written) != len(imageData) {
		t.Errorf("cover.jpg size = %d, want %d", len(written), len(imageData))
	}
	for i := range imageData {
		if written[i] != imageData[i] {
			t.Fatalf("cover.jpg byte %d = %x, want %x", i, written[i], imageData[i])
		}
	}
}

func TestWriteCover_PNGDataURL_WritesAsPNG(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x89, 0x50, 0x4E, 0x47}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/png;base64," + encoded

	filename, mimeType, err := WriteCover(dir, dataURL)
	if err != nil {
		t.Fatalf("WriteCover: %v", err)
	}
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

	filename, mimeType, err := WriteCover(dir, dataURL)
	if err != nil {
		t.Fatalf("WriteCover: %v", err)
	}
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

	filename, mimeType, err := WriteCover(dir, dataURL)
	if err != nil {
		t.Fatalf("WriteCover: %v", err)
	}
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

	_, _, err := WriteCover(dir, dataURL)
	if err == nil {
		t.Fatal("expected error for unsupported image format")
	}
}

func TestWriteCover_InvalidDataURL(t *testing.T) {
	dir := t.TempDir()
	_, _, err := WriteCover(dir, "https://example.com/image.jpg")
	if err == nil {
		t.Fatal("expected error for non-data URL")
	}
}

func TestWriteCover_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	coverPath := filepath.Join(dir, "cover.jpg")

	// Write an initial file.
	if err := os.WriteFile(coverPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	imageData := []byte{0xFF, 0xD8}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	if _, _, err := WriteCover(dir, "data:image/jpeg;base64,"+encoded); err != nil {
		t.Fatalf("WriteCover: %v", err)
	}

	written, err := os.ReadFile(coverPath)
	if err != nil {
		t.Fatalf("read cover.jpg: %v", err)
	}
	if string(written) == "old" {
		t.Error("cover.jpg was not overwritten")
	}
}

func TestWriteCover_RemovesStaleFormats(t *testing.T) {
	dir := t.TempDir()

	for _, ext := range []string{".jpg", ".png"} {
		if err := os.WriteFile(filepath.Join(dir, "cover"+ext), []byte("old"), 0o644); err != nil {
			t.Fatalf("setup %s: %v", ext, err)
		}
	}

	imageData := []byte{0x52, 0x49, 0x46, 0x46}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	if _, _, err := WriteCover(dir, "data:image/webp;base64,"+encoded); err != nil {
		t.Fatalf("WriteCover: %v", err)
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
