package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateBookFile(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() error: %v", err)
	}
	if book == nil {
		t.Fatalf("CreateBook() returned nil book")
	}

	bf, err := d.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024000, strPtr("abc123hash"), "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("CreateBookFile() error: %v", err)
	}
	if bf == nil {
		t.Fatalf("CreateBookFile() returned nil book file")
	}
	if bf.ID == "" {
		t.Error("CreateBookFile() returned empty ID")
	}
	if bf.BookID != book.ID {
		t.Errorf("BookID = %q, want %q", bf.BookID, book.ID)
	}
	if bf.FileType != "epub" {
		t.Errorf("FileType = %q, want %q", bf.FileType, "epub")
	}
	if bf.FileName != "gunslinger.epub" {
		t.Errorf("FileName = %q, want %q", bf.FileName, "gunslinger.epub")
	}
	if bf.FileSize != 1024000 {
		t.Errorf("FileSize = %d, want %d", bf.FileSize, 1024000)
	}
	if bf.FileHash == nil || *bf.FileHash != "abc123hash" {
		t.Errorf("FileHash = %v, want %q", bf.FileHash, "abc123hash")
	}
	if bf.FilePath != "/books/gunslinger.epub" {
		t.Errorf("FilePath = %q, want %q", bf.FilePath, "/books/gunslinger.epub")
	}
}

func TestGetBookFile(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	created, _ := d.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	found, err := d.GetBookFile(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetBookFile() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBookFile(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListBookFiles(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _ = d.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	_, _ = d.CreateBookFile(context.Background(), book.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf")

	files, err := d.ListBookFiles(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("ListBookFiles() error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("ListBookFiles() returned %d, want 2", len(files))
	}
	if files[0].FileName != "gunslinger.epub" {
		t.Errorf("first file FileName = %q, want %q", files[0].FileName, "gunslinger.epub")
	}
}

func TestDeleteBookFile(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := d.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	err := d.DeleteBookFile(context.Background(), bf.ID)
	if err != nil {
		t.Fatalf("DeleteBookFile() error: %v", err)
	}

	_, err = d.GetBookFile(context.Background(), bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBookFile(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteBook_CascadeFiles(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() error: %v", err)
	}
	if book == nil {
		t.Fatalf("CreateBook() returned nil book")
	}
	bf, err := d.CreateBookFile(context.Background(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("CreateBookFile() error: %v", err)
	}
	if bf == nil {
		t.Fatalf("CreateBookFile() returned nil book file")
	}

	if err := d.DeleteBook(context.Background(), book.ID); err != nil {
		t.Fatalf("DeleteBook() error: %v", err)
	}

	_, err = d.GetBookFile(context.Background(), bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after book delete (cascade), got %v", err)
	}
}

func TestGetFilesForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(context.Background(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() for book1 error: %v", err)
	}
	if book1 == nil {
		t.Fatalf("CreateBook() returned nil book1")
	}
	book2, err := d.CreateBook(context.Background(), "Wizard and Glass", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook() for book2 error: %v", err)
	}
	if book2 == nil {
		t.Fatalf("CreateBook() returned nil book2")
	}

	if _, err = d.CreateBookFile(context.Background(), book1.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub"); err != nil {
		t.Fatalf("CreateBookFile() for book1 epub error: %v", err)
	}
	if _, err = d.CreateBookFile(context.Background(), book1.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf"); err != nil {
		t.Fatalf("CreateBookFile() for book1 pdf error: %v", err)
	}
	if _, err = d.CreateBookFile(context.Background(), book2.ID, "epub", "wizard-and-glass.epub", 4096, nil, "/books/wizard-and-glass.epub"); err != nil {
		t.Fatalf("CreateBookFile() for book2 epub error: %v", err)
	}

	got, err := d.GetFilesForBooks(context.Background(), []string{book1.ID, book2.ID})
	if err != nil {
		t.Fatalf("GetFilesForBooks() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("GetFilesForBooks() returned %d book entries, want 2", len(got))
	}
	if len(got[book1.ID]) != 2 {
		t.Fatalf("GetFilesForBooks()[book1] returned %d files, want 2", len(got[book1.ID]))
	}
	if got[book1.ID][0].FileName != "gunslinger.epub" {
		t.Errorf("first file for book1 = %q, want %q", got[book1.ID][0].FileName, "gunslinger.epub")
	}
	if len(got[book2.ID]) != 1 || got[book2.ID][0].FileName != "wizard-and-glass.epub" {
		t.Errorf("files for book2 = %+v, want wizard-and-glass.epub", got[book2.ID])
	}
}
