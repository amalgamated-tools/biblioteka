package calibre

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	_ "modernc.org/sqlite"
)

// calibreSchema is the minimal subset of the Calibre metadata.db schema needed
// for the importer tests.
const calibreSchema = `
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT 'Unknown',
    sort TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pubdate TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    series_index REAL NOT NULL DEFAULT 1.0,
    author_sort TEXT,
    isbn TEXT DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    has_cover BOOL DEFAULT 0,
    last_modified TIMESTAMP NOT NULL DEFAULT '2000-01-01 00:00:00+00:00'
);
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT,
    link TEXT NOT NULL DEFAULT ''
);
CREATE TABLE books_authors_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    UNIQUE(book, author)
);
CREATE TABLE series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT
);
CREATE TABLE books_series_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    series INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    UNIQUE(book, series)
);
CREATE TABLE publishers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE,
    sort TEXT
);
CREATE TABLE books_publishers_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    publisher INTEGER NOT NULL REFERENCES publishers(id) ON DELETE CASCADE,
    UNIQUE(book, publisher)
);
CREATE TABLE comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER UNIQUE NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    text TEXT NOT NULL
);
CREATE TABLE data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    format TEXT NOT NULL,
    uncompressed_size INTEGER NOT NULL,
    name TEXT NOT NULL,
    UNIQUE(book, format)
);
CREATE TABLE identifiers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    val TEXT NOT NULL,
    UNIQUE(book, type)
);
CREATE TABLE languages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lang_code TEXT NOT NULL COLLATE NOCASE
);
CREATE TABLE books_languages_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    lang_code INTEGER NOT NULL REFERENCES languages(id) ON DELETE CASCADE,
    item_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE(book, lang_code)
);
`

// newTestCalibreDB creates an in-memory Calibre database with the minimal
// schema. The returned *DB must be closed by the caller.
func newTestCalibreDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "open calibre test db")
	t.Cleanup(func() { _ = sqlDB.Close() })
	_, err = sqlDB.Exec(calibreSchema)
	require.NoError(t, err, "create calibre schema")
	return newDBFromSQL(sqlDB)
}

// newTestBibliotekaDB creates an in-memory Biblioteka database with all
// migrations applied.
func newTestBibliotekaDB(t *testing.T) *db.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "open biblioteka test db")
	t.Cleanup(func() { _ = sqlDB.Close() })
	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err, "pragmas")
	err = db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite)
	require.NoError(t, err, "run migrations")
	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

// insertCalibreBook inserts a book row and returns its ID.
func insertCalibreBook(t *testing.T, cdb *DB, title, path, pubdate string, seriesIndex float64) int64 {
	t.Helper()
	res, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO books (title, path, pubdate, series_index) VALUES (?, ?, ?, ?)`,
		title, path, pubdate, seriesIndex,
	)
	require.NoError(t, err, "insert calibre book %q", title)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func insertCalibreAuthor(t *testing.T, cdb *DB, name string) int64 {
	t.Helper()
	res, err := cdb.db.ExecContext(t.Context(), `INSERT INTO authors (name) VALUES (?)`, name)
	require.NoError(t, err, "insert calibre author %q", name)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func linkBookAuthor(t *testing.T, cdb *DB, bookID, authorID int64) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO books_authors_link (book, author) VALUES (?, ?)`, bookID, authorID,
	)
	require.NoError(t, err, "link book %d to author %d", bookID, authorID)
}

func insertCalibreSeries(t *testing.T, cdb *DB, name string) int64 {
	t.Helper()
	res, err := cdb.db.ExecContext(t.Context(), `INSERT INTO series (name) VALUES (?)`, name)
	require.NoError(t, err, "insert calibre series %q", name)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func linkBookSeries(t *testing.T, cdb *DB, bookID, seriesID int64) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO books_series_link (book, series) VALUES (?, ?)`, bookID, seriesID,
	)
	require.NoError(t, err, "link book %d to series %d", bookID, seriesID)
}

func insertCalibrePublisher(t *testing.T, cdb *DB, name string) int64 {
	t.Helper()
	res, err := cdb.db.ExecContext(t.Context(), `INSERT INTO publishers (name) VALUES (?)`, name)
	require.NoError(t, err, "insert calibre publisher %q", name)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return id
}

func linkBookPublisher(t *testing.T, cdb *DB, bookID, publisherID int64) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO books_publishers_link (book, publisher) VALUES (?, ?)`, bookID, publisherID,
	)
	require.NoError(t, err, "link book %d to publisher %d", bookID, publisherID)
}

func insertCalibreComment(t *testing.T, cdb *DB, bookID int64, text string) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO comments (book, text) VALUES (?, ?)`, bookID, text,
	)
	require.NoError(t, err, "insert comment for book %d", bookID)
}

func insertCalibreFormat(t *testing.T, cdb *DB, bookID int64, format, name string, size int64) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO data (book, format, name, uncompressed_size) VALUES (?, ?, ?, ?)`,
		bookID, format, name, size,
	)
	require.NoError(t, err, "insert format %q for book %d", format, bookID)
}

func insertCalibreIdentifier(t *testing.T, cdb *DB, bookID int64, typ, val string) {
	t.Helper()
	_, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO identifiers (book, type, val) VALUES (?, ?, ?)`, bookID, typ, val,
	)
	require.NoError(t, err, "insert identifier %q for book %d", typ, bookID)
}

func insertCalibreLanguage(t *testing.T, cdb *DB, bookID int64, langCode string) {
	t.Helper()
	res, err := cdb.db.ExecContext(t.Context(),
		`INSERT INTO languages (lang_code) VALUES (?)`, langCode,
	)
	require.NoError(t, err, "insert language %q", langCode)
	langID, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = cdb.db.ExecContext(t.Context(),
		`INSERT INTO books_languages_link (book, lang_code, item_order) VALUES (?, ?, 0)`,
		bookID, langID,
	)
	require.NoError(t, err, "link book %d to language %q", bookID, langCode)
}

// TestImport_BasicBook verifies that a single Calibre book with all common
// metadata fields is imported into Biblioteka correctly.
func TestImport_BasicBook(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Dune", "Frank Herbert/Dune (1)", "1965-08-01 00:00:00+00:00", 1.0)
	authorID := insertCalibreAuthor(t, cdb, "Frank Herbert")
	linkBookAuthor(t, cdb, bookID, authorID)
	insertCalibreComment(t, cdb, bookID, "A science fiction masterpiece.")
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Dune", 512000)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780441013593")
	insertCalibreIdentifier(t, cdb, bookID, "goodreads", "234225")
	insertCalibreLanguage(t, cdb, bookID, "eng")

	publisherID := insertCalibrePublisher(t, cdb, "Chilton Books")
	linkBookPublisher(t, cdb, bookID, publisherID)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Errors)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)

	book := books[0]
	require.Equal(t, "Dune", book.Title)
	require.NotNil(t, book.Description)
	require.Equal(t, "A science fiction masterpiece.", *book.Description)
	require.NotNil(t, book.ISBN13)
	require.Equal(t, "9780441013593", *book.ISBN13)
	require.NotNil(t, book.GoodreadsID)
	require.Equal(t, "234225", *book.GoodreadsID)
	require.NotNil(t, book.Publisher)
	require.Equal(t, "Chilton Books", *book.Publisher)
	require.NotNil(t, book.Language)
	require.Equal(t, "eng", *book.Language)
	require.NotNil(t, book.PublicationDate)
	require.Equal(t, "1965-08-01", *book.PublicationDate)

	// Verify file record.
	files, err := biblDB.ListBookFiles(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "epub", files[0].FileType)
	require.Equal(t, "Dune.epub", files[0].FileName)
	require.Equal(t, int64(512000), files[0].FileSize)
	require.Equal(t, "/calibre/library/Frank Herbert/Dune (1)/Dune.epub", files[0].FilePath)

	// Verify author association.
	authors, err := biblDB.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, authors, 1)
	require.Equal(t, "Frank Herbert", authors[0].Name)
}

// TestImport_SeriesPosition verifies that a book's series membership and
// position are imported correctly.
func TestImport_SeriesPosition(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "The Fellowship of the Ring", "J.R.R. Tolkien/The Lord of the Rings (1)", "1954-07-29 00:00:00+00:00", 1.0)
	authorID := insertCalibreAuthor(t, cdb, "J.R.R. Tolkien")
	linkBookAuthor(t, cdb, bookID, authorID)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "The Fellowship of the Ring", 1024000)
	seriesID := insertCalibreSeries(t, cdb, "The Lord of the Rings")
	linkBookSeries(t, cdb, bookID, seriesID)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)

	seriesEntries, err := biblDB.GetBookSeries(t.Context(), books[0].ID)
	require.NoError(t, err)
	require.Len(t, seriesEntries, 1)
	require.Equal(t, "The Lord of the Rings", seriesEntries[0].Series.Name)
	require.NotNil(t, seriesEntries[0].Position)
	require.InDelta(t, 1.0, *seriesEntries[0].Position, 0.001)
}

// TestImport_MultipleFormats verifies that all file formats for a book are
// registered as separate book_file records.
func TestImport_MultipleFormats(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Foundation", "Isaac Asimov/Foundation (2)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Foundation", 400000)
	insertCalibreFormat(t, cdb, bookID, "MOBI", "Foundation", 450000)
	insertCalibreFormat(t, cdb, bookID, "PDF", "Foundation", 800000)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)

	files, err := biblDB.ListBookFiles(t.Context(), books[0].ID)
	require.NoError(t, err)
	require.Len(t, files, 3)

	typesSeen := make(map[string]bool)
	for _, f := range files {
		typesSeen[f.FileType] = true
	}
	require.True(t, typesSeen["epub"], "expected epub file")
	require.True(t, typesSeen["mobi"], "expected mobi file")
	require.True(t, typesSeen["pdf"], "expected pdf file")
}

// TestImport_Deduplication verifies that a second import of the same library
// skips books whose file paths are already in book_files.
func TestImport_Deduplication(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "1984", "George Orwell/1984 (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "1984", 300000)

	opts := ImportOptions{LibraryPath: "/calibre/library"}

	// First import.
	result1, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result1.Imported)
	require.Equal(t, 0, result1.Skipped)

	// Second import of the same library: all books should be skipped.
	result2, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 0, result2.Imported)
	require.Equal(t, 1, result2.Skipped)

	// Exactly one book record should exist.
	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
}

// TestImport_LibraryAssociation verifies that books are linked to the given
// Biblioteka library when LibraryID is provided.
func TestImport_LibraryAssociation(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	lib, err := biblDB.CreateLibrary(t.Context(), "My Library", "/calibre/library", db.LibraryOrganizationNone, false)
	require.NoError(t, err)

	bookID := insertCalibreBook(t, cdb, "Brave New World", "Aldous Huxley/Brave New World (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Brave New World", 200000)

	opts := ImportOptions{LibraryPath: "/calibre/library", LibraryID: lib.ID}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	libBooks, err := biblDB.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err)
	require.Len(t, libBooks, 1)
	require.Equal(t, "Brave New World", libBooks[0].Title)
}

// TestImport_ISBNClassification verifies that Calibre isbn identifiers are
// correctly classified as isbn10 or isbn13.
func TestImport_ISBNClassification(t *testing.T) {
	tests := []struct {
		name        string
		calibreType string
		calibreVal  string
		wantISBN10  string
		wantISBN13  string
	}{
		{
			name:        "isbn13 via type",
			calibreType: "isbn13",
			calibreVal:  "978-0-06-112008-4",
			wantISBN13:  "9780061120084",
		},
		{
			name:        "isbn13 via generic isbn type",
			calibreType: "isbn",
			calibreVal:  "9780061120084",
			wantISBN13:  "9780061120084",
		},
		{
			name:        "isbn10 via explicit type",
			calibreType: "isbn10",
			calibreVal:  "0-06-112008-1",
			wantISBN10:  "0061120081",
		},
		{
			name:        "isbn10 via generic isbn type",
			calibreType: "isbn",
			calibreVal:  "0-06-112008-1",
			wantISBN10:  "0061120081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdb := newTestCalibreDB(t)
			biblDB := newTestBibliotekaDB(t)

			bookID := insertCalibreBook(t, cdb, "To Kill a Mockingbird", "Harper Lee/To Kill a Mockingbird (1)", "", 1.0)
			insertCalibreFormat(t, cdb, bookID, "EPUB", "To Kill a Mockingbird", 100000)
			insertCalibreIdentifier(t, cdb, bookID, tt.calibreType, tt.calibreVal)

			opts := ImportOptions{LibraryPath: "/calibre/library"}
			_, err := runImport(t.Context(), biblDB, cdb, opts)
			require.NoError(t, err)

			books, err := biblDB.ListBooks(t.Context())
			require.NoError(t, err)
			require.Len(t, books, 1)

			if tt.wantISBN10 != "" {
				require.NotNil(t, books[0].ISBN10)
				require.Equal(t, tt.wantISBN10, *books[0].ISBN10)
			}
			if tt.wantISBN13 != "" {
				require.NotNil(t, books[0].ISBN13)
				require.Equal(t, tt.wantISBN13, *books[0].ISBN13)
			}
		})
	}
}

// TestImport_EmptyPublicationDate verifies that Calibre's sentinel date
// (0101-01-01) is not written as the publication date.
func TestImport_EmptyPublicationDate(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	// Calibre sentinel date for "no date set".
	bookID := insertCalibreBook(t, cdb, "Unknown Date Book", "Author/Unknown Date Book (1)", "0101-01-01 00:00:00+00:00", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Unknown Date Book", 100000)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	_, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Nil(t, books[0].PublicationDate, "sentinel date should not be imported")
}

// TestImport_MultipleAuthors verifies that a book with multiple authors has
// all of them linked correctly.
func TestImport_MultipleAuthors(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Good Omens", "Terry Pratchett & Neil Gaiman/Good Omens (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Good Omens", 300000)
	a1 := insertCalibreAuthor(t, cdb, "Terry Pratchett")
	a2 := insertCalibreAuthor(t, cdb, "Neil Gaiman")
	linkBookAuthor(t, cdb, bookID, a1)
	linkBookAuthor(t, cdb, bookID, a2)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)

	authors, err := biblDB.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err)
	require.Len(t, authors, 2)

	authorNames := []string{authors[0].Name, authors[1].Name}
	require.Contains(t, authorNames, "Terry Pratchett")
	require.Contains(t, authorNames, "Neil Gaiman")
}

// TestImport_EmptyLibrary verifies that importing an empty Calibre library
// returns a zero-result without errors.
func TestImport_EmptyLibrary(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 0, result.Total)
	require.Equal(t, 0, result.Imported)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Errors)
}

// TestParseCalibreDate verifies date parsing for common Calibre formats.
func TestParseCalibreDate(t *testing.T) {
	tests := []struct {
		input    string
		wantYear int
		wantZero bool
	}{
		{"2001-07-11T00:00:00+00:00", 2001, false},
		{"1965-08-01 00:00:00+00:00", 1965, false},
		{"0101-01-01 00:00:00+00:00", 101, false},
		{"", 0, true},
		{"not-a-date", 0, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
			got := parseCalibreDate(tt.input)
			if tt.wantZero {
				require.True(t, got.IsZero(), "expected zero time for %q", tt.input)
			} else {
				require.False(t, got.IsZero(), "expected non-zero time for %q", tt.input)
				require.Equal(t, tt.wantYear, got.Year())
			}
		})
	}
}

// TestFormat_FilePath verifies that Format.FilePath constructs the correct
// absolute path.
func TestFormat_FilePath(t *testing.T) {
	f := Format{FormatCode: "EPUB", Name: "My Book"}
	got := f.FilePath("/library", "Author/My Book (1)")
	require.Equal(t, "/library/Author/My Book (1)/My Book.epub", got)
}

// TestImport_NoFormats verifies that a Calibre book with no file formats is
// skipped to keep the import idempotent (without formats there is no file path
// to deduplicate on).
func TestImport_NoFormats(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	insertCalibreBook(t, cdb, "Format-less Book", "Some Author/Format-less Book (1)", "", 1.0)

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	result, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 0, result.Imported, "format-less book should be skipped")
	require.Equal(t, 1, result.Skipped)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Empty(t, books, "format-less book should not be written to Biblioteka")
}

// TestImport_AsinIdentifier verifies that the Calibre "asin" identifier type
// is mapped to the ASIN field.
func TestImport_AsinIdentifier(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Some Kindle Book", "Author/Some Kindle Book (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "MOBI", "Some Kindle Book", 200000)
	insertCalibreIdentifier(t, cdb, bookID, "asin", "B001234567")

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	_, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.NotNil(t, books[0].ASIN)
	require.Equal(t, "B001234567", *books[0].ASIN)
}

// TestImport_ISBNPriority verifies that when both isbn13 and a generic isbn
// identifier are present and both normalize to 13 digits, the isbn13 value
// wins (higher priority).
func TestImport_ISBNPriority(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "ISBN Priority", "Author/ISBN Priority (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "ISBN Priority", 100000)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9781234567890")
	insertCalibreIdentifier(t, cdb, bookID, "isbn", "9780987654321")

	opts := ImportOptions{LibraryPath: "/calibre/library"}
	_, err := runImport(t.Context(), biblDB, cdb, opts)
	require.NoError(t, err)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.NotNil(t, books[0].ISBN13)
	require.Equal(t, "9781234567890", *books[0].ISBN13, "isbn13 should take priority over generic isbn")
}

// TestImport_InvalidLibraryID verifies that an invalid library ID causes
// the import to fail before processing any books.
func TestImport_InvalidLibraryID(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Some Book", "Author/Some Book (1)", "", 1.0)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Some Book", 100000)

	opts := ImportOptions{LibraryPath: "/calibre/library", LibraryID: "nonexistent-library-id"}
	_, err := runImport(t.Context(), biblDB, cdb, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
