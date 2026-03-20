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

	if err := WriteCover(dir, dataURL); err != nil {
		t.Fatalf("WriteCover: %v", err)
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

func TestWriteCover_PNGDataURL_WritesAsJPG(t *testing.T) {
	dir := t.TempDir()
	imageData := []byte{0x89, 0x50, 0x4E, 0x47}
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := "data:image/png;base64," + encoded

	if err := WriteCover(dir, dataURL); err != nil {
		t.Fatalf("WriteCover: %v", err)
	}

	// Should still be named cover.jpg regardless of source format.
	if _, err := os.Stat(filepath.Join(dir, "cover.jpg")); err != nil {
		t.Errorf("cover.jpg not found: %v", err)
	}
}

func TestWriteCover_InvalidDataURL(t *testing.T) {
	dir := t.TempDir()
	err := WriteCover(dir, "https://example.com/image.jpg")
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
	if err := WriteCover(dir, "data:image/jpeg;base64,"+encoded); err != nil {
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
