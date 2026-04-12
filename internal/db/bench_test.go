package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// newBenchDB creates an in-memory SQLite database with all migrations applied,
// suitable for use in benchmark functions. It registers a cleanup function so
// the database is closed when the benchmark ends.
func newBenchDB(b *testing.B) *DB {
	b.Helper()
	b.Setenv("BIBLIOTEKA_ENV", "test")
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(b, err, "newBenchDB: open")
	b.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(b, sqlDB.Ping(), "newBenchDB: ping")

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(b, err, "newBenchDB: pragmas")

	d := &DB{DB: sqlDB, Dialect: DialectSQLite}

	require.NoError(b, runMigrations(b.Context(), d), "newBenchDB: migrations")

	return d
}

// seedBooks inserts n books into the DB. Book titles are of the form
// "Book 00001", …, "Book 0NNNN" so that lexicographic and numeric order agree.
func seedBooks(b *testing.B, d *DB, n int) {
	b.Helper()
	ctx := b.Context()
	for i := range n {
		_, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Book %05d", i+1)})
		require.NoError(b, err, "seedBooks: CreateBook")
	}
}

// ---- ListBooksPaginated ----

func BenchmarkListBooksPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 100)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListBooksPaginated(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListBooksPaginated_1000(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 1000)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListBooksPaginated(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListRecentBooks ----

func BenchmarkListRecentBooks_100(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 100)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListRecentBooks(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListRecentBooks_1000(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 1000)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListRecentBooks(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- SearchBooks ----

func BenchmarkSearchBooks_Hit_100(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 100)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		// "Book" matches all 100 titles; benchmarks the first-page path.
		_, _, err := d.SearchBooks(ctx, "Book", 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchBooks_Hit_1000(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 1000)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.SearchBooks(ctx, "Book", 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSearchBooks_NoMatch_1000(b *testing.B) {
	d := newBenchDB(b)
	seedBooks(b, d, 1000)
	ctx := b.Context()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.SearchBooks(ctx, "zzz-no-match", 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListBooksByAuthorPaginated ----

func BenchmarkListBooksByAuthorPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	author, err := d.CreateAuthor(ctx, "Prolific Author", nil, nil, nil, nil)
	require.NoError(b, err, "CreateAuthor")

	for i := range 100 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Author Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		require.NoError(b, d.SetBookAuthors(ctx, bk.ID, []string{author.ID}), "SetBookAuthors")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListBooksByAuthorPaginated(ctx, author.ID, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListBooksBySeriesPaginated ----

func BenchmarkListBooksBySeriesPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	s, err := d.CreateSeries(ctx, "Long Series", nil, nil, nil)
	require.NoError(b, err, "CreateSeries")

	for i := range 100 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Series Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		pos := float64(i + 1)
		require.NoError(b, d.SetBookSeries(ctx, bk.ID, []BookSeriesInput{{SeriesID: s.ID, Position: &pos}}), "SetBookSeries")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListBooksBySeriesPaginated(ctx, s.ID, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListBooksByLibraryPaginated ----

func BenchmarkListBooksByLibraryPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	lib, err := d.CreateLibrary(ctx, "Benchmark Library", "/books", LibraryOrganizationNone, false)
	require.NoError(b, err, "CreateLibrary")

	for i := range 100 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Library Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		require.NoError(b, d.AddBookToLibrary(ctx, lib.ID, bk.ID), "AddBookToLibrary")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListBooksByLibraryPaginated(ctx, lib.ID, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListAuthorsPaginated ----

func BenchmarkListAuthorsPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	for i := range 100 {
		_, err := d.CreateAuthor(ctx, fmt.Sprintf("Author %05d", i+1), nil, nil, nil, nil)
		require.NoError(b, err, "CreateAuthor")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListAuthorsPaginated(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListSeriesPaginated ----

func BenchmarkListSeriesPaginated_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	for i := range 100 {
		_, err := d.CreateSeries(ctx, fmt.Sprintf("Series %05d", i+1), nil, nil, nil)
		require.NoError(b, err, "CreateSeries")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListSeriesPaginated(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}
