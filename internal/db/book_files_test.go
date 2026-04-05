package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBookFile(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	if book == nil {
		require.Fail(t, "CreateBook() returned nil book")
	}

	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024000, new("abc123hash"), "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")
	if bf == nil {
		require.Fail(t, "CreateBookFile() returned nil book file")
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

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	created, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")

	found, err := d.GetBookFile(t.Context(), created.ID)
	require.NoError(t, err, "GetBookFile() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBookFile(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListBookFiles(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	if _, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub"); err != nil {
		require.NoError(t, err, "CreateBookFile() for epub error")
	}
	if _, err := d.CreateBookFile(t.Context(), book.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf"); err != nil {
		require.NoError(t, err, "CreateBookFile() for pdf error")
	}

	files, err := d.ListBookFiles(t.Context(), book.ID)
	require.NoError(t, err, "ListBookFiles() error")
	if len(files) != 2 {
		require.Failf(t, "failed", "ListBookFiles() returned %d, want 2", len(files))
	}
	if files[0].FileName != "gunslinger.epub" {
		t.Errorf("first file FileName = %q, want %q", files[0].FileName, "gunslinger.epub")
	}
}

func TestDeleteBookFile(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")

	err = d.DeleteBookFile(t.Context(), bf.ID)
	require.NoError(t, err, "DeleteBookFile() error")

	_, err = d.GetBookFile(t.Context(), bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBookFile(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteBook_CascadeFiles(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	if book == nil {
		require.Fail(t, "CreateBook() returned nil book")
	}
	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")
	if bf == nil {
		require.Fail(t, "CreateBookFile() returned nil book file")
	}

	require.NoError(t, d.DeleteBook(t.Context(), book.ID), "DeleteBook() error")

	_, err = d.GetBookFile(t.Context(), bf.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after book delete (cascade), got %v", err)
	}
}

func TestGetFilesForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for book1 error")
	if book1 == nil {
		require.Fail(t, "CreateBook() returned nil book1")
	}
	book2, err := d.CreateBook(t.Context(), "Wizard and Glass", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for book2 error")
	if book2 == nil {
		require.Fail(t, "CreateBook() returned nil book2")
	}

	if _, err = d.CreateBookFile(t.Context(), book1.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub"); err != nil {
		require.NoError(t, err, "CreateBookFile() for book1 epub error")
	}
	if _, err = d.CreateBookFile(t.Context(), book1.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf"); err != nil {
		require.NoError(t, err, "CreateBookFile() for book1 pdf error")
	}
	if _, err = d.CreateBookFile(t.Context(), book2.ID, "epub", "wizard-and-glass.epub", 4096, nil, "/books/wizard-and-glass.epub"); err != nil {
		require.NoError(t, err, "CreateBookFile() for book2 epub error")
	}

	got, err := d.GetFilesForBooks(t.Context(), []string{book1.ID, book2.ID})
	require.NoError(t, err, "GetFilesForBooks() error")
	if len(got) != 2 {
		require.Failf(t, "failed", "GetFilesForBooks() returned %d book entries, want 2", len(got))
	}
	if len(got[book1.ID]) != 2 {
		require.Failf(t, "failed", "GetFilesForBooks()[book1] returned %d files, want 2", len(got[book1.ID]))
	}
	if got[book1.ID][0].FileName != "gunslinger.epub" {
		t.Errorf("first file for book1 = %q, want %q", got[book1.ID][0].FileName, "gunslinger.epub")
	}
	if len(got[book2.ID]) != 1 || got[book2.ID][0].FileName != "wizard-and-glass.epub" {
		t.Errorf("files for book2 = %+v, want wizard-and-glass.epub", got[book2.ID])
	}
}
