package organize

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReorganizeFile_MovesToAuthorTitle(t *testing.T) {
	root := t.TempDir()

	// Create a flat file in root.
	srcPath := filepath.Join(root, "test.epub")
	if err := os.WriteFile(srcPath, []byte("epub content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFile(t.Context(), srcPath, root, "Jane Austen", "Pride and Prejudice")
	require.NoError(t, err, "reorganize")

	expected := filepath.Join(root, "Jane Austen", "Pride and Prejudice", "test.epub")
	if newPath != expected {
		t.Errorf("expected path %q, got %q", expected, newPath)
	}

	// Verify file exists at new location.
	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read new file")
	if string(content) != "epub content" {
		t.Errorf("content mismatch: %q", string(content))
	}

	// Verify original is gone.
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Errorf("expected original file to be gone, err: %v", err)
	}
}

func TestReorganizeFile_AlreadyInPlace(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Jane Austen", "Pride and Prejudice")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}
	filePath := filepath.Join(targetDir, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFile(t.Context(), filePath, root, "Jane Austen", "Pride and Prejudice")
	require.NoError(t, err, "reorganize")
	if newPath != filePath {
		t.Errorf("expected same path %q, got %q", filePath, newPath)
	}
}

func TestReorganizeFile_EmptyAuthorOrTitle(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFile(t.Context(), filePath, root, "", "Title")
	require.NoError(t, err, "reorganize")
	if newPath != filePath {
		t.Errorf("expected unchanged path for empty author")
	}

	newPath, err = ReorganizeFile(t.Context(), filePath, root, "Author", "")
	require.NoError(t, err, "reorganize")
	if newPath != filePath {
		t.Errorf("expected unchanged path for empty title")
	}
}

func TestReorganizeFile_CleansEmptySourceDirs(t *testing.T) {
	root := t.TempDir()

	// Create file in a nested dir that should be cleaned up after move.
	srcDir := filepath.Join(root, "OldAuthor", "OldTitle")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}
	srcPath := filepath.Join(srcDir, "book.epub")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	_, err := ReorganizeFile(t.Context(), srcPath, root, "NewAuthor", "NewTitle")
	require.NoError(t, err, "reorganize")

	// Both OldTitle/ and OldAuthor/ should be removed (empty).
	if _, err := os.Stat(filepath.Join(root, "OldAuthor")); !os.IsNotExist(err) {
		t.Errorf("expected OldAuthor dir to be cleaned up")
	}
}

func TestReorganizeFileFlat_MovesToAuthor(t *testing.T) {
	root := t.TempDir()

	srcPath := filepath.Join(root, "test.epub")
	if err := os.WriteFile(srcPath, []byte("epub content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFileFlat(t.Context(), srcPath, root, "Jane Austen")
	require.NoError(t, err, "reorganize flat")

	expected := filepath.Join(root, "Jane Austen", "test.epub")
	if newPath != expected {
		t.Errorf("expected path %q, got %q", expected, newPath)
	}

	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read new file")
	if string(content) != "epub content" {
		t.Errorf("content mismatch: %q", string(content))
	}

	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Errorf("expected original file to be gone, err: %v", err)
	}
}

func TestReorganizeFileFlat_AlreadyInPlace(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Jane Austen")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}
	filePath := filepath.Join(targetDir, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFileFlat(t.Context(), filePath, root, "Jane Austen")
	require.NoError(t, err, "reorganize flat")
	if newPath != filePath {
		t.Errorf("expected same path %q, got %q", filePath, newPath)
	}
}

func TestReorganizeFileFlat_EmptyAuthor(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	newPath, err := ReorganizeFileFlat(t.Context(), filePath, root, "")
	require.NoError(t, err, "reorganize flat")
	if newPath != filePath {
		t.Errorf("expected unchanged path for empty author")
	}
}

func TestReorganizeFileFlat_CleansEmptySourceDirs(t *testing.T) {
	root := t.TempDir()

	srcDir := filepath.Join(root, "OldAuthor", "OldTitle")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}
	srcPath := filepath.Join(srcDir, "book.epub")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	_, err := ReorganizeFileFlat(t.Context(), srcPath, root, "NewAuthor")
	require.NoError(t, err, "reorganize flat")

	if _, err := os.Stat(filepath.Join(root, "OldAuthor")); !os.IsNotExist(err) {
		t.Errorf("expected OldAuthor dir to be cleaned up")
	}
}

func TestTargetPathFlat(t *testing.T) {
	result := TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", "Jane Austen")
	expected := filepath.Join("/lib", "Jane Austen", "book.epub")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Empty author returns empty string.
	if got := TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", ""); got != "" {
		t.Errorf("expected empty for empty author, got %q", got)
	}
}

func TestReorganizeFileFlat_TargetExists(t *testing.T) {
	root := t.TempDir()
	authorDir := filepath.Join(root, "Jane Austen")
	if err := os.MkdirAll(authorDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}

	// Create the target file first.
	existingPath := filepath.Join(authorDir, "book.epub")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		require.NoError(t, err, "write existing")
	}

	// Create source file with the same name at root.
	srcPath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(srcPath, []byte("new content"), 0o644); err != nil {
		require.NoError(t, err, "write source")
	}

	_, err := ReorganizeFileFlat(t.Context(), srcPath, root, "Jane Austen")
	require.Error(t, err, "expected error when target file already exists")

	// Original file should still exist.
	if _, err := os.Stat(srcPath); err != nil {
		t.Errorf("expected original file to still exist: %v", err)
	}
}

func TestReorganizeFile_TargetExists(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Author", "Title")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		require.NoError(t, err, "mkdir")
	}

	existingPath := filepath.Join(targetDir, "book.epub")
	if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
		require.NoError(t, err, "write existing")
	}

	srcPath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(srcPath, []byte("new content"), 0o644); err != nil {
		require.NoError(t, err, "write source")
	}

	_, err := ReorganizeFile(t.Context(), srcPath, root, "Author", "Title")
	require.Error(t, err, "expected error when target file already exists")

	if _, err := os.Stat(srcPath); err != nil {
		t.Errorf("expected original file to still exist: %v", err)
	}
}

func TestReorganizeFileFlat_SanitizedAuthorEmpty(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	// Author that sanitizes to empty (only dots and special chars).
	newPath, err := ReorganizeFileFlat(t.Context(), srcPath, root, "...")
	require.NoError(t, err, "reorganize flat")
	if newPath != srcPath {
		t.Errorf("expected unchanged path for sanitized-to-empty author, got %q", newPath)
	}
}

func TestReorganizeFile_SanitizedFieldsEmpty(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		require.NoError(t, err, "write file")
	}

	// Author sanitizes to empty.
	newPath, err := ReorganizeFile(t.Context(), srcPath, root, "...", "Title")
	require.NoError(t, err, "reorganize")
	if newPath != srcPath {
		t.Errorf("expected unchanged path for sanitized-to-empty author")
	}

	// Title sanitizes to empty.
	newPath, err = ReorganizeFile(t.Context(), srcPath, root, "Author", "...")
	require.NoError(t, err, "reorganize")
	if newPath != srcPath {
		t.Errorf("expected unchanged path for sanitized-to-empty title")
	}
}

func TestTargetPathFlat_SanitizedEmpty(t *testing.T) {
	if got := TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", "..."); got != "" {
		t.Errorf("expected empty for sanitized-to-empty author, got %q", got)
	}
}

func TestTargetPath_SanitizedEmpty(t *testing.T) {
	if got := TargetPath(t.Context(), "/lib/book.epub", "/lib", "...", "Title"); got != "" {
		t.Errorf("expected empty for sanitized-to-empty author, got %q", got)
	}
	if got := TargetPath(t.Context(), "/lib/book.epub", "/lib", "Author", "..."); got != "" {
		t.Errorf("expected empty for sanitized-to-empty title, got %q", got)
	}
}
