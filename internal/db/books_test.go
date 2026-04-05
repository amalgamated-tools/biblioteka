package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBook(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), "The Gunslinger", new("The first book"), nil, new("1234567890"), nil, nil, nil, nil, new("1982-06-10"), new("Grant"), new("en"), nil)
	require.NoError(t, err, "CreateBook() error")
	if b.ID == "" {
		t.Error("CreateBook() returned empty ID")
	}
	if b.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", b.Title, "The Gunslinger")
	}
	if b.ISBN10 == nil || *b.ISBN10 != "1234567890" {
		t.Errorf("ISBN10 = %v, want %q", b.ISBN10, "1234567890")
	}
	if b.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestGetBook(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")

	found, err := d.GetBook(t.Context(), created.ID)
	require.NoError(t, err, "GetBook() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", found.Title, "The Gunslinger")
	}
}

func TestGetBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBook(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListBooks(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook() for A Game of Thrones error")
	}
	if _, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook() for The Gunslinger error")
	}

	books, err := d.ListBooks(t.Context())
	require.NoError(t, err, "ListBooks() error")
	require.Len(t, books, 2)
	if books[0].Title != "A Game of Thrones" {
		t.Errorf("first book Title = %q, want %q", books[0].Title, "A Game of Thrones")
	}
}

func TestUpdateBook(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateBook(t.Context(), "Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")

	updated, err := d.UpdateBook(t.Context(), created.ID, "The Gunslinger", new("Revised edition"), nil, nil, nil, nil, nil, nil, nil, nil, new("en"), nil)
	require.NoError(t, err, "UpdateBook() error")
	if updated.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", updated.Title, "The Gunslinger")
	}
}

func TestDeleteBook(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")

	err = d.DeleteBook(t.Context(), b.ID)
	require.NoError(t, err, "DeleteBook() error")

	_, err = d.GetBook(t.Context(), b.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBook(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestAddBookToLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")

	err = d.AddBookToLibrary(t.Context(), lib.ID, book.ID)
	require.NoError(t, err, "AddBookToLibrary() error")

	books, err := d.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "ListBooksByLibrary() error")
	require.Len(t, books, 1)
	if books[0].ID != book.ID {
		t.Errorf("book ID = %q, want %q", books[0].ID, book.ID)
	}
}

func TestListBooksByLibraryPaginated(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	b1, err := d.CreateBook(t.Context(), "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for Alpha error")
	b2, err := d.CreateBook(t.Context(), "Beta", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for Beta error")
	b3, err := d.CreateBook(t.Context(), "Gamma", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for Gamma error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b1.ID)
	require.NoError(t, err, "AddBookToLibrary() for Alpha error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b2.ID)
	require.NoError(t, err, "AddBookToLibrary() for Beta error")
	err = d.AddBookToLibrary(t.Context(), lib.ID, b3.ID)
	require.NoError(t, err, "AddBookToLibrary() for Gamma error")

	books, total, err := d.ListBooksByLibraryPaginated(t.Context(), lib.ID, 2, 0)
	require.NoError(t, err, "ListBooksByLibraryPaginated() error")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(books))
	}
	if books[0].Title != "Alpha" {
		t.Errorf("first book = %q, want Alpha", books[0].Title)
	}

	books2, total2, err := d.ListBooksByLibraryPaginated(t.Context(), lib.ID, 2, 2)
	require.NoError(t, err, "ListBooksByLibraryPaginated() page 2 error")
	if total2 != 3 {
		t.Errorf("total page 2 = %d, want 3", total2)
	}
	if len(books2) != 1 {
		t.Errorf("len(books) page 2 = %d, want 1", len(books2))
	}
}

func TestRemoveBookFromLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	require.NoError(t, d.AddBookToLibrary(t.Context(), lib.ID, book.ID), "AddBookToLibrary() error")

	err = d.RemoveBookFromLibrary(t.Context(), lib.ID, book.ID)
	require.NoError(t, err, "RemoveBookFromLibrary() error")

	books, err := d.ListBooksByLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "ListBooksByLibrary() error")
	if len(books) != 0 {
		t.Errorf("ListBooksByLibrary() returned %d, want 0", len(books))
	}
}

func TestSetBookAuthors(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	book, err := d.CreateBook(t.Context(), "The Talisman", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if authors[0].ID != a2.ID {
		t.Errorf("author ID = %q, want %q", authors[0].ID, a2.ID)
	}
}

func TestSetBookSeries(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if entries[0].Series.ID != s.ID {
		t.Errorf("series ID = %q, want %q", entries[0].Series.ID, s.ID)
	}
	if entries[0].Position == nil || *(entries[0].Position) != 1.0 {
		t.Errorf("Position = %v, want 1.0", entries[0].Position)
	}
}

func TestGetAuthorsForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for book1 error")
	require.NotNil(t, book1)

	book2, err := d.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if !seen[author1.ID] || !seen[author2.ID] {
		t.Errorf("authors for book1 = %+v, want IDs %q and %q", got[book1.ID], author1.ID, author2.ID)
	}
	if len(got[book2.ID]) != 1 || got[book2.ID][0].ID != author1.ID {
		t.Errorf("authors for book2 = %+v, want [%s]", got[book2.ID], author1.ID)
	}
}

func TestDeleteBook_CascadeAuthorsAndSeries(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	if err != nil {
		t.Errorf("author should still exist after book delete, got: %v", err)
	}
	_, err = d.GetSeries(t.Context(), s.ID)
	if err != nil {
		t.Errorf("series should still exist after book delete, got: %v", err)
	}
}

func TestCreateBookWithFile(t *testing.T) {
	d := newTestDB(t)

	b, bf, err := d.CreateBookWithFile(
		t.Context(),
		"The Gunslinger",
		new("The first book of the Dark Tower series"),
		nil,
		new("1234567890"),
		nil,
		nil, nil, nil,
		new("1982-06-10"),
		new("Grant"),
		new("en"),
		nil,
		"epub",
		"the-gunslinger.epub",
		4096,
		nil,
		"/books/the-gunslinger.epub",
	)
	require.NoError(t, err, "CreateBookWithFile() error")
	if b.ID == "" {
		t.Error("book ID is empty")
	}
	if b.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", b.Title, "The Gunslinger")
	}
	if b.Description == nil || *b.Description != "The first book of the Dark Tower series" {
		t.Errorf("Description = %v, want %q", b.Description, "The first book of the Dark Tower series")
	}
	if b.ISBN10 == nil || *b.ISBN10 != "1234567890" {
		t.Errorf("ISBN10 = %v, want %q", b.ISBN10, "1234567890")
	}
	if b.Publisher == nil || *b.Publisher != "Grant" {
		t.Errorf("Publisher = %v, want %q", b.Publisher, "Grant")
	}
	if b.PublicationDate == nil || *b.PublicationDate != "1982-06-10" {
		t.Errorf("PublicationDate = %v, want %q", b.PublicationDate, "1982-06-10")
	}
	if b.Language == nil || *b.Language != "en" {
		t.Errorf("Language = %v, want %q", b.Language, "en")
	}

	if bf.BookID != b.ID {
		t.Errorf("BookFile.BookID = %q, want %q", bf.BookID, b.ID)
	}
	if bf.FileType != "epub" {
		t.Errorf("FileType = %q, want %q", bf.FileType, "epub")
	}
	if bf.FileName != "the-gunslinger.epub" {
		t.Errorf("FileName = %q, want %q", bf.FileName, "the-gunslinger.epub")
	}
	if bf.FileSize != 4096 {
		t.Errorf("FileSize = %d, want 4096", bf.FileSize)
	}
	if bf.FilePath != "/books/the-gunslinger.epub" {
		t.Errorf("FilePath = %q, want %q", bf.FilePath, "/books/the-gunslinger.epub")
	}
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
		"Orphan Book",
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
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
	if len(books) != 0 {
		t.Errorf("expected 0 books after rollback, got %d", len(books))
	}
}

func TestDeleteLibrary_DoesNotDeleteBook(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `[]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	require.NoError(t, d.AddBookToLibrary(t.Context(), lib.ID, book.ID), "AddBookToLibrary() error")

	require.NoError(t, d.DeleteLibrary(t.Context(), lib.ID), "DeleteLibrary() error")

	// Book should still exist
	found, err := d.GetBook(t.Context(), book.ID)
	require.NoError(t, err, "book should still exist after library delete, got")
	if found.ID != book.ID {
		t.Errorf("book ID = %q, want %q", found.ID, book.ID)
	}
}
