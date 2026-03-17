package db

import (
	"context"
	"database/sql"
	"testing"
)

func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }

func TestCreateBook(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(context.Background(), "The Gunslinger", strPtr("The first book"), nil, strPtr("1234567890"), nil, nil, nil, nil, strPtr("1982-06-10"), strPtr("Grant"), strPtr("en"), intPtr(224), nil)
	if err != nil {
		t.Fatalf("CreateBook() error: %v", err)
	}
	if b.ID == "" {
		t.Error("CreateBook() returned empty ID")
	}
	if b.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", b.Title, "The Gunslinger")
	}
	if b.ISBN10 == nil || *b.ISBN10 != "1234567890" {
		t.Errorf("ISBN10 = %v, want %q", b.ISBN10, "1234567890")
	}
	if b.NumPages == nil || *b.NumPages != 224 {
		t.Errorf("NumPages = %v, want 224", b.NumPages)
	}
	if b.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestGetBook(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	found, err := d.GetBook(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetBook() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", found.Title, "The Gunslinger")
	}
}

func TestGetBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBook(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListBooks(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateBook(context.Background(), "A Game of Thrones", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _ = d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	books, err := d.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("ListBooks() error: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("ListBooks() returned %d, want 2", len(books))
	}
	if books[0].Title != "A Game of Thrones" {
		t.Errorf("first book Title = %q, want %q", books[0].Title, "A Game of Thrones")
	}
}

func TestUpdateBook(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateBook(context.Background(), "Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	updated, err := d.UpdateBook(context.Background(), created.ID, "The Gunslinger", strPtr("Revised edition"), nil, nil, nil, nil, nil, nil, nil, nil, strPtr("en"), intPtr(300), nil)
	if err != nil {
		t.Fatalf("UpdateBook() error: %v", err)
	}
	if updated.Title != "The Gunslinger" {
		t.Errorf("Title = %q, want %q", updated.Title, "The Gunslinger")
	}
	if updated.NumPages == nil || *updated.NumPages != 300 {
		t.Errorf("NumPages = %v, want 300", updated.NumPages)
	}
}

func TestDeleteBook(t *testing.T) {
	d := newTestDB(t)

	b, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := d.DeleteBook(context.Background(), b.ID)
	if err != nil {
		t.Fatalf("DeleteBook() error: %v", err)
	}

	_, err = d.GetBook(context.Background(), b.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteBook_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBook(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestAddBookToLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(context.Background(), "Fiction", `[]`, "book_per_folder", false)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}
	book, err := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() error: %v", err)
	}

	err = d.AddBookToLibrary(context.Background(), lib.ID, book.ID)
	if err != nil {
		t.Fatalf("AddBookToLibrary() error: %v", err)
	}

	books, err := d.ListBooksByLibrary(context.Background(), lib.ID)
	if err != nil {
		t.Fatalf("ListBooksByLibrary() error: %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("ListBooksByLibrary() returned %d, want 1", len(books))
	}
	if books[0].ID != book.ID {
		t.Errorf("book ID = %q, want %q", books[0].ID, book.ID)
	}
}

func TestListBooksByLibraryPaginated(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(context.Background(), "Fiction", `[]`, "book_per_folder", false)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}
	b1, err := d.CreateBook(context.Background(), "Alpha", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() for Alpha error: %v", err)
	}
	b2, err := d.CreateBook(context.Background(), "Beta", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() for Beta error: %v", err)
	}
	b3, err := d.CreateBook(context.Background(), "Gamma", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() for Gamma error: %v", err)
	}
	err = d.AddBookToLibrary(context.Background(), lib.ID, b1.ID)
	if err != nil {
		t.Fatalf("AddBookToLibrary() for Alpha error: %v", err)
	}
	err = d.AddBookToLibrary(context.Background(), lib.ID, b2.ID)
	if err != nil {
		t.Fatalf("AddBookToLibrary() for Beta error: %v", err)
	}
	err = d.AddBookToLibrary(context.Background(), lib.ID, b3.ID)
	if err != nil {
		t.Fatalf("AddBookToLibrary() for Gamma error: %v", err)
	}

	books, total, err := d.ListBooksByLibraryPaginated(context.Background(), lib.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListBooksByLibraryPaginated() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(books))
	}
	if books[0].Title != "Alpha" {
		t.Errorf("first book = %q, want Alpha", books[0].Title)
	}

	books2, total2, err := d.ListBooksByLibraryPaginated(context.Background(), lib.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListBooksByLibraryPaginated() page 2 error: %v", err)
	}
	if total2 != 3 {
		t.Errorf("total page 2 = %d, want 3", total2)
	}
	if len(books2) != 1 {
		t.Errorf("len(books) page 2 = %d, want 1", len(books2))
	}
}

func TestRemoveBookFromLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, _ := d.CreateLibrary(context.Background(), "Fiction", `[]`, "book_per_folder", false)
	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_ = d.AddBookToLibrary(context.Background(), lib.ID, book.ID)

	err := d.RemoveBookFromLibrary(context.Background(), lib.ID, book.ID)
	if err != nil {
		t.Fatalf("RemoveBookFromLibrary() error: %v", err)
	}

	books, _ := d.ListBooksByLibrary(context.Background(), lib.ID)
	if len(books) != 0 {
		t.Errorf("ListBooksByLibrary() returned %d, want 0", len(books))
	}
}

func TestSetBookAuthors(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	a1, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	a2, _ := d.CreateAuthor(context.Background(), "Peter Straub", nil, nil, nil, nil)

	err := d.SetBookAuthors(context.Background(), book.ID, []string{a1.ID, a2.ID})
	if err != nil {
		t.Fatalf("SetBookAuthors() error: %v", err)
	}

	authors, err := d.GetBookAuthors(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("GetBookAuthors() error: %v", err)
	}
	if len(authors) != 2 {
		t.Fatalf("GetBookAuthors() returned %d, want 2", len(authors))
	}
}

func TestSetBookAuthors_Replace(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Talisman", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	a1, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	a2, _ := d.CreateAuthor(context.Background(), "Peter Straub", nil, nil, nil, nil)

	_ = d.SetBookAuthors(context.Background(), book.ID, []string{a1.ID})

	err := d.SetBookAuthors(context.Background(), book.ID, []string{a2.ID})
	if err != nil {
		t.Fatalf("SetBookAuthors() replace error: %v", err)
	}

	authors, _ := d.GetBookAuthors(context.Background(), book.ID)
	if len(authors) != 1 {
		t.Fatalf("GetBookAuthors() returned %d, want 1", len(authors))
	}
	if authors[0].ID != a2.ID {
		t.Errorf("author ID = %q, want %q", authors[0].ID, a2.ID)
	}
}

func TestSetBookSeries(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	err := d.SetBookSeries(context.Background(), book.ID, []BookSeriesInput{{SeriesID: s.ID, Position: floatPtr(1)}})
	if err != nil {
		t.Fatalf("SetBookSeries() error: %v", err)
	}

	entries, err := d.GetBookSeries(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("GetBookSeries() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("GetBookSeries() returned %d, want 1", len(entries))
	}
	if entries[0].Series.ID != s.ID {
		t.Errorf("series ID = %q, want %q", entries[0].Series.ID, s.ID)
	}
	if entries[0].Position == nil || *entries[0].Position != 1 {
		t.Errorf("Position = %v, want 1", entries[0].Position)
	}
}

func TestGetAuthorsForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	book2, _ := d.CreateBook(context.Background(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	author1, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	author2, _ := d.CreateAuthor(context.Background(), "Robin Furth", nil, nil, nil, nil)

	if err := d.SetBookAuthors(context.Background(), book1.ID, []string{author2.ID, author1.ID}); err != nil {
		t.Fatalf("SetBookAuthors() for book1 error: %v", err)
	}
	if err := d.SetBookAuthors(context.Background(), book2.ID, []string{author1.ID}); err != nil {
		t.Fatalf("SetBookAuthors() for book2 error: %v", err)
	}

	got, err := d.GetAuthorsForBooks(context.Background(), []string{book1.ID, book2.ID})
	if err != nil {
		t.Fatalf("GetAuthorsForBooks() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetAuthorsForBooks() returned %d book entries, want 2", len(got))
	}
	if len(got[book1.ID]) != 2 {
		t.Fatalf("GetAuthorsForBooks()[book1] returned %d authors, want 2", len(got[book1.ID]))
	}
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

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	a, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	_ = d.SetBookAuthors(context.Background(), book.ID, []string{a.ID})
	_ = d.SetBookSeries(context.Background(), book.ID, []BookSeriesInput{{SeriesID: s.ID, Position: floatPtr(1)}})

	_ = d.DeleteBook(context.Background(), book.ID)

	// Author and series should still exist (only join table entries are cascaded)
	_, err := d.GetAuthor(context.Background(), a.ID)
	if err != nil {
		t.Errorf("author should still exist after book delete, got: %v", err)
	}
	_, err = d.GetSeries(context.Background(), s.ID)
	if err != nil {
		t.Errorf("series should still exist after book delete, got: %v", err)
	}
}

func TestCreateBookWithFile(t *testing.T) {
	d := newTestDB(t)

	b, bf, err := d.CreateBookWithFile(
		context.Background(),
		"The Gunslinger",
		strPtr("The first book of the Dark Tower series"),
		nil,
		strPtr("1234567890"),
		nil,
		nil, nil, nil,
		strPtr("1982-06-10"),
		strPtr("Grant"),
		strPtr("en"),
		intPtr(224),
		nil,
		"epub",
		"the-gunslinger.epub",
		4096,
		nil,
		"/books/the-gunslinger.epub",
	)
	if err != nil {
		t.Fatalf("CreateBookWithFile() error: %v", err)
	}
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
	if b.NumPages == nil || *b.NumPages != 224 {
		t.Errorf("NumPages = %v, want 224", b.NumPages)
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
	_, err := d.ExecContext(context.Background(), `
		CREATE TRIGGER fail_book_files_insert
		BEFORE INSERT ON book_files
		BEGIN
			SELECT RAISE(ABORT, 'book_files insert forced failure');
		END;
	`)
	if err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	_, _, err = d.CreateBookWithFile(
		context.Background(),
		"Orphan Book",
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		"epub",
		"orphan.epub",
		1024,
		nil,
		"/books/orphan.epub",
	)
	if err == nil {
		t.Fatal("expected error from failing book_files insert")
	}

	// Verify no book was committed
	books, err := d.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("ListBooks() error: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books after rollback, got %d", len(books))
	}
}

func TestDeleteLibrary_DoesNotDeleteBook(t *testing.T) {
	d := newTestDB(t)

	lib, _ := d.CreateLibrary(context.Background(), "Fiction", `[]`, "book_per_folder", false)
	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_ = d.AddBookToLibrary(context.Background(), lib.ID, book.ID)

	_ = d.DeleteLibrary(context.Background(), lib.ID)

	// Book should still exist
	found, err := d.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("book should still exist after library delete, got: %v", err)
	}
	if found.ID != book.ID {
		t.Errorf("book ID = %q, want %q", found.ID, book.ID)
	}
}
