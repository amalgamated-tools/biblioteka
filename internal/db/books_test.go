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

	lib, _ := d.CreateLibrary(context.Background(), "Fiction", `[]`, "book_per_folder", false)
	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := d.AddBookToLibrary(context.Background(), lib.ID, book.ID)
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
