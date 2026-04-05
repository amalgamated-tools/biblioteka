package jobs

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"

	"github.com/stretchr/testify/require"
)

func TestProcessBookFile_OrganizeFiles(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	// Create a library root with a book file at the top level.
	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	// Create a library with book_per_folder organization.
	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// Verify the original file was moved.
	if _, err := os.Stat(epubPath); !os.IsNotExist(err) {
		t.Error("expected original file to be removed after reorganization")
	}

	// Verify the file was moved to the expected Author/Title/ structure.
	expectedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "The Great Gatsby.epub")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "expected reorganized file at %q", expectedPath)

	// Verify book_files.file_path matches the new location.
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != expectedPath {
		t.Errorf("expected file path %q, got %q", expectedPath, files[0].FilePath)
	}
}

func TestProcessBookFile_OrganizeFiles_BookPerFile(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	// Create a library with book_per_file organization (flat Author/ structure).
	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFile, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// Verify the file was moved to Author/ (no title subfolder).
	expectedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby.epub")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "expected reorganized file at %q", expectedPath)

	// Verify book_files.file_path matches the new location.
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != expectedPath {
		t.Errorf("expected file path %q, got %q", expectedPath, files[0].FilePath)
	}
}

func TestProcessBookFile_OrganizeFiles_None(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	// Create a library with no file organization.
	lib, err := database.CreateLibrary(t.Context(), "Unorganized", `["`+root+`"]`, db.LibraryOrganizationNone, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// Verify the file was NOT moved — it should remain at its original path.
	_, err = os.Stat(epubPath)
	require.NoError(t, err, "expected file to remain at %q", epubPath)

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != epubPath {
		t.Errorf("expected file path %q, got %q", epubPath, files[0].FilePath)
	}
}

func TestProcessBookFile_NonExistentLibrarySkipsOrganization(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	// Use a non-existent library ID — lookup should fail gracefully and
	// skip file organization rather than error out.
	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   "nonexistent-library-id",
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// File should NOT have been moved since library lookup failed.
	_, err = os.Stat(epubPath)
	require.NoError(t, err, "expected file to remain at %q", epubPath)

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != epubPath {
		t.Errorf("expected file path %q, got %q", epubPath, files[0].FilePath)
	}
}

func TestProcessBookFile_NoLibraryIDSkipsOrganization(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	epubPath := filepath.Join(root, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	// No library ID — organization type stays empty, no file moves.
	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        epubPath,
		FileName:    "The Great Gatsby.epub",
		FileType:    "epub",
		FileSize:    1024,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	// File should remain in place.
	_, err = os.Stat(epubPath)
	require.NoError(t, err, "expected file to remain at %q", epubPath)
}

func TestProcessBookFile_ContinuesFromReorganizedPathWhenSourceMoved(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	reorganizedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "F. Scott Fitzgerald - The Great Gatsby.epub")

	require.NoError(t, os.MkdirAll(filepath.Dir(reorganizedPath), 0o755), "mkdir reorganized dir")
	testutils.MakeTestEPUB(t, reorganizedPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != reorganizedPath {
		t.Errorf("expected file path %q, got %q", reorganizedPath, files[0].FilePath)
	}
}

func TestProcessBookFile_ContinuesFromFlatReorganizedPathWhenSourceMoved(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	reorganizedPath := filepath.Join(root, "F. Scott Fitzgerald", "F. Scott Fitzgerald - The Great Gatsby.epub")

	require.NoError(t, os.MkdirAll(filepath.Dir(reorganizedPath), 0o755), "mkdir reorganized dir")
	testutils.MakeTestEPUB(t, reorganizedPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFile, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != reorganizedPath {
		t.Errorf("expected file path %q, got %q", reorganizedPath, files[0].FilePath)
	}
}

func TestProcessBookFile_FlatRecoveryDoesNotUseFolderCandidate(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	flatPath := filepath.Join(root, "F. Scott Fitzgerald", "F. Scott Fitzgerald - The Great Gatsby.epub")
	folderPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "F. Scott Fitzgerald - The Great Gatsby.epub")

	require.NoError(t, os.MkdirAll(filepath.Dir(flatPath), 0o755), "mkdir flat dir")
	require.NoError(t, os.MkdirAll(filepath.Dir(folderPath), 0o755), "mkdir folder dir")
	testutils.MakeTestEPUB(t, flatPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")
	testutils.MakeTestEPUB(t, folderPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFile, false)
	require.NoError(t, err, "create library")

	err = ProcessBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "ProcessBookFile()")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FilePath != flatPath {
		t.Errorf("expected file path %q, got %q", flatPath, files[0].FilePath)
	}
}

func TestResolveSourcePath_ReturnsErrorWhenCandidateLookupFails(t *testing.T) {
	database := newTestDB(t)
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	defer ext.Close(t.Context())

	root := t.TempDir()
	originalPath := filepath.Join(root, "F. Scott Fitzgerald - The Great Gatsby.epub")
	reorganizedPath := filepath.Join(root, "F. Scott Fitzgerald", "The Great Gatsby", "F. Scott Fitzgerald - The Great Gatsby.epub")

	// Create the file at the reorganized location only (source is "missing").
	require.NoError(t, os.MkdirAll(filepath.Dir(reorganizedPath), 0o755), "mkdir reorganized dir")
	testutils.MakeTestEPUB(t, reorganizedPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["`+root+`"]`, db.LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "create library")

	candidateLookupErr := errors.New("candidate lookup failed")
	lookup := func(ctx context.Context, database *db.DB, path string) (*db.BookFile, error) {
		switch path {
		case originalPath:
			return nil, sql.ErrNoRows
		case reorganizedPath:
			return nil, candidateLookupErr
		default:
			return database.GetBookFileByPath(ctx, path)
		}
	}

	err = processBookFile(t.Context(), database, ext, ProcessFilePayload{
		Path:        originalPath,
		FileName:    filepath.Base(originalPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	}, lookup)
	require.Error(t, err, "expected processBookFile() to return an error when candidate lookup fails due to DB error")
	require.Contains(t, err.Error(), candidateLookupErr.Error(), "expected error to include candidate lookup error")
}
