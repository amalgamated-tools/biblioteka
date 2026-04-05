package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- ListBooksBySeries ----

func TestListBooksBySeries_Empty(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Empty Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	require.NoError(t, err, "ListBooksBySeries() error")
	require.Len(t, books, 0)
}

func TestListBooksBySeries_OrderedByPosition(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	b1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b1)")
	b2, err := d.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b2)")
	b3, err := d.CreateBook(t.Context(), "The Waste Lands", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b3)")

	// Assign series entries out of order.
	require.NoError(t, d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}), "SetBookSeries(b1)")
	require.NoError(t, d.SetBookSeries(t.Context(), b3.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(3.0)}}), "SetBookSeries(b3)")
	require.NoError(t, d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(2.0)}}), "SetBookSeries(b2)")

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	require.NoError(t, err, "ListBooksBySeries() error")
	require.Len(t, books, 3, "len(books)")
	require.Equal(t, b1.ID, books[0].ID)
	require.Equal(t, b2.ID, books[1].ID)
	require.Equal(t, b3.ID, books[2].ID)
}

// TestListBooksBySeries_NullPositionSorting verifies that books with NULL
// positions and books with explicit positions are all returned. The exact
// NULL ordering differs by dialect (SQLite sorts NULLs first; Postgres
// uses NULLS LAST), so this test only checks that both books are present.
func TestListBooksBySeries_NullPositionSorting(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Nullable Positions", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	b1, err := d.CreateBook(t.Context(), "Positioned", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(positioned)")
	b2, err := d.CreateBook(t.Context(), "Unpositioned", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(unpositioned)")

	require.NoError(t, d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: nil}}), "SetBookSeries(b2 nil position)")
	require.NoError(t, d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}), "SetBookSeries(b1 pos 1)")

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	require.NoError(t, err, "ListBooksBySeries() error")
	require.Len(t, books, 2, "len(books)")
	// Both books must appear; collect the IDs to verify without relying on
	// dialect-specific NULL ordering.
	ids := map[string]bool{books[0].ID: true, books[1].ID: true}
	require.True(t, ids[b1.ID])
	require.True(t, ids[b2.ID])
}

func TestListBooksBySeriesPaginated(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Big Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")

	for i, title := range []string{"Book One", "Book Two", "Book Three", "Book Four"} {
		b, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
		pos := float64(i + 1)
		require.NoError(t, d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: &pos}}), "SetBookSeries(%q)", title)
	}

	page1, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 2, 0)
	require.NoError(t, err, "ListBooksBySeriesPaginated(page1) error")
	require.Equal(t, 4, total)
	require.Len(t, page1, 2, "len(page1)")
	require.Equal(t, "Book One", page1[0].Title)
}

func TestListBooksBySeriesPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Small Series", nil, nil, nil)
	require.NoError(t, err, "CreateSeries()")
	b, err := d.CreateBook(t.Context(), "One Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	require.NoError(t, d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}), "SetBookSeries()")

	books, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 10, 50)
	require.NoError(t, err, "ListBooksBySeriesPaginated(offset=50) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 0)
}
