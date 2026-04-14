package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PostgreSQL integration tests for SearchBooks.
//
// These tests run against a real PostgreSQL instance. They are skipped when
// TEST_DATABASE_URL is not set (e.g. in standard local development without a
// running Postgres server). In CI they run in the go-test-postgres job which
// provides a Postgres 17 service container.
//
// PostgreSQL uses an ILIKE-based substring match (accelerated by the pg_trgm
// GIN index added in migration 20260412000000_add_books_trgm), so the expected
// behavior for some edge cases differs from the SQLite FTS5 implementation:
//
//   - Prefix queries match any substring (ILIKE `%query%`).
//   - The LIKE wildcard characters % and _ are escaped to \% and \_ so they
//     match literally in the search term.
//   - Backslash is escaped to \\ so it matches literally.
//   - FTS5-specific operator characters (+, AND, OR, NOT, *) are treated as
//     plain text because ILIKE does not parse them specially.

// ---- Core search behaviour ----

func TestSearchBooks_Postgres_NoMatch(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "Asimov", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, books)
}

func TestSearchBooks_Postgres_MatchesByTitle(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation and Empire"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "Foundation", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, books, 2)
}

func TestSearchBooks_Postgres_MatchesByDescription(t *testing.T) {
	d := newTestPostgresDB(t)

	desc := "A story about a desert planet"
	_, err := d.CreateBook(t.Context(), BookInput{Title: "Dune", Description: &desc})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "desert planet", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "Dune", books[0].Title)
}

func TestSearchBooks_Postgres_CaseInsensitive(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)

	for _, q := range []string{"foundation", "FOUNDATION", "Foundation", "fOuNdAtIoN"} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q)", q)
		require.Equal(t, 1, total, "SearchBooks(%q) total", q)
		require.Len(t, books, 1, "SearchBooks(%q) books", q)
	}
}

// ---- LIKE wildcard escaping ----

func TestSearchBooks_Postgres_PercentSignMatchedLiterally(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "100% Pure Fiction"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Something Else"})
	require.NoError(t, err)

	// "%" is escaped to "\%" in the ILIKE pattern, so it matches a literal
	// percent sign; only the book whose title contains "%" should be returned.
	books, total, err := d.SearchBooks(t.Context(), "%", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "100% Pure Fiction", books[0].Title)
}

func TestSearchBooks_Postgres_UnderscoreMatchedLiterally(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "hello_world"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "helloXworld"})
	require.NoError(t, err)

	// "_" is escaped to "\_" in the ILIKE pattern, so it matches a literal
	// underscore; only the book whose title contains "_" should be returned.
	books, total, err := d.SearchBooks(t.Context(), "_", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, "hello_world", books[0].Title)

	// Searching by full title with underscore also finds the right book.
	books, _, err = d.SearchBooks(t.Context(), "hello_world", 10, 0)
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "hello_world", books[0].Title)
}

func TestSearchBooks_Postgres_BackslashMatchedLiterally(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: `Back\slash`})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "BackXslash"})
	require.NoError(t, err)

	// "\" is escaped to "\\" in the ILIKE pattern, so it matches a literal
	// backslash; only the book whose title contains "\" should be returned.
	books, total, err := d.SearchBooks(t.Context(), `\`, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	require.Equal(t, `Back\slash`, books[0].Title)
}

// ---- Edge cases ----

func TestSearchBooks_Postgres_EmptyQuery(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)

	for _, q := range []string{"", "   "} {
		books, total, err := d.SearchBooks(t.Context(), q, 10, 0)
		require.NoError(t, err, "SearchBooks(%q)", q)
		require.Equal(t, 0, total, "SearchBooks(%q) total", q)
		require.Empty(t, books, "SearchBooks(%q) books", q)
	}
}

func TestSearchBooks_Postgres_SubstringMatch(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Foundation"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Foundation and Empire"})
	require.NoError(t, err)
	_, err = d.CreateBook(t.Context(), BookInput{Title: "Dune"})
	require.NoError(t, err)

	// ILIKE %Founda% is a substring match, so it finds both Foundation books.
	books, total, err := d.SearchBooks(t.Context(), "Founda", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, books, 2)
}

// ---- Pagination ----

func TestSearchBooks_Postgres_Paginated(t *testing.T) {
	d := newTestPostgresDB(t)

	for _, title := range []string{
		"Abc Foundation",
		"Def Foundation",
		"Ghi Foundation",
		"Jkl Foundation",
	} {
		_, err := d.CreateBook(t.Context(), BookInput{Title: title})
		require.NoError(t, err, "CreateBook(%q)", title)
	}

	page1, total, err := d.SearchBooks(t.Context(), "Foundation", 2, 0)
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Len(t, page1, 2)
	require.Equal(t, "Abc Foundation", page1[0].Title)

	page2, total2, err := d.SearchBooks(t.Context(), "Foundation", 2, 2)
	require.NoError(t, err)
	require.Equal(t, 4, total2)
	require.Len(t, page2, 2)
	require.Equal(t, "Ghi Foundation", page2[0].Title)
}

func TestSearchBooks_Postgres_OffsetBeyondTotal(t *testing.T) {
	d := newTestPostgresDB(t)

	_, err := d.CreateBook(t.Context(), BookInput{Title: "Searchable Book"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "Searchable", 10, 50)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Empty(t, books)
}

// ---- Live-data consistency ----

func TestSearchBooks_Postgres_UpdateTitleReflected(t *testing.T) {
	d := newTestPostgresDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Original Title"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "Original", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)

	_, err = d.UpdateBook(t.Context(), b.ID, BookInput{Title: "Updated Title"})
	require.NoError(t, err)

	// Old title must no longer match.
	books, total, err = d.SearchBooks(t.Context(), "Original", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, books)

	// New title must match.
	books, total, err = d.SearchBooks(t.Context(), "Updated", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
}

func TestSearchBooks_Postgres_DeleteReflected(t *testing.T) {
	d := newTestPostgresDB(t)

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Ephemeral Book"})
	require.NoError(t, err)

	books, total, err := d.SearchBooks(t.Context(), "Ephemeral", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)

	err = d.DeleteBook(t.Context(), b.ID)
	require.NoError(t, err)

	books, total, err = d.SearchBooks(t.Context(), "Ephemeral", 10, 0)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, books)
}
