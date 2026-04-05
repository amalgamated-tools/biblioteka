package main

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/jobs"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

// newTestDB creates an in-memory SQLite database with all migrations applied.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "open")
	t.Cleanup(func() { _ = sqlDB.Close() })

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err, "pragmas")

	err = db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite)
	require.NoError(t, err, "migrations")

	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

// requireExtractor creates a metadata extractor. If ExifTool is not installed,
// the extractor silently falls back to filename-derived metadata — the tests are
// written so that assertions hold either way.
func requireExtractor(t *testing.T) *metadata.Extractor {
	t.Helper()
	ext, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
	t.Cleanup(func() { ext.Close(context.Background()) })
	return ext
}

func TestProcessFile_MOBI(t *testing.T) {
	dir := t.TempDir()
	// Name the file to match the embedded title so the assertion holds whether
	// or not exiftool is available to extract metadata from the MOBI header.
	mobiPath := filepath.Join(dir, "The Prince.mobi")
	testutils.MakeTestMOBI(t, mobiPath, "The Prince", "Niccolò Machiavelli", testutils.MOBIOptions{
		Publisher: "Public Domain",
		Language:  "en",
	})

	database := newTestDB(t)
	ext := requireExtractor(t)

	info := fileInfo(t, mobiPath)
	err := jobs.ProcessBookFile(t.Context(), database, ext, jobs.ProcessFilePayload{
		Path:     mobiPath,
		FileName: filepath.Base(mobiPath),
		FileType: "mobi",
		FileSize: info.Size(),
	})
	require.NoError(t, err, "ProcessBookFile() error")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	if books[0].Title != "The Prince" {
		t.Errorf("expected title %q, got %q", "The Prince", books[0].Title)
	}

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FileType != "mobi" {
		t.Errorf("expected file type %q, got %q", "mobi", files[0].FileType)
	}
}

func TestProcessFile_AZW3(t *testing.T) {
	dir := t.TempDir()
	// Name the file to match the embedded title so the assertion holds whether
	// or not exiftool is available to extract metadata from the MOBI header.
	azw3Path := filepath.Join(dir, "The Prince.azw3")
	testutils.MakeTestAZW3(t, azw3Path, "The Prince", "Niccolò Machiavelli", testutils.MOBIOptions{
		ISBN:      "9781234567897",
		Publisher: "Public Domain",
		Language:  "en",
	})

	database := newTestDB(t)
	ext := requireExtractor(t)

	info := fileInfo(t, azw3Path)
	err := jobs.ProcessBookFile(t.Context(), database, ext, jobs.ProcessFilePayload{
		Path:     azw3Path,
		FileName: filepath.Base(azw3Path),
		FileType: "azw3",
		FileSize: info.Size(),
	})
	require.NoError(t, err, "ProcessBookFile() error")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	if books[0].Title != "The Prince" {
		t.Errorf("expected title %q, got %q", "The Prince", books[0].Title)
	}

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FileType != "azw3" {
		t.Errorf("expected file type %q, got %q", "azw3", files[0].FileType)
	}
}

func TestProcessFile_EPUB(t *testing.T) {
	dir := t.TempDir()
	// Name the file to match the embedded title so the assertion holds whether
	// or not exiftool is available to extract EPUB metadata.
	epubPath := filepath.Join(dir, "Alice in Wonderland.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "Alice in Wonderland", "Lewis Carroll", "urn:isbn:9780141439761", testutils.EPUBOptions{
		Version:         "2.0",
		Description:     "A classic children's novel",
		Publisher:       "Macmillan",
		PublicationDate: "1865-11-26",
		Language:        "en",
	})

	database := newTestDB(t)
	ext := requireExtractor(t)

	info := fileInfo(t, epubPath)
	err := jobs.ProcessBookFile(t.Context(), database, ext, jobs.ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: info.Size(),
	})
	require.NoError(t, err, "ProcessBookFile() error")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	if books[0].Title != "Alice in Wonderland" {
		t.Errorf("expected title %q, got %q", "Alice in Wonderland", books[0].Title)
	}

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FileType != "epub" {
		t.Errorf("expected file type %q, got %q", "epub", files[0].FileType)
	}
}

func TestProcessFile_EPUB3(t *testing.T) {
	dir := t.TempDir()
	// Name the file to match the embedded title so the assertion holds whether
	// or not exiftool is available to extract EPUB metadata.
	epub3Path := filepath.Join(dir, "EPUB 3 Specification.epub")
	testutils.MakeTestEPUBWithOptions(t, epub3Path, "EPUB 3 Specification", "IDPF", "urn:isbn:9780000000000", testutils.EPUBOptions{
		Version:        "3.0",
		EPUB3Cover:     true,
		CoverImageData: testutils.TinyPNG(),
		Description:    "The EPUB 3.0 specification document",
		Publisher:      "IDPF",
		Language:       "en",
		Subjects:       []string{"Publishing", "Standards", "Digital Books"},
	})

	database := newTestDB(t)
	ext := requireExtractor(t)

	info := fileInfo(t, epub3Path)
	err := jobs.ProcessBookFile(t.Context(), database, ext, jobs.ProcessFilePayload{
		Path:     epub3Path,
		FileName: filepath.Base(epub3Path),
		FileType: "epub",
		FileSize: info.Size(),
	})
	require.NoError(t, err, "ProcessBookFile() error")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	if books[0].Title != "EPUB 3 Specification" {
		t.Errorf("expected title %q, got %q", "EPUB 3 Specification", books[0].Title)
	}

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	if files[0].FileType != "epub" {
		t.Errorf("expected file type %q, got %q", "epub", files[0].FileType)
	}
}

// fileInfo stats a file and fails the test if it does not exist.
func fileInfo(t *testing.T, path string) fs.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "stat %q", path)
	return info
}
