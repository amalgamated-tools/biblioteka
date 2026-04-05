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
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
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
	if len(books) != 3 {
		t.Fatalf("len(books) = %d, want 3", len(books))
	}
	if books[0].ID != b1.ID {
		t.Errorf("books[0] = %q, want book 1 (%q)", books[0].Title, b1.Title)
	}
	if books[1].ID != b2.ID {
		t.Errorf("books[1] = %q, want book 2 (%q)", books[1].Title, b2.Title)
	}
	if books[2].ID != b3.ID {
		t.Errorf("books[2] = %q, want book 3 (%q)", books[2].Title, b3.Title)
	}
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
	if len(books) != 2 {
		t.Fatalf("len(books) = %d, want 2", len(books))
	}
	// Both books must appear; collect the IDs to verify without relying on
	// dialect-specific NULL ordering.
	ids := map[string]bool{books[0].ID: true, books[1].ID: true}
	if !ids[b1.ID] {
		t.Errorf("positioned book %q not found in results", b1.Title)
	}
	if !ids[b2.ID] {
		t.Errorf("unpositioned book %q not found in results", b2.Title)
	}
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
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "Book One" {
		t.Errorf("page1[0].Title = %q, want Book One", page1[0].Title)
	}
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
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}
