package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests cover edge cases in book_relations.go that are not already
// exercised by books_test.go (which covers the basic happy-path scenarios).

// ---- GetBookAuthors ----

func TestGetBookAuthors_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Authorless Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "GetBookAuthors() error")
	require.Len(t, authors, 0)
}

// ---- SetBookAuthors ----

func TestSetBookAuthors_ClearAll(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	author, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	require.NoError(t, d.SetBookAuthors(t.Context(), book.ID, []string{author.ID}), "SetBookAuthors(set)")

	// Clear by passing an empty slice.
	require.NoError(t, d.SetBookAuthors(t.Context(), book.ID, []string{}), "SetBookAuthors(clear)")

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "GetBookAuthors() error")
	require.Len(t, authors, 0)
}

func TestSetBookAuthors_DeduplicatesIDs(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Shared Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	author, err := d.CreateAuthor(t.Context(), "Author A", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	// Pass the same author ID three times.
	require.NoError(t, d.SetBookAuthors(t.Context(), book.ID, []string{author.ID, author.ID, author.ID}), "SetBookAuthors() with duplicates error")

	authors, err := d.GetBookAuthors(t.Context(), book.ID)
	require.NoError(t, err, "GetBookAuthors() error")
	require.Len(t, authors, 1)
}

// ---- GetBookSeries ----

func TestGetBookSeries_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Standalone Novel", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")

	entries, err := d.GetBookSeries(t.Context(), book.ID)
	require.NoError(t, err, "GetBookSeries() error")
	require.Len(t, entries, 0)
}

// ---- SetBookSeries ----

func TestSetBookSeries_ClearAll(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	s, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	require.NoError(t, d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{{SeriesID: s.ID}}), "SetBookSeries(set)")

	// Clear by passing empty slice.
	require.NoError(t, d.SetBookSeries(t.Context(), book.ID, []BookSeriesInput{}), "SetBookSeries(clear)")

	entries, err := d.GetBookSeries(t.Context(), book.ID)
	require.NoError(t, err, "GetBookSeries() error")
	require.Len(t, entries, 0)
}

func TestSetBookSeries_DeduplicatesLastPositionWins(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Crossover Novel", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	s, err := d.CreateSeries(t.Context(), "A Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	// Pass the same series ID twice with different positions; last position wins.
	entries := []BookSeriesInput{
		{SeriesID: s.ID, Position: new(1.0)},
		{SeriesID: s.ID, Position: new(99.0)},
	}
	require.NoError(t, d.SetBookSeries(t.Context(), book.ID, entries), "SetBookSeries() with duplicates error")

	got, err := d.GetBookSeries(t.Context(), book.ID)
	require.NoError(t, err, "GetBookSeries() error")
	require.Len(t, got, 1)
	// The implementation processes entries in reverse order so the last
	// element (position 99) is the one that survives deduplication.
	require.NotNil(t, got[0].Position)
	require.Equal(t, 99.0, *got[0].Position)
}

// ---- GetAuthorsForBooks ----

func TestGetAuthorsForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetAuthorsForBooks(t.Context(), []string{})
	require.NoError(t, err, "GetAuthorsForBooks(empty) error")
	require.Nil(t, result)
}

func TestGetAuthorsForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetAuthorsForBooks(t.Context(), nil)
	require.NoError(t, err, "GetAuthorsForBooks(nil) error")
	require.Nil(t, result)
}

func TestGetAuthorsForBooks_BookWithNoAuthors(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), "Orphan Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")

	result, err := d.GetAuthorsForBooks(t.Context(), []string{book.ID})
	require.NoError(t, err, "GetAuthorsForBooks() error")
	// Map is returned but the key for this book should be absent.
	if authors, ok := result[book.ID]; ok {
		require.Empty(t, authors, "expected no authors for book")
	}
}
