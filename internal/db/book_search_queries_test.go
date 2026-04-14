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
	// Each of these sanitizes to a phrase-quoted form containing "Foundation",
	// so they should find the book.
	for _, q := range []string{`"Foundation"`, "Foundation*", "Foundation-", "-Foundation"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q) must not error", q)
		require.Equal(t, 1, total, "SearchBooks(%q) should find 1 book", q)
		require.Len(t, books, 1, "SearchBooks(%q) should return 1 book", q)
		require.Equal(t, "Foundation", books[0].Title, "SearchBooks(%q) book title", q)
	}

	// Multi-word query: "Foundation AND Dune" sanitizes to three required tokens
	// ("Foundation"*, "AND"*, "Dune"*). A book with only "Foundation" in the
	// title won't match all three, but the query must still not error.
	_, _, err = d.SearchBooks(t.Context(), "Foundation AND Dune", 10, 0)
	require.NoError(t, err, `SearchBooks("Foundation AND Dune") must not error`)
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

// TestSearchBooks_MultiWordSemanticDivergence explicitly documents the
// behavioral gap between the two search backends:
//
//   - SQLite FTS5 (token-AND): each whitespace-separated word in the query
//     becomes an independent FTS5 term; all terms must match somewhere in the
//     combined FTS document (title ∪ description), but they may appear in any
//     order and may each be satisfied by a different column.
//
//   - PostgreSQL ILIKE (phrase-substring): the entire query string is wrapped in
//     `%…%` and matched as a single contiguous substring against each column
//     independently; tokens that are present but in a different order, or split
//     across columns, do NOT produce a match.
//
// The two sub-cases below run against the SQLite path and assert the FTS5
// token-AND semantics. Comments on each sub-case describe what the PostgreSQL
// ILIKE path would return for the same input so that the divergence is
// self-documenting.
func TestSearchBooks_MultiWordSemanticDivergence(t *testing.T) {
	t.Run("reversed word order in same field", func(t *testing.T) {
		// Description: "Planet of Deserts" – the words "desert" and "planet" both appear
		// but in the opposite order to the query "desert planet".
		//
		// FTS5 (SQLite): "desert"* AND "planet"* → both tokens match the
		// combined document → MATCH.
		//
		// ILIKE (PostgreSQL): %desert planet% → "Planet of Deserts" does not
		// contain the literal substring "desert planet" → NO MATCH.
		d := newTestDB(t)

		desc := "Planet of Deserts"
		_, err := d.CreateBook(t.Context(), BookInput{Title: "Reversed Order", Description: &desc})
		require.NoError(t, err, "CreateBook(Reversed Order)")

		books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
		require.NoError(t, err, "SearchBooks('desert planet') error")
		require.Equal(t, 1, total, "FTS5 token-AND should match reversed word order")
		require.Len(t, books, 1)
		require.Equal(t, "Reversed Order", books[0].Title)
	})

	t.Run("words split across title and description", func(t *testing.T) {
		// Title contains "desert"; description contains "planet".
		// Neither field alone contains both words.
		//
		// FTS5 (SQLite): the combined FTS document (title ∥ description) has
		// both "desert" and "planet" → "desert"* AND "planet"* → MATCH.
		//
		// ILIKE (PostgreSQL): %desert planet% is checked against title and
		// description separately; neither field alone contains the phrase
		// "desert planet" → NO MATCH.
		d := newTestDB(t)

		desc := "life on another planet"
		_, err := d.CreateBook(t.Context(), BookInput{Title: "Desert Chronicles", Description: &desc})
		require.NoError(t, err, "CreateBook(Desert Chronicles)")

		books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
		require.NoError(t, err, "SearchBooks('desert planet') error")
		require.Equal(t, 1, total, "FTS5 token-AND should match words split across title and description")
		require.Len(t, books, 1)
		require.Equal(t, "Desert Chronicles", books[0].Title)
	})
}

// ---- FTS trigger sync tests ----

func TestSearchBooks_UpdateTitleSyncsIndex(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Original Title"})
	require.NoError(t, err, "CreateBook()")

	// Verify the original title is searchable.
	books, total, err := d.SearchBooks(t.Context(), "Original", 10, 0)
	require.NoError(t, err, "SearchBooks(Original) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)

	// Update the title.
	_, err = d.UpdateBook(t.Context(), b.ID, BookInput{Title: "Updated Title"})
	require.NoError(t, err, "UpdateBook()")

	// Old title should no longer match.
	books, total, err = d.SearchBooks(t.Context(), "Original", 10, 0)
	require.NoError(t, err, "SearchBooks(Original) after update error")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)

	// New title should match.
	books, total, err = d.SearchBooks(t.Context(), "Updated", 10, 0)
	require.NoError(t, err, "SearchBooks(Updated) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
}

func TestSearchBooks_UpdateDescriptionSyncsIndex(t *testing.T) {
	d := newTestDB(t)

	oldDesc := "desert planet adventure"
	b, err := d.CreateBook(t.Context(), BookInput{Title: "Dune", Description: &oldDesc})
	require.NoError(t, err, "CreateBook()")

	// Verify the original description is searchable.
	books, total, err := d.SearchBooks(t.Context(), "desert", 10, 0)
	require.NoError(t, err, "SearchBooks(desert) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)

	// Update the description.
	newDesc := "galactic empire saga"
	_, err = d.UpdateBook(t.Context(), b.ID, BookInput{Title: "Dune", Description: &newDesc})
	require.NoError(t, err, "UpdateBook()")

	// Old description terms should no longer match.
	books, total, err = d.SearchBooks(t.Context(), "desert", 10, 0)
	require.NoError(t, err, "SearchBooks(desert) after update error")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)

	// New description terms should match.
	books, total, err = d.SearchBooks(t.Context(), "galactic", 10, 0)
	require.NoError(t, err, "SearchBooks(galactic) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
}

func TestSearchBooks_DeleteRemovesFromIndex(t *testing.T) {
	d := newTestDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Ephemeral Book"})
	require.NoError(t, err, "CreateBook()")

	// Verify the book is searchable.
	books, total, err := d.SearchBooks(t.Context(), "Ephemeral", 10, 0)
	require.NoError(t, err, "SearchBooks(Ephemeral) error")
	require.Equal(t, 1, total)
	require.Len(t, books, 1)

	// Delete the book.
	err = d.DeleteBook(t.Context(), b.ID)
	require.NoError(t, err, "DeleteBook()")

	// Deleted book should no longer appear in search results.
	books, total, err = d.SearchBooks(t.Context(), "Ephemeral", 10, 0)
	require.NoError(t, err, "SearchBooks(Ephemeral) after delete error")
	require.Equal(t, 0, total)
	require.Len(t, books, 0)
}
