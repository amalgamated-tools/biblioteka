package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBookFile(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
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

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
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

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
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

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
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

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
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

	book1, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() for book1 error")
	require.NotNil(t, book1)
	book2, err := d.CreateBook(t.Context(), BookInput{Title: "Wizard and Glass"})
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

func TestGetFilesForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetFilesForBooks(t.Context(), []string{})
	require.NoError(t, err, "GetFilesForBooks(empty) error")
	require.Nil(t, result)
}

func TestGetFilesForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetFilesForBooks(t.Context(), nil)
	require.NoError(t, err, "GetFilesForBooks(nil) error")
	require.Nil(t, result)
}

func TestIncrementBookFileDownloadCount(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook() error")
	bf, err := d.CreateBookFile(t.Context(), book.ID, "epub", "gunslinger.epub", 1024, nil, "/books/gunslinger.epub")
	require.NoError(t, err, "CreateBookFile() error")
	require.Equal(t, int64(0), bf.DownloadCount)

	// Increment once.
	err = d.IncrementBookFileDownloadCount(t.Context(), bf.ID)
	require.NoError(t, err, "IncrementBookFileDownloadCount() first call error")

	got, err := d.GetBookFile(t.Context(), bf.ID)
	require.NoError(t, err, "GetBookFile() after first increment error")
	require.Equal(t, int64(1), got.DownloadCount)

	// Increment again.
	err = d.IncrementBookFileDownloadCount(t.Context(), bf.ID)
	require.NoError(t, err, "IncrementBookFileDownloadCount() second call error")

	got, err = d.GetBookFile(t.Context(), bf.ID)
	require.NoError(t, err, "GetBookFile() after second increment error")
	require.Equal(t, int64(2), got.DownloadCount)
}

func TestIncrementBookFileDownloadCount_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.IncrementBookFileDownloadCount(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
