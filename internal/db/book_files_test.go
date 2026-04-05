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
	require.NotNil(t, book)

	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024000, new("abc123hash"), "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")
	require.NotNil(t, bf)
	require.NotEqual(t, "", bf.ID)
	require.Equal(t, book.ID, bf.BookID)
	require.Equal(t, "epub", bf.FileType)
	require.Equal(t, "gunslinger.epub", bf.FileName)
	require.Equal(t, int64(1024000), bf.FileSize)
	require.NotNil(t, bf.FileHash)
	require.Equal(t, "abc123hash", *bf.FileHash)
	require.Equal(t, "/books/gunslinger.epub", bf.FilePath)
}

func TestGetBookFile(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	created, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")

	found, err := d.GetBookFile(t.Context(), created.ID)
	require.NoError(t, err, "GetBookFile() error")
	require.Equal(t, created.ID, found.ID)
}

func TestGetBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetBookFile(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListBookFiles(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	_, err = d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() for epub error")
	_, err = d.CreateBookFile(t.Context(), book.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf")
	require.NoError(t, err, "CreateBookFile() for pdf error")

	files, err := d.ListBookFiles(t.Context(), book.ID)
	require.NoError(t, err, "ListBookFiles() error")
	require.Len(t, files, 2)
	require.Equal(t, "gunslinger.epub", files[0].FileName)
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
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteBookFile_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteBookFile(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteBook_CascadeFiles(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() error")
	require.NotNil(t, book)
	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")
	require.NotNil(t, bf)

	require.NoError(t, d.DeleteBook(t.Context(), book.ID), "DeleteBook() error")

	_, err = d.GetBookFile(t.Context(), bf.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetFilesForBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for book1 error")
	require.NotNil(t, book1)
	book2, err := d.CreateBook(t.Context(), "Wizard and Glass", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook() for book2 error")
	require.NotNil(t, book2)

	_, err = d.CreateBookFile(t.Context(), book1.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() for book1 epub error")
	_, err = d.CreateBookFile(t.Context(), book1.ID, "pdf", "gunslinger.pdf", 2048, nil, "/books/gunslinger.pdf")
	require.NoError(t, err, "CreateBookFile() for book1 pdf error")
	_, err = d.CreateBookFile(t.Context(), book2.ID, "epub", "wizard-and-glass.epub", 4096, nil, "/books/wizard-and-glass.epub")
	require.NoError(t, err, "CreateBookFile() for book2 epub error")

	got, err := d.GetFilesForBooks(t.Context(), []string{book1.ID, book2.ID})
	require.NoError(t, err, "GetFilesForBooks() error")
	require.Len(t, got, 2)
	require.Len(t, got[book1.ID], 2)
	require.Equal(t, "gunslinger.epub", got[book1.ID][0].FileName)
	require.Len(t, got[book2.ID], 1)
	require.Equal(t, "wizard-and-glass.epub", got[book2.ID][0].FileName)
}
