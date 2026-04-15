package calibre

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// TestLoadPreview_Basic verifies that LoadPreview returns a summary with the
// correct total and at most previewLimit books.
func TestLoadPreview_Basic(t *testing.T) {
	cdb := newTestCalibreDB(t)

	// Insert more books than the preview limit.
	for i := range previewLimit + 5 {
		id := insertCalibreBook(t, cdb, "Book "+string(rune('A'+i)), "Author/Book (1)", "", 1.0)
		insertCalibreFormat(t, cdb, id, "EPUB", "Book", 100000)
	}

	preview, err := LoadPreview(t.Context(), cdb)
	require.NoError(t, err)
	require.Equal(t, previewLimit+5, preview.Total)
	require.Len(t, preview.Books, previewLimit)
}

// TestLoadPreview_Fields verifies that book fields are mapped correctly.
func TestLoadPreview_Fields(t *testing.T) {
	cdb := newTestCalibreDB(t)

	bookID := insertCalibreBook(t, cdb, "Dune", "Frank Herbert/Dune (1)", "1965-08-01 00:00:00+00:00", 1.0)
	authorID := insertCalibreAuthor(t, cdb, "Frank Herbert")
	linkBookAuthor(t, cdb, bookID, authorID)
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Dune", 512000)
	insertCalibreFormat(t, cdb, bookID, "PDF", "Dune", 800000)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780441013593")
	seriesID := insertCalibreSeries(t, cdb, "Dune Series")
	linkBookSeries(t, cdb, bookID, seriesID)
	publisherID := insertCalibrePublisher(t, cdb, "Chilton Books")
	linkBookPublisher(t, cdb, bookID, publisherID)

	preview, err := LoadPreview(t.Context(), cdb)
	require.NoError(t, err)
	require.Equal(t, 1, preview.Total)
	require.Len(t, preview.Books, 1)

	pb := preview.Books[0]
	require.Equal(t, "Dune", pb.Title)
	require.Equal(t, []string{"Frank Herbert"}, pb.Authors)
	require.Equal(t, "9780441013593", pb.ISBN13)
	require.Equal(t, "1965-08-01", pb.PublicationDate)
	require.Equal(t, "Chilton Books", pb.Publisher)
	require.Len(t, pb.Series, 1)
	require.Equal(t, "Dune Series", pb.Series[0].Name)
	require.ElementsMatch(t, []string{"epub", "pdf"}, pb.Formats)
}

// TestLoadPreview_Empty verifies that an empty Calibre library returns a zero
// preview without error.
func TestLoadPreview_Empty(t *testing.T) {
	cdb := newTestCalibreDB(t)

	preview, err := LoadPreview(t.Context(), cdb)
	require.NoError(t, err)
	require.Equal(t, 0, preview.Total)
	require.Empty(t, preview.Books)
}

// TestLoadPreview_NilSlices verifies that books with no authors or formats
// return empty slices (not nil) in the preview.
func TestLoadPreview_NilSlices(t *testing.T) {
	cdb := newTestCalibreDB(t)
	insertCalibreBook(t, cdb, "Authorless", "Unknown/Authorless (1)", "", 1.0)

	preview, err := LoadPreview(t.Context(), cdb)
	require.NoError(t, err)
	require.Len(t, preview.Books, 1)

	pb := preview.Books[0]
	require.NotNil(t, pb.Authors, "authors should never be nil")
	require.NotNil(t, pb.Formats, "formats should never be nil")
	require.NotNil(t, pb.Series, "series should never be nil")
}

// TestWebImport_Basic verifies that a simple book is imported with metadata.
func TestWebImport_Basic(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Dune", "Frank Herbert/Dune (1)", "1965-08-01 00:00:00+00:00", 1.0)
	authorID := insertCalibreAuthor(t, cdb, "Frank Herbert")
	linkBookAuthor(t, cdb, bookID, authorID)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780441013593")

	result, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Errors)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Dune", books[0].Title)
	require.NotNil(t, books[0].ISBN13)
	require.Equal(t, "9780441013593", *books[0].ISBN13)

	authors, err := biblDB.GetBookAuthors(t.Context(), books[0].ID)
	require.NoError(t, err)
	require.Len(t, authors, 1)
	require.Equal(t, "Frank Herbert", authors[0].Name)
}

// TestWebImport_NoFormats verifies that format-less books ARE imported in web
// mode (unlike the CLI importer which skips them for file-path dedup).
func TestWebImport_NoFormats(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	insertCalibreBook(t, cdb, "Metadata Only", "Author/Metadata Only (1)", "", 1.0)

	result, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Imported, "format-less books should be imported in web mode")

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
}

// TestWebImport_DeduplicateByISBN13 verifies that a book with an existing
// ISBN-13 is skipped on re-import.
func TestWebImport_DeduplicateByISBN13(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "Dune", "Frank Herbert/Dune (1)", "", 1.0)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780441013593")

	// First import: book should be written.
	result1, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result1.Imported)

	// Second import: same ISBN-13 → skip.
	result2, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 0, result2.Imported)
	require.Equal(t, 1, result2.Skipped)

	// Only one book in the database.
	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)
}

// TestWebImport_DeduplicateByASIN verifies deduplication by ASIN when no ISBN
// is present.
func TestWebImport_DeduplicateByASIN(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "A Kindle Book", "Author/Kindle (1)", "", 1.0)
	insertCalibreIdentifier(t, cdb, bookID, "asin", "B001234567")

	result1, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result1.Imported)

	result2, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result2.Skipped)
}

// TestWebImport_WithLibrary verifies that books are associated with a library
// when LibraryID is set.
func TestWebImport_WithLibrary(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	lib, err := biblDB.CreateLibrary(t.Context(), "My Library", "/books", db.LibraryOrganizationNone, false)
	require.NoError(t, err)

	bookID := insertCalibreBook(t, cdb, "Foundation", "Isaac Asimov/Foundation (1)", "", 1.0)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780553293357")

	result, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{LibraryID: lib.ID})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	libBooks, err := biblDB.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err)
	require.Len(t, libBooks, 1)
	require.Equal(t, "Foundation", libBooks[0].Title)
}

// TestWebImport_InvalidLibraryID verifies that an unknown library ID causes
// the import to fail before processing any books.
func TestWebImport_InvalidLibraryID(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	insertCalibreBook(t, cdb, "Some Book", "Author/Some Book (1)", "", 1.0)

	_, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{LibraryID: "nonexistent-id"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestWebImport_Series verifies that series memberships are linked.
func TestWebImport_Series(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	bookID := insertCalibreBook(t, cdb, "The Fellowship of the Ring", "Tolkien/LOTR (1)", "", 1.0)
	seriesID := insertCalibreSeries(t, cdb, "The Lord of the Rings")
	linkBookSeries(t, cdb, bookID, seriesID)

	result, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)

	books, err := biblDB.ListBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 1)

	seriesEntries, err := biblDB.GetBookSeries(t.Context(), books[0].ID)
	require.NoError(t, err)
	require.Len(t, seriesEntries, 1)
	require.Equal(t, "The Lord of the Rings", seriesEntries[0].Series.Name)
}

// TestWebImport_NoIdentifiers verifies that books without any external
// identifier are always imported (no deduplication applied).
func TestWebImport_NoIdentifiers(t *testing.T) {
	cdb := newTestCalibreDB(t)
	biblDB := newTestBibliotekaDB(t)

	insertCalibreBook(t, cdb, "Identifier-less Book", "Author/Book (1)", "", 1.0)

	// Import twice — both should succeed since there is nothing to dedup on.
	result1, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result1.Imported)

	result2, err := WebImport(t.Context(), biblDB, cdb, WebImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result2.Imported, "book without identifiers cannot be deduped; import again")
}
