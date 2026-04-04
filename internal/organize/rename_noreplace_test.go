package organize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenameNoReplace_Success(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	if err := os.WriteFile(oldPath, []byte("hello"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	if err := renameNoReplace(t.Context(), oldPath, newPath); err != nil {
		require.NoError(t, err, "renameNoReplace")
	}

	// New file should exist with correct content.
	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read new file")
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
		require.NoError(t, err, "write old file")
	}
	if err := os.WriteFile(newPath, []byte("existing"), 0o644); err != nil {
		require.NoError(t, err, "write new file")
	}

	err := renameNoReplace(t.Context(), oldPath, newPath)
	require.ErrorIs(t, err, os.ErrExist)

	// Existing destination should be unchanged.
	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read existing file")
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
	require.Error(t, err, "expected error for missing source, got nil")
}
