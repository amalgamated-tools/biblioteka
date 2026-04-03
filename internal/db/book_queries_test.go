package db

import (
	"testing"
)

// ---- ListBooksPaginated ----

func TestListBooksPaginated_Empty(t *testing.T) {
	d := newTestDB(t)

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	if err != nil {
		t.Fatalf("ListBooksPaginated() error: %v", err)
	}
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
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	if err != nil {
		t.Fatalf("ListBooksPaginated() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(books) != 3 {
		t.Fatalf("len(books) = %d, want 3", len(books))
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
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
	}

	page1, total, err := d.ListBooksPaginated(t.Context(), 2, 0)
	if err != nil {
		t.Fatalf("ListBooksPaginated(limit=2, offset=0) error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
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
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
	}

	page2, total, err := d.ListBooksPaginated(t.Context(), 2, 2)
	if err != nil {
		t.Fatalf("ListBooksPaginated(limit=2, offset=2) error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
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
		t.Fatalf("CreateBook(): %v", err)
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 100)
	if err != nil {
		t.Fatalf("ListBooksPaginated(offset=100) error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("ListRecentBooks() error: %v", err)
	}
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
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
	}

	page1, total, err := d.ListRecentBooks(t.Context(), 2, 0)
	if err != nil {
		t.Fatalf("ListRecentBooks(limit=2, offset=0) error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}

	page2, total2, err := d.ListRecentBooks(t.Context(), 2, 2)
	if err != nil {
		t.Fatalf("ListRecentBooks(limit=2, offset=2) error: %v", err)
	}
	if total2 != 5 {
		t.Errorf("page2 total = %d, want 5", total2)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}
}

func TestListRecentBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Solo", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	books, total, err := d.ListRecentBooks(t.Context(), 10, 50)
	if err != nil {
		t.Fatalf("ListRecentBooks(offset=50) error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	if err != nil {
		t.Fatalf("ListBooksByAuthor() error: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestListBooksByAuthor_ReturnsMatchingBooks(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}
	other, err := d.CreateAuthor(t.Context(), "J.K. Rowling", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateAuthor(other): %v", err)
	}

	b1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b1): %v", err)
	}
	b2, err := d.CreateBook(t.Context(), "It", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b2): %v", err)
	}
	b3, err := d.CreateBook(t.Context(), "Harry Potter", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b3): %v", err)
	}

	if err := d.SetBookAuthors(t.Context(), b1.ID, []string{author.ID}); err != nil {
		t.Fatalf("SetBookAuthors(b1): %v", err)
	}
	if err := d.SetBookAuthors(t.Context(), b2.ID, []string{author.ID}); err != nil {
		t.Fatalf("SetBookAuthors(b2): %v", err)
	}
	if err := d.SetBookAuthors(t.Context(), b3.ID, []string{other.ID}); err != nil {
		t.Fatalf("SetBookAuthors(b3): %v", err)
	}

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	if err != nil {
		t.Fatalf("ListBooksByAuthor() error: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("len(books) = %d, want 2", len(books))
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
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}

	for _, title := range []string{"Book A", "Book B", "Book C", "Book D"} {
		b, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
		if err := d.SetBookAuthors(t.Context(), b.ID, []string{author.ID}); err != nil {
			t.Fatalf("SetBookAuthors(%q): %v", title, err)
		}
	}

	page1, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListBooksByAuthorPaginated(page1) error: %v", err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "Book A" {
		t.Errorf("page1[0].Title = %q, want Book A", page1[0].Title)
	}

	page2, total2, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListBooksByAuthorPaginated(page2) error: %v", err)
	}
	if total2 != 4 {
		t.Errorf("page2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Title != "Book C" {
		t.Errorf("page2[0].Title = %q, want Book C", page2[0].Title)
	}
}

func TestListBooksByAuthorPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Solo Author", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateAuthor(): %v", err)
	}
	b, err := d.CreateBook(t.Context(), "One Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	if err := d.SetBookAuthors(t.Context(), b.ID, []string{author.ID}); err != nil {
		t.Fatalf("SetBookAuthors(): %v", err)
	}

	books, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 10, 50)
	if err != nil {
		t.Fatalf("ListBooksByAuthorPaginated(offset=50) error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("ListBooksBySeries() error: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}

func TestListBooksBySeries_OrderedByPosition(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	b1, err := d.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b1): %v", err)
	}
	b2, err := d.CreateBook(t.Context(), "The Drawing of the Three", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b2): %v", err)
	}
	b3, err := d.CreateBook(t.Context(), "The Waste Lands", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(b3): %v", err)
	}

	// Assign series entries out of order.
	if err := d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		t.Fatalf("SetBookSeries(b1): %v", err)
	}
	if err := d.SetBookSeries(t.Context(), b3.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(3.0)}}); err != nil {
		t.Fatalf("SetBookSeries(b3): %v", err)
	}
	if err := d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(2.0)}}); err != nil {
		t.Fatalf("SetBookSeries(b2): %v", err)
	}

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("ListBooksBySeries() error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	b1, err := d.CreateBook(t.Context(), "Positioned", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(positioned): %v", err)
	}
	b2, err := d.CreateBook(t.Context(), "Unpositioned", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(unpositioned): %v", err)
	}

	if err := d.SetBookSeries(t.Context(), b2.ID, []BookSeriesInput{{SeriesID: s.ID, Position: nil}}); err != nil {
		t.Fatalf("SetBookSeries(b2 nil position): %v", err)
	}
	if err := d.SetBookSeries(t.Context(), b1.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		t.Fatalf("SetBookSeries(b1 pos 1): %v", err)
	}

	books, err := d.ListBooksBySeries(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("ListBooksBySeries() error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}

	for i, title := range []string{"Book One", "Book Two", "Book Three", "Book Four"} {
		b, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
		pos := float64(i + 1)
		if err := d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: &pos}}); err != nil {
			t.Fatalf("SetBookSeries(%q): %v", title, err)
		}
	}

	page1, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 2, 0)
	if err != nil {
		t.Fatalf("ListBooksBySeriesPaginated(page1) error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateSeries(): %v", err)
	}
	b, err := d.CreateBook(t.Context(), "One Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	if err := d.SetBookSeries(t.Context(), b.ID, []BookSeriesInput{{SeriesID: s.ID, Position: new(1.0)}}); err != nil {
		t.Fatalf("SetBookSeries(): %v", err)
	}

	books, total, err := d.ListBooksBySeriesPaginated(t.Context(), s.ID, 10, 50)
	if err != nil {
		t.Fatalf("ListBooksBySeriesPaginated(offset=50) error: %v", err)
	}
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
		t.Fatalf("CreateBook(): %v", err)
	}

	books, total, err := d.SearchBooks(t.Context(), "Asimov", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks() error: %v", err)
	}
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
		t.Fatalf("CreateBook(Foundation): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "Foundation and Empire", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(Foundation and Empire): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "Dune", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(Dune): %v", err)
	}

	books, total, err := d.SearchBooks(t.Context(), "Foundation", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks() error: %v", err)
	}
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
		t.Fatalf("CreateBook(Dune): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "Foundation", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(Foundation): %v", err)
	}

	books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 1 {
		t.Fatalf("len(books) = %d, want 1", len(books))
	}
	if books[0].Title != "Dune" {
		t.Errorf("books[0].Title = %q, want Dune", books[0].Title)
	}
}

func TestSearchBooks_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Foundation", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	for _, q := range []string{"foundation", "FOUNDATION", "Foundation", "fOuNdAtIoN"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		if err != nil {
			t.Fatalf("SearchBooks(%q) error: %v", q, err)
		}
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
		t.Fatalf("CreateBook(100%% Pure Fiction): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "Something Else", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(Something Else): %v", err)
	}

	// Searching for "%" as a literal character should find exactly one book.
	books, total, err := d.SearchBooks(t.Context(), "%", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks(%%) error: %v", err)
	}
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
		t.Fatalf("CreateBook(hello_world): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "helloXworld", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(helloXworld): %v", err)
	}

	// Searching for literal underscore should find only the one with "_".
	books, _, err := d.SearchBooks(t.Context(), "hello_world", 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks(hello_world) error: %v", err)
	}
	if len(books) != 1 {
		t.Errorf("len(books) = %d, want 1 (literal underscore)", len(books))
	}
}

func TestSearchBooks_BackslashEscaping(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), `Back\slash`, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}
	if _, err := d.CreateBook(t.Context(), "BackXslash", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(BackXslash): %v", err)
	}

	books, _, err := d.SearchBooks(t.Context(), `\`, 10, 0)
	if err != nil {
		t.Fatalf("SearchBooks(backslash) error: %v", err)
	}
	if len(books) != 1 {
		t.Errorf("len(books) = %d, want 1 (literal backslash)", len(books))
	}
}

func TestSearchBooks_Paginated(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"Abc Foundation", "Def Foundation", "Ghi Foundation", "Jkl Foundation"} {
		if _, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			t.Fatalf("CreateBook(%q): %v", title, err)
		}
	}

	page1, total, err := d.SearchBooks(t.Context(), "Foundation", 2, 0)
	if err != nil {
		t.Fatalf("SearchBooks(page1) error: %v", err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Title != "Abc Foundation" {
		t.Errorf("page1[0].Title = %q, want Abc Foundation", page1[0].Title)
	}

	page2, total2, err := d.SearchBooks(t.Context(), "Foundation", 2, 2)
	if err != nil {
		t.Fatalf("SearchBooks(page2) error: %v", err)
	}
	if total2 != 4 {
		t.Errorf("page2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Title != "Ghi Foundation" {
		t.Errorf("page2[0].Title = %q, want Ghi Foundation", page2[0].Title)
	}
}

func TestSearchBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateBook(t.Context(), "Searchable Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("CreateBook(): %v", err)
	}

	books, total, err := d.SearchBooks(t.Context(), "Searchable", 10, 50)
	if err != nil {
		t.Fatalf("SearchBooks(offset=50) error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}
