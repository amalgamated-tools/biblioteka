package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- ListBooksPaginated ----

func TestListBooksPaginated_Empty(t *testing.T) {
	d := newTestDB(t)

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListBooksPaginated() error")
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestListBooksPaginated_OrdersByTitle(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"Zebra", "Apple", "Mango"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateBook(%q)", title)
		}
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListBooksPaginated() error")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(books) != 3 {
		require.Failf(t, "failed", "len(books) = %d, want 3", len(books))
	}
	if books[0].Title != "Apple" {
		t.Errorf("books[0].Title = %q, want Apple", books[0].Title)
	}
	if books[1].Title != "Mango" {
		t.Errorf("books[1].Title = %q, want Mango", books[1].Title)
	}
	if books[2].Title != "Zebra" {
		t.Errorf("books[2].Title = %q, want Zebra", books[2].Title)
	}
}

func TestListBooksPaginated_FirstPage(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateBook(%q)", title)
		}
	}

	page1, total, err := d.ListBooksPaginated(t.Context(), 2, 0)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=0) error")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "A" {
		t.Errorf("page1[0].Title = %q, want A", page1[0].Title)
	}
	if page1[1].Title != "B" {
		t.Errorf("page1[1].Title = %q, want B", page1[1].Title)
	}
}

func TestListBooksPaginated_SecondPage(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateBook(%q)", title)
		}
	}

	page2, total, err := d.ListBooksPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=2) error")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Title != "C" {
		t.Errorf("page2[0].Title = %q, want C", page2[0].Title)
	}
}

// When offset is beyond the last row the window function returns zero rows,
// so the implementation issues a separate COUNT query. Verify the total is
// still reported correctly.
func TestListBooksPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Only Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 100)
	require.NoError(t, err, "ListBooksPaginated(offset=100) error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

// ---- ListRecentBooks ----

func TestListRecentBooks_Empty(t *testing.T) {
	d := newTestDB(t)

	books, total, err := d.ListRecentBooks(t.Context(), 10, 0)
	require.NoError(t, err, "ListRecentBooks() error")
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestListRecentBooks_Paginated(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateBook(%q)", title)
		}
	}

	page1, total, err := d.ListRecentBooks(t.Context(), 2, 0)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=0) error")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}

	page2, total2, err := d.ListRecentBooks(t.Context(), 2, 2)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=2) error")
	if total2 != 5 {
		t.Errorf("page2 total = %d, want 5", total2)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
}

func TestListRecentBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Solo", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}

	books, total, err := d.ListRecentBooks(t.Context(), 10, 50)
	require.NoError(t, err, "ListRecentBooks(offset=50) error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

// ---- ListBooksByAuthor ----

func TestListBooksByAuthor_Empty(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Nobody", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	require.NoError(t, err, "ListBooksByAuthor() error")
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestListBooksByAuthor_ReturnsMatchingBooks(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")
	other, err := d.CreateAuthor(t.Context(), "J.K. Rowling", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor(other)")

	b1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b1)")
	b2, err := d.CreateBook(t.Context(), "It", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b2)")
	b3, err := d.CreateBook(t.Context(), "Harry Potter", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook(b3)")

	if err := d.SetBookAuthors(t.Context(), b1.ID, []string{author.ID}); err != nil {
		require.NoError(t, err, "SetBookAuthors(b1)")
	}
	if err := d.SetBookAuthors(t.Context(), b2.ID, []string{author.ID}); err != nil {
		require.NoError(t, err, "SetBookAuthors(b2)")
	}
	if err := d.SetBookAuthors(t.Context(), b3.ID, []string{other.ID}); err != nil {
		require.NoError(t, err, "SetBookAuthors(b3)")
	}

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	require.NoError(t, err, "ListBooksByAuthor() error")
	if len(books) != 2 {
		require.Failf(t, "failed", "len(books) = %d, want 2", len(books))
	}
	// Ordered by title: "It" before "The Gunslinger".
	if books[0].Title != "It" {
		t.Errorf("books[0].Title = %q, want It", books[0].Title)
	}
	if books[1].Title != "The Gunslinger" {
		t.Errorf("books[1].Title = %q, want The Gunslinger", books[1].Title)
	}
}

func TestListBooksByAuthorPaginated(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Prolific Author", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	for _, title := range []string{"Book A", "Book B", "Book C", "Book D"} {
		b, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
		if err := d.SetBookAuthors(t.Context(), b.ID, []string{author.ID}); err != nil {
			require.NoError(t, err, "SetBookAuthors(%q)", title)
		}
	}

	page1, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 0)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page1) error")
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "Book A" {
		t.Errorf("page1[0].Title = %q, want Book A", page1[0].Title)
	}

	page2, total2, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 2)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page2) error")
	if total2 != 4 {
		t.Errorf("page2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Title != "Book C" {
		t.Errorf("page2[0].Title = %q, want Book C", page2[0].Title)
	}
}

func TestListBooksByAuthorPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Solo Author", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")
	b, err := d.CreateBook(t.Context(), "One Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")
	if err := d.SetBookAuthors(t.Context(), b.ID, []string{author.ID}); err != nil {
		require.NoError(t, err, "SetBookAuthors()")
	}

	books, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 10, 50)
	require.NoError(t, err, "ListBooksByAuthorPaginated(offset=50) error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

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
	if err := d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		require.NoError(t, err, "SetBookSeries(b1)")
	}
	if err := d.SetBookSeries(t.Context(), b3.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(3.0)}}); err != nil {
		require.NoError(t, err, "SetBookSeries(b3)")
	}
	if err := d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(2.0)}}); err != nil {
		require.NoError(t, err, "SetBookSeries(b2)")
	}

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	require.NoError(t, err, "ListBooksBySeries() error")
	if len(books) != 3 {
		require.Failf(t, "failed", "len(books) = %d, want 3", len(books))
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

	if err := d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: nil}}); err != nil {
		require.NoError(t, err, "SetBookSeries(b2 nil position)")
	}
	if err := d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		require.NoError(t, err, "SetBookSeries(b1 pos 1)")
	}

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	require.NoError(t, err, "ListBooksBySeries() error")
	if len(books) != 2 {
		require.Failf(t, "failed", "len(books) = %d, want 2", len(books))
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
		if err := d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: &pos}}); err != nil {
			require.NoError(t, err, "SetBookSeries(%q)", title)
		}
	}

	page1, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 2, 0)
	require.NoError(t, err, "ListBooksBySeriesPaginated(page1) error")
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
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
	if err := d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		require.NoError(t, err, "SetBookSeries()")
	}

	books, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 10, 50)
	require.NoError(t, err, "ListBooksBySeriesPaginated(offset=50) error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

// ---- SearchBooks ----

func TestSearchBooks_NoMatch(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}

	books, total, err := d.SearchBooks(t.Context(), "Asimov", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestSearchBooks_MatchesByTitle(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Foundation", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Foundation)")
	}
	if _, err := d.CreateBook(t.Context(), "Foundation and Empire", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Foundation and Empire)")
	}
	if _, err := d.CreateBook(t.Context(), "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Dune)")
	}

	books, total, err := d.SearchBooks(t.Context(), "Foundation", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(books) != 2 {
		t.Errorf("len(books) = %d, want 2", len(books))
	}
}

func TestSearchBooks_MatchesByDescription(t *testing.T) {
	d := newTestDB(t)

	desc := "A story about a desert planet"
	if _, err := d.CreateBook(t.Context(), "Dune", &desc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Dune)")
	}
	if _, err := d.CreateBook(t.Context(), "Foundation", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Foundation)")
	}

	books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 1 {
		require.Failf(t, "failed", "len(books) = %d, want 1", len(books))
	}
	if books[0].Title != "Dune" {
		t.Errorf("books[0].Title = %q, want Dune", books[0].Title)
	}
}

func TestSearchBooks_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Foundation", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}

	for _, q := range []string{"foundation", "FOUNDATION", "Foundation", "fOuNdAtIoN"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q) error", q)
		if total != 1 {
			t.Errorf("SearchBooks(%q) total = %d, want 1", q, total)
		}
		if len(books) != 1 {
			t.Errorf("SearchBooks(%q) len = %d, want 1", q, len(books))
		}
	}
}

func TestSearchBooks_SpecialCharacterEscaping(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "100% Pure Fiction", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err)
	}
	if _, err := d.CreateBook(t.Context(), "Something Else", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Something Else)")
	}

	// Searching for "%" as a literal character should find exactly one book.
	books, total, err := d.SearchBooks(t.Context(), "%", 10, 0)
	require.NoError(t, err)
	if total != 1 {
		t.Errorf("SearchBooks(%%) total = %d, want 1", total)
	}
	if len(books) != 1 {
		t.Errorf("SearchBooks(%%) len = %d, want 1", len(books))
	}
	if books[0].Title != "100% Pure Fiction" {
		t.Errorf("books[0].Title = %q, want 100%% Pure Fiction", books[0].Title)
	}
}

func TestSearchBooks_UnderscoreEscaping(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "hello_world", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(hello_world)")
	}
	if _, err := d.CreateBook(t.Context(), "helloXworld", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(helloXworld)")
	}

	// Searching for literal underscore should find only the one with "_".
	books, _, err := d.SearchBooks(t.Context(), "hello_world", 10, 0)
	require.NoError(t, err, "SearchBooks(hello_world) error")
	if len(books) != 1 {
		t.Errorf("len(books) = %d, want 1 (literal underscore)", len(books))
	}
}

func TestSearchBooks_BackslashEscaping(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), `Back\slash`, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}
	if _, err := d.CreateBook(t.Context(), "BackXslash", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(BackXslash)")
	}

	books, _, err := d.SearchBooks(t.Context(), `\`, 10, 0)
	require.NoError(t, err, "SearchBooks(backslash) error")
	if len(books) != 1 {
		t.Errorf("len(books) = %d, want 1 (literal backslash)", len(books))
	}
}

func TestSearchBooks_Paginated(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"Abc Foundation", "Def Foundation", "Ghi Foundation", "Jkl Foundation"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateBook(%q)", title)
		}
	}

	page1, total, err := d.SearchBooks(t.Context(), "Foundation", 2, 0)
	require.NoError(t, err, "SearchBooks(page1) error")
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "Abc Foundation" {
		t.Errorf("page1[0].Title = %q, want Abc Foundation", page1[0].Title)
	}

	page2, total2, err := d.SearchBooks(t.Context(), "Foundation", 2, 2)
	require.NoError(t, err, "SearchBooks(page2) error")
	if total2 != 4 {
		t.Errorf("page2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Title != "Ghi Foundation" {
		t.Errorf("page2[0].Title = %q, want Ghi Foundation", page2[0].Title)
	}
}

func TestSearchBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Searchable Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook()")
	}

	books, total, err := d.SearchBooks(t.Context(), "Searchable", 10, 50)
	require.NoError(t, err, "SearchBooks(offset=50) error")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}
