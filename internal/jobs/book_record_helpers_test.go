package jobs

import (
	"context"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"

	"github.com/stretchr/testify/require"
)

// TestCreateBookRecord_BasicFields verifies that createBookRecord correctly
// sets the book title and creates a book_file record.
func TestCreateBookRecord_BasicFields(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	p := ProcessFilePayload{
		Path:     "/library/Author/Book Title/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 1024,
	}

	book, err := createBookRecord(t.Context(), database, "My Test Book", nil, p, p.Path)
	require.NoError(t, err, "createBookRecord() error")
	require.NotNil(t, book)
	if book.Title != "My Test Book" {
		t.Errorf("book.Title = %q, want %q", book.Title, "My Test Book")
	}
}

// TestCreateBookRecord_WithMetadata verifies that book metadata from
// ExifToolOutput is persisted into the book record.
func TestCreateBookRecord_WithMetadata(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	description := "A great book"
	meta := &exif.ExifToolOutput{
		Description:     description,
		PublicationDate: "2024-01-01",
		Publisher:       "Test Publisher",
		Language:        "en",
	}
	meta.SetISBN("9781234567890")

	p := ProcessFilePayload{
		Path:     "/library/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 2048,
	}

	book, err := createBookRecord(t.Context(), database, "Metadata Book", meta, p, p.Path)
	require.NoError(t, err, "createBookRecord() error")

	if book.Description == nil || *book.Description != description {
		t.Errorf("Description = %v, want %q", book.Description, description)
	}
	if book.PublicationDate == nil || *book.PublicationDate != "2024-01-01" {
		t.Errorf("PublicationDate = %v, want 2024-01-01", book.PublicationDate)
	}
	if book.Publisher == nil || *book.Publisher != "Test Publisher" {
		t.Errorf("Publisher = %v, want Test Publisher", book.Publisher)
	}
	if book.Language == nil || *book.Language != "en" {
		t.Errorf("Language = %v, want en", book.Language)
	}
}

// TestCreateBookRecord_ISBN10 verifies that a 10-digit ISBN is stored as isbn10.
func TestCreateBookRecord_ISBN10(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	meta := &exif.ExifToolOutput{}
	meta.SetISBN("123456789X")

	p := ProcessFilePayload{
		Path:     "/library/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 1024,
	}

	book, err := createBookRecord(t.Context(), database, "ISBN10 Book", meta, p, p.Path)
	require.NoError(t, err, "createBookRecord() error")

	if book.ISBN10 == nil || *book.ISBN10 != "123456789X" {
		t.Errorf("ISBN10 = %v, want 123456789X", book.ISBN10)
	}
}

// TestLinkBookAssociations_Author verifies that an author is created and
// linked when a non-empty author name is provided.
func TestLinkBookAssociations_Author(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	book, err := createBookRecord(t.Context(), database, "Author Test Book", nil, ProcessFilePayload{
		Path:     "/library/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 512,
	}, "/library/book.epub")
	require.NoError(t, err, "create book")

	linkBookAssociations(t.Context(), database, book.ID, "Terry Pratchett", "", pathparser.PathInfo{}, "/library/book.epub")

	authors, err := database.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "get book authors")
	require.Len(t, authors, 1)
	if authors[0].Name != "Terry Pratchett" {
		t.Errorf("author name = %q, want Terry Pratchett", authors[0].Name)
	}
}

// TestLinkBookAssociations_EmptyAuthor verifies that no author is created when
// the author name is empty.
func TestLinkBookAssociations_EmptyAuthor(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	book, err := createBookRecord(t.Context(), database, "No Author Book", nil, ProcessFilePayload{
		Path:     "/library/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 512,
	}, "/library/book.epub")
	require.NoError(t, err, "create book")

	linkBookAssociations(t.Context(), database, book.ID, "", "", pathparser.PathInfo{}, "/library/book.epub")

	authors, err := database.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "get book authors")
	if len(authors) != 0 {
		t.Errorf("expected 0 authors, got %d", len(authors))
	}
}

// TestLinkBookAssociations_Series verifies that a series is created and linked
// when a non-empty series name is provided in the path info.
func TestLinkBookAssociations_Series(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	book, err := createBookRecord(t.Context(), database, "Series Book", nil, ProcessFilePayload{
		Path:     "/library/book.epub",
		FileName: "book.epub",
		FileType: "epub",
		FileSize: 512,
	}, "/library/book.epub")
	require.NoError(t, err, "create book")

	pos := 1.0
	pathInfo := pathparser.PathInfo{
		SeriesName:     "Discworld",
		SeriesPosition: &pos,
	}
	linkBookAssociations(t.Context(), database, book.ID, "", "", pathInfo, "/library/book.epub")

	series, err := database.GetBookSeries(t.Context(), book.ID)
	require.NoError(t, err, "get book series")
	require.Len(t, series, 1)
	if series[0].Series.Name != "Discworld" {
		t.Errorf("series name = %q, want Discworld", series[0].Series.Name)
	}
}

// TestCreateBookRecord_NilMetadata verifies that createBookRecord works
// correctly when no metadata is provided (nil).
func TestCreateBookRecord_NilMetadata(t *testing.T) {
	t.Parallel()

	database := newTestDB(t)

	p := ProcessFilePayload{
		Path:     "/library/book.pdf",
		FileName: "book.pdf",
		FileType: "pdf",
		FileSize: 8192,
	}

	book, err := createBookRecord(context.Background(), database, "Nil Meta Book", nil, p, p.Path)
	require.NoError(t, err, "createBookRecord() error")
	if book.Title != "Nil Meta Book" {
		t.Errorf("book.Title = %q, want Nil Meta Book", book.Title)
	}
	// Without metadata, description, publisher, etc. should be nil.
	if book.Description != nil {
		t.Errorf("expected nil Description without metadata")
	}
}
