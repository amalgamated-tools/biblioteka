package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBook(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger", Description: new("The first book"), ISBN10: new("1234567890"), PublicationDate: new("1982-06-10"), Publisher: new("Grant"), Language: new("en")})
	require.NoError(t, err, "CreateBook() error")
	require.NotEqual(t, "", b.ID)
	require.Equal(t, "The Gunslinger", b.Title)
	require.NotNil(t, b.ISBN10)
	require.Equal(t, "1234567890", *b.ISBN10)
	require.False(t, b.CreatedAt.IsZero())
}

func TestGetBook(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")

	found, err := d.GetBook(t.Context(), created.ID)
	require.NoError(t, err, "GetBook() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "The Gunslinger", found.Title)
}

func TestGetBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBook(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListBooks(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "A Game of Thrones"})
	require.NoError(t, err, "CreateBook() for A Game of Thrones error")
	_, err = d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() for The Gunslinger error")

	books, err := d.ListBooks(t.Context())
	require.NoError(t, err, "ListBooks() error")
	require.Len(t, books, 2)
	require.Equal(t, "A Game of Thrones", books[0].Title)
}

func TestUpdateBook(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateBook(t.Context(), BookInput{Title: "Gunslinger"})
	require.NoError(t, err, "CreateBook() error")

	updated, err := d.UpdateBook(t.Context(), created.ID, BookInput{Title: "The Gunslinger", Description: new("Revised edition"), Language: new("en")})
	require.NoError(t, err, "UpdateBook() error")
	require.Equal(t, "The Gunslinger", updated.Title)
}

func TestDeleteBook(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")

	err = d.DeleteBook(t.Context(), b.ID)
	require.NoError(t, err, "DeleteBook() error")

	_, err = d.GetBook(t.Context(), b.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBook(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestAddBookToLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")

	err = d.AddBookToLibrary(t.Context(), lib.ID, book.ID)
	require.NoError(t, err, "AddBookToLibrary() error")

	books, err := d.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "ListBooksByLibrary() error")
	require.Len(t, books, 1)
	require.Equal(t, book.ID, books[0].ID)
}

func TestListBooksByLibraryPaginated(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	b1, err := d.CreateBook(t.Context(), BookInput{Title: "Alpha"})
	require.NoError(t, err, "CreateBook() for Alpha error")
	b2, err := d.CreateBook(t.Context(), BookInput{Title: "Beta"})
	require.NoError(t, err, "CreateBook() for Beta error")
	b3, err := d.CreateBook(t.Context(), BookInput{Title: "Gamma"})
	require.NoError(t, err, "CreateBook() for Gamma error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b1.ID)
	require.NoError(t, err, "AddBookToLibrary() for Alpha error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b2.ID)
	require.NoError(t, err, "AddBookToLibrary() for Beta error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b3.ID)
	require.NoError(t, err, "AddBookToLibrary() for Gamma error")

	books, total, err := d.ListBooksByLibraryPaginated(t.Context(), lib.ID, 2, 0)
	require.NoError(t, err, "ListBooksByLibraryPaginated() error")
	require.Equal(t, 3, total)
	require.Len(t, books, 2)
	require.Equal(t, "Alpha", books[0].Title)

	books2, total2, err := d.ListBooksByLibraryPaginated(t.Context(), lib.ID, 2, 2)
	require.NoError(t, err, "ListBooksByLibraryPaginated() page 2 error")
	require.Equal(t, 3, total2)
	require.Len(t, books2, 1)
}

func TestRemoveBookFromLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	require.NoError(t, d.AddBookToLibrary(t.Context(), lib.ID, book.ID), "AddBookToLibrary() error")

	err = d.RemoveBookFromLibrary(t.Context(), lib.ID, book.ID)
	require.NoError(t, err, "RemoveBookFromLibrary() error")

	books, err := d.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "ListBooksByLibrary() error")
	require.Len(t, books, 0)
}

func TestSetBookAuthors(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	a1, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for Stephen King error")
	a2, err := d.CreateAuthor(t.Context(), "Peter Straub", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for Peter Straub error")

	err = d.SetBookAuthors(t.Context(), book.ID, []string{a1.ID, a2.ID})
	require.NoError(t, err, "SetBookAuthors() error")

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "GetBookAuthors() error")
	require.Len(t, authors, 2)
}

func TestSetBookAuthors_Replace(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Talisman"})
	require.NoError(t, err, "CreateBook() error")
	a1, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for Stephen King error")
	a2, err := d.CreateAuthor(t.Context(), "Peter Straub", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for Peter Straub error")

	require.NoError(t, d.SetBookAuthors(t.Context(), book.ID, []string{a1.ID}), "SetBookAuthors() initial error")

	err = d.SetBookAuthors(t.Context(), book.ID, []string{a2.ID})
	require.NoError(t, err, "SetBookAuthors() replace error")

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "GetBookAuthors() error")
	require.Len(t, authors, 1)
	require.Equal(t, a2.ID, authors[0].ID)
}

func TestSetBookSeries(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	require.NotNil(t, book)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")
	require.NotNil(t, s)

	err = d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(float64(1))}})
	require.NoError(t, err, "SetBookSeries() error")

	entries, err := d.GetBookSeries(t.Context(), book.ID)
	require.NoError(t, err, "GetBookSeries() error")
	require.Len(t, entries, 1)
	require.Equal(t, s.ID, entries[0].Series.ID)
	require.NotNil(t, entries[0].Position)
	require.Equal(t, 1.0, *(entries[0].Position))
}

func TestGetAuthorsForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() for book1 error")
	require.NotNil(t, book1)

	book2, err := d.CreateBook(t.Context(), BookInput{Title: "The Drawing of the Three"})
	require.NoError(t, err, "CreateBook() for book2 error")
	require.NotNil(t, book2)

	author1, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for author1 error")
	require.NotNil(t, author1)

	author2, err := d.CreateAuthor(t.Context(), "Robin Furth", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() for author2 error")
	require.NotNil(t, author2)

	require.NoError(t, d.SetBookAuthors(t.Context(), book1.ID, []string{author2.ID, author1.ID}), "SetBookAuthors() for book1 error")
	require.NoError(t, d.SetBookAuthors(t.Context(), book2.ID, []string{author1.ID}), "SetBookAuthors() for book2 error")

	got, err := d.GetAuthorsForBooks(t.Context(), []string{book1.ID, book2.ID})
	require.NoError(t, err, "GetAuthorsForBooks() error")
	require.Len(t, got, 2)
	require.Len(t, got[book1.ID], 2)
	seen := map[string]bool{}
	for _, author := range got[book1.ID] {
		seen[author.ID] = true
	}
	require.True(t, seen[author1.ID] || !seen[author2.ID])
	require.Len(t, got[book2.ID], 1)
	require.Equal(t, author1.ID, got[book2.ID][0].ID)
}

func TestGetSeriesForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() for book1 error")
	require.NotNil(t, book1)

	book2, err := d.CreateBook(t.Context(), BookInput{Title: "The Drawing of the Three"})
	require.NoError(t, err, "CreateBook() for book2 error")
	require.NotNil(t, book2)

	book3, err := d.CreateBook(t.Context(), BookInput{Title: "Wolves of the Calla"})
	require.NoError(t, err, "CreateBook() for book3 error")
	require.NotNil(t, book3)

	seriesA, err := d.CreateSeries(t.Context(), "A Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for seriesA error")
	require.NotNil(t, seriesA)

	seriesB, err := d.CreateSeries(t.Context(), "B Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for seriesB error")
	require.NotNil(t, seriesB)

	require.NoError(t, d.SetBookSeries(t.Context(), book1.ID, []BookSeriesInput{
		{SeriesID: seriesB.ID, Position: new(2.0)},
		{SeriesID: seriesA.ID, Position: new(1.0)},
	}), "SetBookSeries() for book1 error")
	require.NoError(t, d.SetBookSeries(t.Context(), book2.ID, []BookSeriesInput{
		{SeriesID: seriesA.ID, Position: nil},
	}), "SetBookSeries() for book2 error")

	got, err := d.GetSeriesForBooks(t.Context(), []string{book1.ID, book2.ID, book3.ID})
	require.NoError(t, err, "GetSeriesForBooks() error")
	require.Len(t, got, 2)

	require.Len(t, got[book1.ID], 2)
	require.Equal(t, seriesA.ID, got[book1.ID][0].Series.ID)
	require.NotNil(t, got[book1.ID][0].Position)
	require.Equal(t, 1.0, *got[book1.ID][0].Position)
	require.Equal(t, seriesB.ID, got[book1.ID][1].Series.ID)
	require.NotNil(t, got[book1.ID][1].Position)
	require.Equal(t, 2.0, *got[book1.ID][1].Position)

	require.Len(t, got[book2.ID], 1)
	require.Equal(t, seriesA.ID, got[book2.ID][0].Series.ID)
	require.Nil(t, got[book2.ID][0].Position)

	_, ok := got[book3.ID]
	require.False(t, ok)
}

func TestGetSeriesForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetSeriesForBooks(t.Context(), []string{})
	require.NoError(t, err, "GetSeriesForBooks(empty) error")
	require.Nil(t, result)
}

func TestGetSeriesForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetSeriesForBooks(t.Context(), nil)
	require.NoError(t, err, "GetSeriesForBooks(nil) error")
	require.Nil(t, result)
}

func TestDeleteBook_CascadeAuthorsAndSeries(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	a, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")
	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	require.NoError(t, d.SetBookAuthors(t.Context(), book.ID, []string{a.ID}), "SetBookAuthors() error")
	require.NoError(t, d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.1)}}), "SetBookSeries() error")

	require.NoError(t, d.DeleteBook(t.Context(), book.ID), "DeleteBook() error")

	// Author and series should still exist (only join table entries are cascaded)
	_, err = d.GetAuthor(t.Context(), a.ID)
	require.NoError(t, err)
	_, err = d.GetSeries(t.Context(), s.ID)
	require.NoError(t, err)
}

func TestCreateBookWithFile(t *testing.T) {
	d := newTestDB(t)

	b, bf, err := d.CreateBookWithFile(
		t.Context(),
		BookInput{
			Title:           "The Gunslinger",
			Description:     new("The first book of the Dark Tower series"),
			ISBN10:          new("1234567890"),
			PublicationDate: new("1982-06-10"),
			Publisher:       new("Grant"),
			Language:        new("en"),
		},
		"epub",
		"the-gunslinger.epub",
		4096,
		nil,
		"/books/the-gunslinger.epub",
	)
	require.NoError(t, err, "CreateBookWithFile() error")
	require.NotEqual(t, "", b.ID)
	require.Equal(t, "The Gunslinger", b.Title)
	require.NotNil(t, b.Description)
	require.Equal(t, "The first book of the Dark Tower series", *b.Description)
	require.NotNil(t, b.ISBN10)
	require.Equal(t, "1234567890", *b.ISBN10)
	require.NotNil(t, b.Publisher)
	require.Equal(t, "Grant", *b.Publisher)
	require.NotNil(t, b.PublicationDate)
	require.Equal(t, "1982-06-10", *b.PublicationDate)
	require.NotNil(t, b.Language)
	require.Equal(t, "en", *b.Language)

	require.Equal(t, b.ID, bf.BookID)
	require.Equal(t, "epub", bf.FileType)
	require.Equal(t, "the-gunslinger.epub", bf.FileName)
	require.Equal(t, int64(4096), bf.FileSize)
	require.Equal(t, "/books/the-gunslinger.epub", bf.FilePath)
}

func TestCreateBookWithFile_RollbackOnFileFailure(t *testing.T) {
	d := newTestDB(t)

	// Install a trigger that forces inserts into book_files to fail. This lets
	// the book insert succeed while the book_files insert fails, so we can
	// verify that the transaction is rolled back and no orphan book remains.
	_, err := d.ExecContext(t.Context(), `
		CREATE TRIGGER fail_book_files_insert
		BEFORE INSERT ON book_files
		BEGIN
			SELECT RAISE(ABORT, 'book_files insert forced failure');
		END;
	`)
	require.NoError(t, err, "failed to create trigger")

	_, _, err = d.CreateBookWithFile(
		t.Context(),
		BookInput{Title: "Orphan Book"},
		"epub",
		"orphan.epub",
		1024,
		nil,
		"/books/orphan.epub",
	)
	require.Error(t, err, "expected error from failing book_files insert")

	// Verify no book was committed
	books, err := d.ListBooks(t.Context())
	require.NoError(t, err, "ListBooks() error")
	require.Len(t, books, 0)
}

func TestDeleteLibrary_DoesNotDeleteBook(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	require.NoError(t, d.AddBookToLibrary(t.Context(), lib.ID, book.ID), "AddBookToLibrary() error")

	require.NoError(t, d.DeleteLibrary(t.Context(), lib.ID), "DeleteLibrary() error")

	// Book should still exist
	found, err := d.GetBook(t.Context(), book.ID)
	require.NoError(t, err, "book should still exist after library delete, got")
	require.Equal(t, book.ID, found.ID)
}
