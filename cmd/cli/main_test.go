package main

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name         string
		fileName     string
		fileType     string
		expectedTitle string
		createFile   func(t *testing.T, path string)
	}{
		{
			name:          "MOBI",
			fileName:      "The Prince.mobi",
			fileType:      "mobi",
			expectedTitle: "The Prince",
			createFile: func(t *testing.T, path string) {
				testutils.MakeTestMOBI(t, path, "The Prince", "Niccolò Machiavelli", testutils.MOBIOptions{
					Publisher: "Public Domain",
					Language:  "en",
				})
			},
		},
		{
			name:          "AZW3",
			fileName:      "The Prince.azw3",
			fileType:      "azw3",
			expectedTitle: "The Prince",
			createFile: func(t *testing.T, path string) {
				testutils.MakeTestAZW3(t, path, "The Prince", "Niccolò Machiavelli", testutils.MOBIOptions{
					ISBN:      "9781234567897",
					Publisher: "Public Domain",
					Language:  "en",
				})
			},
		},
		{
			name:          "EPUB",
			fileName:      "Alice in Wonderland.epub",
			fileType:      "epub",
			expectedTitle: "Alice in Wonderland",
			createFile: func(t *testing.T, path string) {
				testutils.MakeTestEPUBWithOptions(t, path, "Alice in Wonderland", "Lewis Carroll", "urn:isbn:9780141439761", testutils.EPUBOptions{
					Version:         "2.0",
					Description:     "A classic children's novel",
					Publisher:       "Macmillan",
					PublicationDate: "1865-11-26",
					Language:        "en",
				})
			},
		},
		{
			name:          "EPUB3",
			fileName:      "EPUB 3 Specification.epub",
			fileType:      "epub",
			expectedTitle: "EPUB 3 Specification",
			createFile: func(t *testing.T, path string) {
				testutils.MakeTestEPUBWithOptions(t, path, "EPUB 3 Specification", "IDPF", "urn:isbn:9780000000000", testutils.EPUBOptions{
					Version:        "3.0",
					EPUB3Cover:     true,
					CoverImageData: testutils.TinyPNG(),
					Description:    "The EPUB 3.0 specification document",
					Publisher:      "IDPF",
					Language:       "en",
					Subjects:       []string{"Publishing", "Standards", "Digital Books"},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.fileName)
			tt.createFile(t, path)

			database := newTestDB(t)
			ext := requireExtractor(t)

			info := fileInfo(t, path)
			err := jobs.ProcessBookFile(t.Context(), database, ext, nil, jobs.ProcessFilePayload{
				Path:     path,
				FileName: filepath.Base(path),
				FileType: tt.fileType,
				FileSize: info.Size(),
			})
			require.NoError(t, err, "ProcessBookFile() error")

			books, err := database.ListBooks(t.Context())
			require.NoError(t, err, "list books")
			require.Len(t, books, 1)
			require.Equal(t, tt.expectedTitle, books[0].Title)

			files, err := database.ListBookFiles(t.Context(), books[0].ID)
			require.NoError(t, err, "list book files")
			require.Len(t, files, 1)
			require.Equal(t, tt.fileType, files[0].FileType)
		})
	}
}

func TestPathExists(t *testing.T) {
	dir := t.TempDir()
	existingFile := filepath.Join(dir, "book.epub")
	err := os.WriteFile(existingFile, []byte("x"), 0o600)
	require.NoError(t, err, "create file")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "empty", path: "", want: false},
		{name: "missing", path: filepath.Join(dir, "missing"), want: false},
		{name: "existing file", path: existingFile, want: true},
		{name: "existing directory", path: dir, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pathExists(tt.path), "pathExists(%q)", tt.path)
		})
	}
}

func TestRunProcessFile_Errors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		errMessage string
	}{
		{
			name:       "missing file",
			path:       filepath.Join(t.TempDir(), "missing.epub"),
			errMessage: "error stating file",
		},
		{
			name:       "empty path",
			path:       "",
			errMessage: "error stating file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runProcessFile(t.Context(), tt.path)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.errMessage)
		})
	}
}

func TestRunCalibreImport_InvalidPath(t *testing.T) {
	baseDir := t.TempDir()

	filePath := filepath.Join(baseDir, "library.txt")
	err := os.WriteFile(filePath, []byte("not a directory"), 0o600)
	require.NoError(t, err, "create test file")

	dirWithoutMetadata := filepath.Join(baseDir, "no-metadata")
	err = os.Mkdir(dirWithoutMetadata, 0o755)
	require.NoError(t, err, "create directory without metadata")

	dirWithMetadataDir := filepath.Join(baseDir, "metadata-is-dir")
	err = os.Mkdir(dirWithMetadataDir, 0o755)
	require.NoError(t, err, "create metadata parent directory")
	err = os.Mkdir(filepath.Join(dirWithMetadataDir, "metadata.db"), 0o755)
	require.NoError(t, err, "create metadata.db directory")

	tests := []struct {
		name       string
		path       string
		errMessage string
	}{
		{
			name:       "missing path",
			path:       filepath.Join(baseDir, "does-not-exist"),
			errMessage: "failed to stat library path",
		},
		{
			name:       "path is file",
			path:       filePath,
			errMessage: "is not a directory",
		},
		{
			name:       "missing metadata db",
			path:       dirWithoutMetadata,
			errMessage: "does not contain metadata.db",
		},
		{
			name:       "metadata db is directory",
			path:       dirWithMetadataDir,
			errMessage: "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCalibreImport(t.Context(), tt.path, "")
			require.Error(t, err)
			require.ErrorContains(t, err, tt.errMessage)
		})
	}
}

func TestRunDBMigrate(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	dbPath := defaultSQLiteDBPath(t)
	removeSQLiteFiles(t, dbPath)
	t.Cleanup(func() {
		removeSQLiteFiles(t, dbPath)
	})

	err := runDBMigrate(t.Context())
	require.NoError(t, err)

	sqlDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err, "open sqlite database")
	t.Cleanup(func() { _ = sqlDB.Close() })

	var count int
	err = sqlDB.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	require.NoError(t, err, "query schema_migrations")
	require.Greater(t, count, 0)
}

// fileInfo stats a file and fails the test if it does not exist.
func fileInfo(t *testing.T, path string) fs.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "stat %q", path)
	return info
}

func defaultSQLiteDBPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve caller path")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	return filepath.Join(repoRoot, "db", "biblioteka.db")
}

func removeSQLiteFiles(t *testing.T, dbPath string) {
	t.Helper()
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			require.NoError(t, err, "remove %q", path)
		}
	}
}
