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
	require.Equal(t, 0, total)
	require.Len(t, books, 0)
}

func TestListBooksPaginated_OrdersByTitle(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"Zebra", "Apple", "Mango"} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListBooksPaginated()")
	require.Equal(t, 3, total)
	require.Len(t, books, 3)
	require.Equal(t, "Apple", books[0].Title)
	require.Equal(t, "Mango", books[1].Title)
	require.Equal(t, "Zebra", books[2].Title)
}

func TestListBooksPaginated_FirstPage(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.ListBooksPaginated(t.Context(), 2, 0)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=0)")
	require.Equal(t, 5, total)
	require.Len(t, page1, 2)
	require.Equal(t, "A", page1[0].Title)
	require.Equal(t, "B", page1[1].Title)
}

func TestListBooksPaginated_SecondPage(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page2, total, err := d.ListBooksPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListBooksPaginated(limit=2, offset=2)")
	require.Equal(t, 5, total)
	require.Len(t, page2, 2)
	require.Equal(t, "C", page2[0].Title)
}

// When offset is beyond the last row the window function returns zero rows,
// so the implementation issues a separate COUNT query. Verify the total is
// still reported correctly.
func TestListBooksPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Only Book"})
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.ListBooksPaginated(t.Context(), 10, 100)
	require.NoError(t, err, "ListBooksPaginated(offset=100)")
	require.Equal(t, 1, total)
	require.Len(t, books, 0)
}

// ---- ListRecentBooks ----

func TestListRecentBooks_Empty(t *testing.T) {
	d := newTestDB(t)

	books, total, err := d.ListRecentBooks(t.Context(), 10, 0)
	require.NoError(t, err, "ListRecentBooks()")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)
}

func TestListRecentBooks_Paginated(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"A", "B", "C", "D", "E"} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.ListRecentBooks(t.Context(), 2, 0)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=0)")
	require.Equal(t, 5, total)
	require.Len(t, page1, 2)

	page2, total2, err := d.ListRecentBooks(t.Context(), 2, 2)
	require.NoError(t, err, "ListRecentBooks(limit=2, offset=2)")
	require.Equal(t, 5, total2)
	require.Len(t, page2, 2)
}

func TestListRecentBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Solo"})
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.ListRecentBooks(t.Context(), 10, 50)
	require.NoError(t, err, "ListRecentBooks(offset=50)")
	require.Equal(t, 1, total)
	require.Len(t, books, 0)
}

// ---- ListBooksByAuthor ----

func TestListBooksByAuthor_Empty(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Nobody", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	books, err := d.ListBooksByAuthor(t.Context(), author.ID)
	require.NoError(t, err, "ListBooksByAuthor()")
	require.Len(t, books, 0)
}

func TestListBooksByAuthor_ReturnsMatchingBooks(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")
	other, err := d.CreateAuthor(t.Context(), "J.K. Rowling", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor(other)")

	b1, err := d.CreateBook(t.Context(), BookInput{Title: "The Gunslinger"})
	require.NoError(t, err, "CreateBook(b1)")
	b2, err := d.CreateBook(t.Context(), BookInput{Title: "It"})
	require.NoError(t, err, "CreateBook(b2)")
	b3, err := d.CreateBook(t.Context(), BookInput{Title: "Harry Potter"})
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
	require.Equal(t, "It", books[0].Title)
	require.Equal(t, "The Gunslinger", books[1].Title)
}

func TestListBooksByAuthorPaginated(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Prolific Author", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")

	for _, title := range []string{"Book A", "Book B", "Book C", "Book D"} {
		b, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
		err = d.SetBookAuthors(t.Context(), b.ID, []string{author.ID})
		require.NoError(t, err, "SetBookAuthors(%q)", title)
	}

	page1, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 0)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page1)")
	require.Equal(t, 4, total)
	require.Len(t, page1, 2)
	require.Equal(t, "Book A", page1[0].Title)

	page2, total2, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 2, 2)
	require.NoError(t, err, "ListBooksByAuthorPaginated(page2)")
	require.Equal(t, 4, total2)
	require.Len(t, page2, 2)
	require.Equal(t, "Book C", page2[0].Title)
}

func TestListBooksByAuthorPaginated_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	author, err := d.CreateAuthor(t.Context(), "Solo Author", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor()")
	b, err := d.CreateBook(t.Context(), BookInput{Title: "One Book"})
	require.NoError(t, err, "CreateBook()")
	err = d.SetBookAuthors(t.Context(), b.ID, []string{author.ID})
	require.NoError(t, err, "SetBookAuthors()")

	books, total, err := d.ListBooksByAuthorPaginated(t.Context(), author.ID, 10, 50)
	require.NoError(t, err, "ListBooksByAuthorPaginated(offset=50)")
	require.Equal(t, 1, total)
	require.Len(t, books, 0)
}
