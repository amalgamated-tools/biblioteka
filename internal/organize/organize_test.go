package organize

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReorganizeFile_MovesToAuthorTitle(t *testing.T) {
	root := t.TempDir()

	// Create a flat file in root.
	srcPath := filepath.Join(root, "test.epub")
	if err := os.WriteFile(srcPath, []byte("epub content"), 0o644); err != nil {
		failNowf(t, "write file: %v", err)
	}

	newPath, err := ReorganizeFile(srcPath, root, "Jane Austen", "Pride and Prejudice")
	if err != nil {
		failNowf(t, "reorganize: %v", err)
	}

	expected := filepath.Join(root, "Jane Austen", "Pride and Prejudice", "test.epub")
	if newPath != expected {
		failf(t, "expected path %q, got %q", expected, newPath)
	}

	// Verify file exists at new location.
	content, err := os.ReadFile(newPath)
	if err != nil {
		failNowf(t, "read new file: %v", err)
	}
	if string(content) != "epub content" {
		failf(t, "content mismatch: %q", string(content))
	}

	// Verify original is gone.
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		failf(t, "expected original file to be gone, err: %v", err)
	}
}

func TestReorganizeFile_AlreadyInPlace(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Jane Austen", "Pride and Prejudice")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		failNowf(t, "mkdir: %v", err)
	}
	filePath := filepath.Join(targetDir, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		failNowf(t, "write file: %v", err)
	}

	newPath, err := ReorganizeFile(filePath, root, "Jane Austen", "Pride and Prejudice")
	if err != nil {
		failNowf(t, "reorganize: %v", err)
	}
	if newPath != filePath {
		failf(t, "expected same path %q, got %q", filePath, newPath)
	}
}

func TestReorganizeFile_EmptyAuthorOrTitle(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "book.epub")
	if err := os.WriteFile(filePath, []byte("content"), 0o644); err != nil {
		failNowf(t, "write file: %v", err)
	}

	newPath, err := ReorganizeFile(filePath, root, "", "Title")
	if err != nil {
		failNowf(t, "reorganize: %v", err)
	}
	if newPath != filePath {
		failf(t, "expected unchanged path for empty author")
	}

	newPath, err = ReorganizeFile(filePath, root, "Author", "")
	if err != nil {
		failNowf(t, "reorganize: %v", err)
	}
	if newPath != filePath {
		failf(t, "expected unchanged path for empty title")
	}
}

func TestReorganizeFile_CleansEmptySourceDirs(t *testing.T) {
	root := t.TempDir()

	// Create file in a nested dir that should be cleaned up after move.
	srcDir := filepath.Join(root, "OldAuthor", "OldTitle")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		failNowf(t, "mkdir: %v", err)
	}
	srcPath := filepath.Join(srcDir, "book.epub")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		failNowf(t, "write file: %v", err)
	}

	_, err := ReorganizeFile(srcPath, root, "NewAuthor", "NewTitle")
	if err != nil {
		failNowf(t, "reorganize: %v", err)
	}

	// Both OldTitle/ and OldAuthor/ should be removed (empty).
	if _, err := os.Stat(filepath.Join(root, "OldAuthor")); !os.IsNotExist(err) {
		failf(t, "expected OldAuthor dir to be cleaned up")
	}
}
