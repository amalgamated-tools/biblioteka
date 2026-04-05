package jobs

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

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

func TestProcessFileHandler(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")

	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 1024,
	})
	require.NoError(t, err, "marshal")

	require.NoError(t, handler(t.Context(), payload), "handler")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	require.Equal(t, "The Great Gatsby", books[0].Title)

	files, err := database.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err, "list book files")
	require.Len(t, files, 1)
	require.Equal(t, "epub", files[0].FileType)
	require.Equal(t, epubPath, files[0].FilePath)
	require.Equal(t, int64(1024), files[0].FileSize)

	// Verify author was created and associated with the book
	authors, err := database.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, authors, 1)
	require.Equal(t, "F. Scott Fitzgerald", authors[0].Name)

	// Verify ISBN13 was extracted and normalized
	require.NotNil(t, books[0].ISBN13)
	require.Equal(t, "9780743273565", *books[0].ISBN13)

	// Verify language was extracted from EPUB metadata
	require.NotNil(t, books[0].Language)
	require.Equal(t, "en", *books[0].Language)
}

func TestProcessFileHandler_MetadataFields(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "rich.epub")

	testutils.MakeTestEPUBWithOptions(t, epubPath, "Dune", "Frank Herbert", "urn:isbn:9780441172719", testutils.EPUBOptions{
		Description:     "A science fiction masterpiece",
		Publisher:       "Chilton Books",
		PublicationDate: "1965-08-01",
	})

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 2048,
	})
	require.NoError(t, err, "marshal")

	require.NoError(t, handler(t.Context(), payload), "handler")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
	book := books[0]

	require.Equal(t, "Dune", book.Title)
	require.NotNil(t, book.Description)
	require.Equal(t, "A science fiction masterpiece", *book.Description)
	require.NotNil(t, book.Publisher)
	require.Equal(t, "Chilton Books", *book.Publisher)
	require.NotNil(t, book.PublicationDate)
	require.Equal(t, "1965-08-01", *book.PublicationDate)
	require.NotNil(t, book.Language)
	require.Equal(t, "en", *book.Language)
	require.NotNil(t, book.ISBN13)
	require.Equal(t, "9780441172719", *book.ISBN13)

	// Verify author creation and association
	authors, err := database.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, authors, 1)
	require.Equal(t, "Frank Herbert", authors[0].Name)
}

func TestProcessFileHandler_EmptyPath(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	payload, err := json.Marshal(ProcessFilePayload{Path: ""})
	require.NoError(t, err, "marshal")
	err = handler(t.Context(), payload)
	require.Error(t, err, "expected error for empty path")
}

func TestProcessFileHandler_AuthorAndLibraryLinking(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	// Create a library to link the book to.
	lib, err := database.CreateLibrary(t.Context(), "Fiction", `["/books"]`, db.LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "create library")

	// Set up: Author/Title.epub structure in a temp dir simulating a library root.
	root := t.TempDir()
	authorDir := filepath.Join(root, "F. Scott Fitzgerald")
	require.NoError(t, os.MkdirAll(authorDir, 0o755), "mkdir")
	epubPath := filepath.Join(authorDir, "The Great Gatsby.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:        epubPath,
		FileName:    filepath.Base(epubPath),
		FileType:    "epub",
		FileSize:    1024,
		LibraryID:   lib.ID,
		LibraryRoot: root,
	})
	require.NoError(t, err, "marshal")

	require.NoError(t, handler(t.Context(), payload), "handler")

	// Verify book was created.
	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	// Verify author was linked.
	authors, err := database.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, authors, 1)
	require.Equal(t, "F. Scott Fitzgerald", authors[0].Name)

	// Verify book was added to library.
	libBooks, err := database.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "list library books")
	require.Len(t, libBooks, 1)
}

func TestProcessFileHandler_SeriesFromPath(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)

	// Set up: Author/Series/N. Title.epub
	root := t.TempDir()
	seriesDir := filepath.Join(root, "Alexander McCall Smith", "No. 1 Ladies' Detective Agency")
	require.NoError(t, os.MkdirAll(seriesDir, 0o755), "mkdir")
	epubPath := filepath.Join(seriesDir, "10. Tea Time for the Traditionally Built (2009).epub")
	testutils.MakeTestEPUB(t, epubPath, "", "", "")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:        epubPath,
		FileName:    filepath.Base(epubPath),
		FileType:    "epub",
		FileSize:    512,
		LibraryRoot: root,
	})
	require.NoError(t, err, "marshal")

	require.NoError(t, handler(t.Context(), payload), "handler")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)

	// Verify series was linked.
	seriesEntries, err := database.GetBookSeries(t.Context(), books[0].ID)
	require.NoError(t, err, "get book series")
	require.Len(t, seriesEntries, 1)
	require.Equal(t, "No. 1 Ladies' Detective Agency", seriesEntries[0].Series.Name)
	require.NotNil(t, seriesEntries[0].Position)
	require.Equal(t, float64(10), *seriesEntries[0].Position)

	// Verify author was linked from directory.
	authors, err := database.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, authors, 1)
	require.Equal(t, "Alexander McCall Smith", authors[0].Name)
}

func TestProcessFileHandler_DuplicateSkipped(t *testing.T) {
	database := newTestDB(t)
	extractor, err := metadata.NewExtractor(t.Context())
	require.NoError(t, err, "failed to create metadata extractor")
	defer extractor.Close(t.Context())
	handler := NewProcessFileHandler(database, extractor)
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "Duplicate Book", "Author", "")

	payload, err := json.Marshal(ProcessFilePayload{
		Path:     epubPath,
		FileName: filepath.Base(epubPath),
		FileType: "epub",
		FileSize: 1024,
	})
	require.NoError(t, err, "marshal")

	// Process once.
	require.NoError(t, handler(t.Context(), payload), "first handler call")

	// Process again — should be skipped (no error, no duplicate book).
	require.NoError(t, handler(t.Context(), payload), "second handler call")

	books, err := database.ListBooks(t.Context())
	require.NoError(t, err, "list books")
	require.Len(t, books, 1)
}
