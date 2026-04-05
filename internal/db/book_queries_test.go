package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- ListBooksPaginated ----

func TestListBooksPaginated_Empty(t *testing.T) {
	d := newTestDB(t)

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListBooksPaginated()")
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
		_, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListBooksPaginated()")
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	require.Len(t, books, 3)
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
		_, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.ListBooksPaginated(t.Context(), 2, 0)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=0)")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	require.Len(t, page1, 2)
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
		_, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page2, total, err := d.ListBooksPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=2)")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	require.Len(t, page2, 2)
	if page2[0].Title != "C" {
		t.Errorf("page2[0].Title = %q, want C", page2[0].Title)
	}
}

// When offset is beyond the last row the window function returns zero rows,
// so the implementation issues a separate COUNT query. Verify the total is
// still reported correctly.
func TestListBooksPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), "Only Book", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 100)
	require.NoError(t, err, "ListBooksPaginated(offset=100)")
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
	require.NoError(t, err, "ListRecentBooks()")
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
		_, err := d.CreateBook(t.Context(), title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.ListRecentBooks(t.Context(), 2, 0)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=0)")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	require.Len(t, page1, 2)

	page2, total2, err := d.ListRecentBooks(t.Context(), 2, 2)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=2)")
	if total2 != 5 {
		t.Errorf("page2 total = %d, want 5", total2)
	}
	require.Len(t, page2, 2)
}

func TestListRecentBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), "Solo", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.ListRecentBooks(t.Context(), 10, 50)
	require.NoError(t, err, "ListRecentBooks(offset=50)")
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
	require.NoError(t, err, "ListBooksByAuthor()")
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

	err = d.SetBookAuthors(t.Context(), b1.ID, []string{author.ID})
	require.NoError(t, err, "SetBookAuthors(b1)")
	err = d.SetBookAuthors(t.Context(), b2.ID, []string{author.ID})
	require.NoError(t, err, "SetBookAuthors(b2)")
	err = d.SetBookAuthors(t.Context(), b3.ID, []string{other.ID})
	require.NoError(t, err, "SetBookAuthors(b3)")

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	require.NoError(t, err, "ListBooksByAuthor()")
	require.Len(t, books, 2)
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
		err = d.SetBookAuthors(t.Context(), b.ID, []string{author.ID})
		require.NoError(t, err, "SetBookAuthors(%q)", title)
	}

	page1, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 0)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page1)")
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	require.Len(t, page1, 2)
	if page1[0].Title != "Book A" {
		t.Errorf("page1[0].Title = %q, want Book A", page1[0].Title)
	}

	page2, total2, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 2)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page2)")
	if total2 != 4 {
		t.Errorf("page2 total = %d, want 4", total2)
	}
	require.Len(t, page2, 2)
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
	err = d.SetBookAuthors(t.Context(), b.ID, []string{author.ID})
	require.NoError(t, err, "SetBookAuthors()")

	books, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 10, 50)
	require.NoError(t, err, "ListBooksByAuthorPaginated(offset=50)")
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(books) != 0 {
		t.Errorf("len(books) = %d, want 0", len(books))
	}
}
