package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- SearchBooks ----

func TestSearchBooks_NoMatch(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.SearchBooks(t.Context(), "Asimov", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)
}

func TestSearchBooks_MatchesByTitle(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook(Foundation)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation and Empire"})
	require.NoError(t, err, "CreateBook(Foundation and Empire)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err, "CreateBook(Dune)")

	books, total, err := d.SearchBooks(t.Context(), "Foundation", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	require.Equal(t, 2, total)
	require.Len(t, books, 2)
}

func TestSearchBooks_MatchesByDescription(t *testing.T) {
	d := newTestDB(t)

	desc := "A story about a desert planet"
	_, err := d.CreateBook(t.Context(), BookInput{Title: "Dune", Description: &desc})
	require.NoError(t, err, "CreateBook(Dune)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook(Foundation)")

	books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
	require.NoError(t, err, "SearchBooks() error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "Dune", books[0].Title)
}

func TestSearchBooks_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook()")

	for _, q := range []string{"foundation", "FOUNDATION", "Foundation", "fOuNdAtIoN"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q) error", q)
		require.Equal(t, 1, total)
		require.Len(t, books, 1)
	}
}

func TestSearchBooks_SpecialCharacterEscaping(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "100% Pure Fiction"})
	require.NoError(t, err, "CreateBook(100%% Pure Fiction)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Something Else"})
	require.NoError(t, err, "CreateBook(Something Else)")

	// "%" contains no letter or digit so it produces an empty FTS5 query;
	// the search returns no results rather than matching everything.
	books, total, err := d.SearchBooks(t.Context(), "%", 10, 0)
	require.NoError(t, err, "SearchBooks(%%) error")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)

	// Searching for the word that appears in the title still works.
	books, total, err = d.SearchBooks(t.Context(), "Pure", 10, 0)
	require.NoError(t, err, "SearchBooks(Pure) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "100% Pure Fiction", books[0].Title)
}

func TestSearchBooks_UnderscoreEscaping(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "hello_world"})
	require.NoError(t, err, "CreateBook(hello_world)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "helloXworld"})
	require.NoError(t, err, "CreateBook(helloXworld)")

	// Searching for literal underscore should find only the one with "_".
	books, _, err := d.SearchBooks(t.Context(), "hello_world", 10, 0)
	require.NoError(t, err, "SearchBooks(hello_world) error")
	require.Len(t, books, 1)
}

func TestSearchBooks_BackslashEscaping(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: `Back\slash`})
	require.NoError(t, err, "CreateBook()")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "BackXslash"})
	require.NoError(t, err, "CreateBook(BackXslash)")

	// "\" contains no letter or digit so it produces an empty FTS5 query;
	// the search returns no results.
	books, _, err := d.SearchBooks(t.Context(), `\`, 10, 0)
	require.NoError(t, err, "SearchBooks(backslash) error")
	require.Len(t, books, 0)

	// Searching for a real word token from the title still works.
	books, _, err = d.SearchBooks(t.Context(), "Back", 10, 0)
	require.NoError(t, err, "SearchBooks(Back) error")
	require.Len(t, books, 2)
}

func TestSearchBooks_Paginated(t *testing.T) {
	d := newTestDB(t)

	for _, title := range []string{"Abc Foundation", "Def Foundation", "Ghi Foundation", "Jkl Foundation"} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.SearchBooks(t.Context(), "Foundation", 2, 0)
	require.NoError(t, err, "SearchBooks(page1) error")
	require.Equal(t, 4, total)
	require.Len(t, page1, 2)
	require.Equal(t, "Abc Foundation", page1[0].Title)

	page2, total2, err := d.SearchBooks(t.Context(), "Foundation", 2, 2)
	require.NoError(t, err, "SearchBooks(page2) error")
	require.Equal(t, 4, total2)
	require.Len(t, page2, 2)
	require.Equal(t, "Ghi Foundation", page2[0].Title)
}

func TestSearchBooks_OffsetBeyondTotal(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Searchable Book"})
	require.NoError(t, err, "CreateBook()")

	books, total, err := d.SearchBooks(t.Context(), "Searchable", 10, 50)
	require.NoError(t, err, "SearchBooks(offset=50) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 0)
}

func TestSearchBooks_PrefixMatch(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook(Foundation)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation and Empire"})
	require.NoError(t, err, "CreateBook(Foundation and Empire)")

	_, err = d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err, "CreateBook(Dune)")

	// Partial-word prefix query should match both Foundation books.
	books, total, err := d.SearchBooks(t.Context(), "Founda", 10, 0)
	require.NoError(t, err, "SearchBooks(Founda) error")
	require.Equal(t, 2, total)
	require.Len(t, books, 2)
}

func TestSearchBooks_FTS5OperatorCharsDoNotError(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook(Foundation)")

	// FTS5 operator characters passed raw by the user must not cause errors.
	for _, q := range []string{`"Foundation"`, "Foundation*", "Foundation-", "-Foundation", "Foundation AND Dune"} {
		books, _, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q) must not error", q)
		_ = books
	}
}

func TestSearchBooks_EmptyAfterSanitize(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err, "CreateBook(Foundation)")

	// Queries composed entirely of non-word characters produce an empty FTS5
	// expression; SearchBooks must return zero results without erroring.
	for _, q := range []string{"%", `\`, "*", "-", "---", "% * -"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q) must not error", q)
		require.Equal(t, 0, total, "SearchBooks(%q) total", q)
		require.Len(t, books, 0, "SearchBooks(%q) books", q)
	}
}
