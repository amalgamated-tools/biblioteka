package organize

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplace_Success(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := renameNoReplace(t.Context(), oldPath, newPath); err != nil {
		t.Fatalf("renameNoReplace: %v", err)
	}

	// New file should exist with correct content.
	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new file: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("expected content %q, got %q", "hello", string(content))
	}

	// Old file should be gone.
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("expected old file to be removed, got err: %v", err)
	}
}

func TestRenameNoReplace_DestinationExists(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write old file: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	err := renameNoReplace(t.Context(), oldPath, newPath)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected os.ErrExist, got: %v", err)
	}

	// Existing destination should be unchanged.
	content, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(content) != "existing" {
		t.Errorf("expected destination unchanged %q, got %q", "existing", string(content))
	}

	// Source should still exist.
	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("expected source file to still exist: %v", err)
	}
}

func TestRenameNoReplace_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "nonexistent.txt")
	newPath := filepath.Join(dir, "new.txt")

	err := renameNoReplace(t.Context(), oldPath, newPath)
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}
