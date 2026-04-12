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
	require.NoError(t, os.WriteFile(srcPath, []byte("epub content"), 0o644), "write file")

	newPath, err := ReorganizeFile(t.Context(), srcPath, root, "Jane Austen", "Pride and Prejudice")
	require.NoError(t, err, "reorganize")

	expected := filepath.Join(root, "Jane Austen", "Pride and Prejudice", "test.epub")
	require.Equal(t, expected, newPath)

	// Verify file exists at new location.
	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read new file")
	require.Equal(t, "epub content", string(content))

	// Verify original is gone.
	_, err = os.Stat(srcPath)
	require.True(t, os.IsNotExist(err), "expected original file to be gone")
}

func TestReorganizeFile_AlreadyInPlace(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Jane Austen", "Pride and Prejudice")
	require.NoError(t, os.MkdirAll(targetDir, 0o755), "mkdir")
	filePath := filepath.Join(targetDir, "book.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644), "write file")

	newPath, err := ReorganizeFile(t.Context(), filePath, root, "Jane Austen", "Pride and Prejudice")
	require.NoError(t, err, "reorganize")
	require.Equal(t, filePath, newPath)
}

func TestReorganizeFile_EmptyAuthorOrTitle(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644), "write file")

	newPath, err := ReorganizeFile(t.Context(), filePath, root, "", "Title")
	require.NoError(t, err, "reorganize")
	require.Equal(t, filePath, newPath)

	newPath, err = ReorganizeFile(t.Context(), filePath, root, "Author", "")
	require.NoError(t, err, "reorganize")
	require.Equal(t, filePath, newPath)
}

func TestReorganizeFile_CleansEmptySourceDirs(t *testing.T) {
	root := t.TempDir()

	// Create file in a nested dir that should be cleaned up after move.
	srcDir := filepath.Join(root, "OldAuthor", "OldTitle")
	require.NoError(t, os.MkdirAll(srcDir, 0o755), "mkdir")
	srcPath := filepath.Join(srcDir, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("content"), 0o644), "write file")

	_, err := ReorganizeFile(t.Context(), srcPath, root, "NewAuthor", "NewTitle")
	require.NoError(t, err, "reorganize")

	// Both OldTitle/ and OldAuthor/ should be removed (empty).
	_, err = os.Stat(filepath.Join(root, "OldAuthor"))
	require.True(t, os.IsNotExist(err), "expected OldAuthor dir to be cleaned up")
}

func TestReorganizeFileFlat_MovesToAuthor(t *testing.T) {
	root := t.TempDir()

	srcPath := filepath.Join(root, "test.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("epub content"), 0o644), "write file")

	newPath, err := ReorganizeFileFlat(t.Context(), srcPath, root, "Jane Austen")
	require.NoError(t, err, "reorganize flat")

	expected := filepath.Join(root, "Jane Austen", "test.epub")
	require.Equal(t, expected, newPath)

	content, err := os.ReadFile(newPath)
	require.NoError(t, err, "read new file")
	require.Equal(t, "epub content", string(content))

	_, err = os.Stat(srcPath)
	require.True(t, os.IsNotExist(err), "expected original file to be gone")
}

func TestReorganizeFileFlat_AlreadyInPlace(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Jane Austen")
	require.NoError(t, os.MkdirAll(targetDir, 0o755), "mkdir")
	filePath := filepath.Join(targetDir, "book.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644), "write file")

	newPath, err := ReorganizeFileFlat(t.Context(), filePath, root, "Jane Austen")
	require.NoError(t, err, "reorganize flat")
	require.Equal(t, filePath, newPath)
}

func TestReorganizeFileFlat_EmptyAuthor(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644), "write file")

	newPath, err := ReorganizeFileFlat(t.Context(), filePath, root, "")
	require.NoError(t, err, "reorganize flat")
	require.Equal(t, filePath, newPath)
}

func TestReorganizeFileFlat_CleansEmptySourceDirs(t *testing.T) {
	root := t.TempDir()

	srcDir := filepath.Join(root, "OldAuthor", "OldTitle")
	require.NoError(t, os.MkdirAll(srcDir, 0o755), "mkdir")
	srcPath := filepath.Join(srcDir, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("content"), 0o644), "write file")

	_, err := ReorganizeFileFlat(t.Context(), srcPath, root, "NewAuthor")
	require.NoError(t, err, "reorganize flat")

	_, err = os.Stat(filepath.Join(root, "OldAuthor"))
	require.True(t, os.IsNotExist(err), "expected OldAuthor dir to be cleaned up")
}

func TestTargetPathFlat(t *testing.T) {
	result := TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", "Jane Austen")
	expected := filepath.Join("/lib", "Jane Austen", "book.epub")
	require.Equal(t, expected, result)

	// Empty author returns empty string.
	require.Equal(t, "", TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", ""))
}

func TestReorganizeFileFlat_TargetExists(t *testing.T) {
	root := t.TempDir()
	authorDir := filepath.Join(root, "Jane Austen")
	require.NoError(t, os.MkdirAll(authorDir, 0o755), "mkdir")

	// Create the target file first.
	existingPath := filepath.Join(authorDir, "book.epub")
	require.NoError(t, os.WriteFile(existingPath, []byte("existing"), 0o644), "write existing")

	// Create source file with the same name at root.
	srcPath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("new content"), 0o644), "write source")

	_, err := ReorganizeFileFlat(t.Context(), srcPath, root, "Jane Austen")
	require.Error(t, err, "expected error when target file already exists")

	// Original file should still exist.
	_, err = os.Stat(srcPath)
	require.NoError(t, err, "expected original file to still exist")
}

func TestReorganizeFile_TargetExists(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "Author", "Title")
	require.NoError(t, os.MkdirAll(targetDir, 0o755), "mkdir")

	existingPath := filepath.Join(targetDir, "book.epub")
	require.NoError(t, os.WriteFile(existingPath, []byte("existing"), 0o644), "write existing")

	srcPath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("new content"), 0o644), "write source")

	_, err := ReorganizeFile(t.Context(), srcPath, root, "Author", "Title")
	require.Error(t, err, "expected error when target file already exists")

	_, err = os.Stat(srcPath)
	require.NoError(t, err, "expected original file to still exist")
}

func TestReorganizeFileFlat_SanitizedAuthorEmpty(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("content"), 0o644), "write file")

	// Author that sanitizes to empty (only dots and special chars).
	newPath, err := ReorganizeFileFlat(t.Context(), srcPath, root, "...")
	require.NoError(t, err, "reorganize flat")
	require.Equal(t, srcPath, newPath)
}

func TestReorganizeFile_SanitizedFieldsEmpty(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "book.epub")
	require.NoError(t, os.WriteFile(srcPath, []byte("content"), 0o644), "write file")

	// Author sanitizes to empty.
	newPath, err := ReorganizeFile(t.Context(), srcPath, root, "...", "Title")
	require.NoError(t, err, "reorganize")
	require.Equal(t, srcPath, newPath)

	// Title sanitizes to empty.
	newPath, err = ReorganizeFile(t.Context(), srcPath, root, "Author", "...")
	require.NoError(t, err, "reorganize")
	require.Equal(t, srcPath, newPath)
}

func TestTargetPathFlat_SanitizedEmpty(t *testing.T) {
	require.Equal(t, "", TargetPathFlat(t.Context(), "/lib/book.epub", "/lib", "..."))
}

func TestTargetPath_SanitizedEmpty(t *testing.T) {
	require.Equal(t, "", TargetPath(t.Context(), "/lib/book.epub", "/lib", "...", "Title"))
	require.Equal(t, "", TargetPath(t.Context(), "/lib/book.epub", "/lib", "Author", "..."))
}

// TestSanitizeDirName documents the exact character-filtering behaviour of
// sanitizeDirName, which is applied to untrusted epub author/title metadata
// before constructing filesystem paths. Explicit unit tests here make future
// changes to the sanitizer easy to reason about and safe to verify.
func TestSanitizeDirName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Ordinary text is left untouched.
		{"plain text", "Jane Austen", "Jane Austen"},
		{"unicode preserved", "Ångström", "Ångström"},
		{"internal dot preserved", "J.R.R. Tolkien", "J.R.R. Tolkien"},

		// Whitespace handling.
		{"leading and trailing spaces trimmed", "  Author Name  ", "Author Name"},
		{"only spaces", "   ", ""},

		// Path separators — both Unix and Windows variants must be removed so
		// that metadata cannot inject additional directory levels.
		{"forward slash removed", "path/name", "pathname"},
		{"backslash removed", `back\slash`, "backslash"},

		// Null byte removal prevents filesystem confusion on some platforms.
		{"null byte removed", "a\x00b", "ab"},

		// Windows-problematic characters. Even though Biblioteka targets Linux,
		// removing them keeps directory names portable and avoids surprises when
		// syncing libraries to Windows hosts.
		{"colon removed", "author:name", "authorname"},
		{"asterisk removed", "author*name", "authorname"},
		{"question mark removed", "author?name", "authorname"},
		{"double quote removed", `author"name`, "authorname"},
		{"less than removed", "author<name", "authorname"},
		{"greater than removed", "author>name", "authorname"},
		{"pipe removed", "author|name", "authorname"},

		// Leading dots are stripped so the resulting directory is not hidden on
		// Unix filesystems (where names starting with '.' are hidden by default).
		{"single leading dot stripped", ".hidden", "hidden"},
		{"multiple leading dots stripped", "...name", "name"},
		{"only leading dots become empty", "...", ""},

		// Internal and trailing dots are left intact — they are valid and common
		// (e.g. "J.R.R." or "Vol. 1").
		{"trailing dots preserved", "Vol. 1.", "Vol. 1."},

		// Combined / edge cases.
		{"all special chars become empty", "/\\:\x00*?\"<>|", ""},
		{"empty string", "", ""},
		{"special chars with surrounding text", "auth/or:name", "authorname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, sanitizeDirName(tt.input))
		})
	}
}
