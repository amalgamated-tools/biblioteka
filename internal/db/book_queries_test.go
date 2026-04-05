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
