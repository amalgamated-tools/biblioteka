package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

// ── isPathUnderRoot ──────────────────────────────────────────────────────────

func TestIsPathUnderRoot_EqualToRoot(t *testing.T) {
	dir := t.TempDir()
	require.True(t, isPathUnderRoot(dir, dir))
}

func TestIsPathUnderRoot_NestedUnderRoot(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "subdir", "file.epub")
	require.True(t, isPathUnderRoot(child, dir))
}

func TestIsPathUnderRoot_Sibling_NotUnder(t *testing.T) {
	// /tmp/lib and /tmp/library — "lib" must not match as a prefix of "library".
	parent := t.TempDir()
	lib := filepath.Join(parent, "lib")
	require.NoError(t, os.MkdirAll(lib, 0o755))
	library := filepath.Join(parent, "library")
	require.NoError(t, os.MkdirAll(library, 0o755))

	// A file inside /tmp/.../library must NOT be considered under /tmp/.../lib.
	fileInLibrary := filepath.Join(library, "book.epub")
	require.False(t, isPathUnderRoot(fileInLibrary, lib))
}

func TestIsPathUnderRoot_ParentOfRoot_NotUnder(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(child, 0o755))

	// The root's parent should not be considered under the root.
	require.False(t, isPathUnderRoot(dir, child))
}

func TestIsPathUnderRoot_NonExistentChild(t *testing.T) {
	// The child does not exist on disk; function should fall back to cleaned paths.
	dir := t.TempDir()
	child := filepath.Join(dir, "nonexistent", "book.epub")
	require.True(t, isPathUnderRoot(child, dir))
}

func TestIsPathUnderRoot_NonExistentRoot_ReturnsFalse(t *testing.T) {
	// An absolute path that does not resolve and an unrelated root.
	dir := t.TempDir()
	otherDir := t.TempDir()
	child := filepath.Join(dir, "book.epub")
	require.False(t, isPathUnderRoot(child, otherDir))
}

// ── isBookFilePathAllowed ────────────────────────────────────────────────────

func TestIsBookFilePathAllowed_AllowedPath(t *testing.T) {
	d := newTestDB(t)
	// createTestLibrary is declared in books_files_test.go.
	dir := createTestLibrary(t, d)

	allowed, err := isBookFilePathAllowed(t.Context(), d, filepath.Join(dir, "book.epub"))
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestIsBookFilePathAllowed_DisallowedPath(t *testing.T) {
	d := newTestDB(t)
	createTestLibrary(t, d)

	other := t.TempDir()
	allowed, err := isBookFilePathAllowed(t.Context(), d, filepath.Join(other, "book.epub"))
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestIsBookFilePathAllowed_NoLibraries(t *testing.T) {
	d := newTestDB(t)

	allowed, err := isBookFilePathAllowed(t.Context(), d, "/any/path/book.epub")
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestIsBookFilePathAllowed_MultipleLibraries_MatchesSecond(t *testing.T) {
	d := newTestDB(t)
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	p1, err := json.Marshal([]string{dir1})
	require.NoError(t, err)
	_, err = d.CreateLibrary(t.Context(), "Library 1", string(p1), db.LibraryOrganizationNone, false)
	require.NoError(t, err)

	p2, err := json.Marshal([]string{dir2})
	require.NoError(t, err)
	_, err = d.CreateLibrary(t.Context(), "Library 2", string(p2), db.LibraryOrganizationNone, false)
	require.NoError(t, err)

	// A file under dir2 should be allowed even though dir1 doesn't match.
	allowed, err := isBookFilePathAllowed(t.Context(), d, filepath.Join(dir2, "book.epub"))
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestIsBookFilePathAllowed_InvalidPathsJSON_SkipsLibrary(t *testing.T) {
	d := newTestDB(t)

	// Insert a library with invalid JSON paths directly; CreateLibrary validates
	// the name but stores the paths string as-is in the DB.
	_, err := d.DB.ExecContext(t.Context(),
		`INSERT INTO libraries (name, paths, organization_type, monitored) VALUES ('Bad Lib', 'not-json', 'none', 0)`,
	)
	require.NoError(t, err)

	// The bad library should be skipped, so no path is allowed.
	allowed, err := isBookFilePathAllowed(t.Context(), d, "/some/path/book.epub")
	require.NoError(t, err)
	require.False(t, allowed, "library with unparseable paths JSON should be skipped")
}
