package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
		t.Fatalf("len(books) = %d, want 1", len(books))
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
		require.NoError(t, err, "CreateBook(100%% Pure Fiction)")
	}
	if _, err := d.CreateBook(t.Context(), "Something Else", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateBook(Something Else)")
	}

	// Searching for "%" as a literal character should find exactly one book.
	books, total, err := d.SearchBooks(t.Context(), "%", 10, 0)
	require.NoError(t, err, "SearchBooks(%%) error")
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
		t.Fatalf("len(page1) = %d, want 2", len(page1))
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
		t.Fatalf("len(page2) = %d, want 2", len(page2))
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
