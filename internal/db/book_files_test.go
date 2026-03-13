package db

import (
	"database/sql"
	"testing"
)

func TestCreateBookFile(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	bf, err := d.CreateBookFile(book.ID, "epub", "gunslinger.epub", 1024000, strPtr("abc123hash"), "/books/gunslinger.epub")
	if err != nil {
		t.Fatalf("CreateBookFile() error: %v", err)
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

	book, _ := d.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	created, _ := d.CreateBookFile(book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	found, err := d.GetBookFile(created.ID)
	if err != nil {
		t.Fatalf("GetBookFile() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBookFile("nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListBookFiles(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _ = d.CreateBookFile(book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	_, _ = d.CreateBookFile(book.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf")

	files, err := d.ListBookFiles(book.ID)
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

	book, _ := d.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := d.CreateBookFile(book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	err := d.DeleteBookFile(bf.ID)
	if err != nil {
		t.Fatalf("DeleteBookFile() error: %v", err)
	}

	_, err = d.GetBookFile(bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBookFile("nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteBook_CascadeFiles(t *testing.T) {
	d := newTestDB(t)

	book, _ := d.CreateBook("The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	bf, _ := d.CreateBookFile(book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")

	_ = d.DeleteBook(book.ID)

	_, err := d.GetBookFile(bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after book delete (cascade), got %v", err)
	}
}
