package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

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

	// In-memory SQLite databases are connection-scoped: each new connection
	// from the pool gets its own empty database.  Pinning to a single
	// connection ensures all queries see the same schema and data.  This
	// serializes concurrent callers (e.g. LoadBookRelations' errgroup), so
	// it does not exercise true parallelism; a shared-cache DSN
	// (file::memory:?cache=shared) would be needed for that.
	sqlDB.SetMaxOpenConns(1)

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

// ---- LoadBookRelations ----

// BenchmarkLoadBookRelations measures the concurrent three-query fetch for a
// single book's authors, files, and series. This benchmark covers the book
// detail and Kobo metadata endpoints.
func BenchmarkLoadBookRelations(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	bk, err := d.CreateBook(ctx, BookInput{Title: "Benchmark Book"})
	require.NoError(b, err, "CreateBook")

	author, err := d.CreateAuthor(ctx, "Benchmark Author", nil, nil, nil, nil)
	require.NoError(b, err, "CreateAuthor")
	require.NoError(b, d.SetBookAuthors(ctx, bk.ID, []string{author.ID}), "SetBookAuthors")

	series, err := d.CreateSeries(ctx, "Benchmark Series", nil, nil, nil)
	require.NoError(b, err, "CreateSeries")
	pos := 1.0
	require.NoError(b, d.SetBookSeries(ctx, bk.ID, []BookSeriesInput{{SeriesID: series.ID, Position: &pos}}), "SetBookSeries")

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.LoadBookRelations(ctx, bk.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- GetYearInBooks ----

// BenchmarkGetYearInBooks measures the concurrent four-query fetch for a user's
// year-in-books statistics (books finished, active days, total downloads, and
// longest streak). The four queries run concurrently via errgroup.
func BenchmarkGetYearInBooks(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-user", "bench@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	// Seed 30 reading progress records across distinct documents.
	for i := range 30 {
		doc := fmt.Sprintf("doc-%02d", i)
		_, err := d.UpsertReadingProgress(ctx, user.ID, doc, "/p[1]", 0.5, nil, nil)
		require.NoError(b, err, "UpsertReadingProgress")
	}

	year := time.Now().UTC().Year()
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.GetYearInBooks(ctx, user.ID, year)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListAuditLogs ----

// BenchmarkListAuditLogs measures pagination over the audit_logs table using
// the idx_audit_logs_created_at_id ordering index (created_at DESC, id DESC)
// to support efficient ORDER BY and page traversal.
func BenchmarkListAuditLogs_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-audit-user", "audit@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	for range 100 {
		err := d.CreateAuditLog(ctx, user.ID, "book.created", "book", "book-1", nil)
		require.NoError(b, err, "CreateAuditLog")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListAuditLogs(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListAuditLogs_1000(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-audit-user", "audit@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	for range 1000 {
		err := d.CreateAuditLog(ctx, user.ID, "book.created", "book", "book-1", nil)
		require.NoError(b, err, "CreateAuditLog")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListAuditLogs(ctx, 50, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListGroupMembers ----

// BenchmarkListGroupMembers measures the reading_group_members query that
// uses the composite index idx_reading_group_members_group_joined_at
// (group_id, joined_at) to avoid a temp B-tree sort.
func BenchmarkListGroupMembers_20(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	owner, err := d.CreateUser(ctx, "bench-owner", "owner@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser owner")

	grp, err := d.CreateGroup(ctx, owner.ID, "Bench Group", nil)
	require.NoError(b, err, "CreateGroup")

	for i := range 19 {
		member, err := d.CreateUser(ctx,
			fmt.Sprintf("bench-member-%02d", i),
			fmt.Sprintf("member%02d@example.com", i),
			"hashedpw",
		)
		require.NoError(b, err, "CreateUser member")
		_, err = d.AddGroupMember(ctx, grp.ID, owner.ID, member.ID)
		require.NoError(b, err, "AddGroupMember")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListGroupMembers(ctx, grp.ID, owner.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListAnnotationsForBook ----

// BenchmarkListAnnotationsForBook measures the annotation list query which
// filters by book_id and user membership with an OR+subquery predicate.
// Half the annotations are personal (group_id IS NULL) and half are shared
// via a group the user belongs to, exercising the subquery branch.
func BenchmarkListAnnotationsForBook_10(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-annot-user", "annot@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	other, err := d.CreateUser(ctx, "bench-annot-other", "other@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser other")

	book, err := d.CreateBook(ctx, BookInput{Title: "Bench Book"})
	require.NoError(b, err, "CreateBook")

	grp, err := d.CreateGroup(ctx, other.ID, "Bench Group", nil)
	require.NoError(b, err, "CreateGroup")

	_, err = d.AddGroupMember(ctx, grp.ID, other.ID, user.ID)
	require.NoError(b, err, "AddGroupMember")

	for i := range 10 {
		gid := (*string)(nil)
		if i%2 == 0 {
			gid = &grp.ID // shared annotation — exercises the subquery branch
		}
		_, err := d.CreateAnnotation(ctx, user.ID, book.ID, fmt.Sprintf("annotation %d", i), nil, gid)
		require.NoError(b, err, "CreateAnnotation")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListAnnotationsForBook(ctx, book.ID, user.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListReadingLists ----

// BenchmarkListReadingLists_10 measures the reading-list listing query for a
// user who owns 10 reading lists, each containing 5 books. This exercises the
// book_count aggregation in the SELECT, which previously used a GROUP BY +
// LEFT JOIN and will use a correlated subquery once PR #2533 is merged.
func BenchmarkListReadingLists_10(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-rl-user", "rl@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	books := make([]string, 5)
	for i := range 5 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("RL Book %02d", i+1)})
		require.NoError(b, err, "CreateBook")
		books[i] = bk.ID
	}

	for i := range 10 {
		rl, err := d.CreateReadingList(ctx, user.ID, fmt.Sprintf("Reading List %02d", i+1), nil)
		require.NoError(b, err, "CreateReadingList")
		for _, bookID := range books {
			_, err := d.AddBookToReadingList(ctx, rl.ID, user.ID, bookID)
			require.NoError(b, err, "AddBookToReadingList")
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListReadingLists(ctx, user.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListReadingListBooks ----

// BenchmarkListReadingListBooks_50 measures the paginated book-list query for a
// reading list containing 50 books. This exercises the ORDER BY
// rlb.added_at ASC, b.id ASC sort, which previously required a full filesort
// and will use a composite ordering index on
// (reading_list_id, added_at, book_id) once PR #2499 is merged.
func BenchmarkListReadingListBooks_50(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-rlb-user", "rlb@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	rl, err := d.CreateReadingList(ctx, user.ID, "Bench List", nil)
	require.NoError(b, err, "CreateReadingList")

	for i := range 50 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("RLB Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		_, err = d.AddBookToReadingList(ctx, rl.ID, user.ID, bk.ID)
		require.NoError(b, err, "AddBookToReadingList")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, _, err := d.ListReadingListBooks(ctx, rl.ID, user.ID, 25, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListAuthors ----

// BenchmarkListAuthors_100 measures the full (non-paginated) author-list query
// against a table with 100 rows. The ORDER BY LOWER(name) clause leverages the
// existing idx_authors_name_ci functional index instead of triggering a temp
// B-tree sort.
func BenchmarkListAuthors_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	for i := range 100 {
		_, err := d.CreateAuthor(ctx, fmt.Sprintf("Author %03d", i+1), nil, nil, nil, nil)
		require.NoError(b, err, "CreateAuthor")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListAuthors(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListKoboReadingStatesSince ----

// BenchmarkListKoboReadingStatesSince_All_50 measures the full-sync path of
// ListKoboReadingStatesSince (since=zero), which fetches all 50 reading states
// for a user. This exercises the idx_kobo_reading_states_user_updated composite
// index on (user_id, updated_at) for an ORDER BY updated_at ASC scan.
func BenchmarkListKoboReadingStatesSince_All_50(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-kobo-user", "kobo@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	pct := 0.5
	for i := range 50 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Kobo Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		_, err = d.UpsertKoboReadingState(ctx, user.ID, bk.ID, StatusReading, &pct, nil, nil, nil)
		require.NoError(b, err, "UpsertKoboReadingState")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListKoboReadingStatesSince(ctx, user.ID, time.Time{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListKoboReadingStatesSince_Incremental_50 measures the incremental
// sync path of ListKoboReadingStatesSince (since=non-zero), which returns only
// states updated after the given timestamp. This exercises the
// idx_kobo_reading_states_user_updated composite index on (user_id, updated_at)
// for a range scan with an ORDER BY updated_at ASC filter.
func BenchmarkListKoboReadingStatesSince_Incremental_50(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	user, err := d.CreateUser(ctx, "bench-kobo-incr-user", "koboincr@example.com", "hashedpw")
	require.NoError(b, err, "CreateUser")

	pct := 0.5
	// Seed 50 reading states; use a fixed past timestamp as the sync boundary.
	for i := range 50 {
		bk, err := d.CreateBook(ctx, BookInput{Title: fmt.Sprintf("Kobo Incr Book %05d", i+1)})
		require.NoError(b, err, "CreateBook")
		_, err = d.UpsertKoboReadingState(ctx, user.ID, bk.ID, StatusReading, &pct, nil, nil, nil)
		require.NoError(b, err, "UpsertKoboReadingState")
	}

	// since is set before all seeded states so all 50 rows are returned —
	// a realistic worst-case for a first incremental sync after initial setup.
	since := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListKoboReadingStatesSince(ctx, user.ID, since)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ListSeries ----

// BenchmarkListSeries_100 measures the full (non-paginated) series-list query
// against a table with 100 rows. The ORDER BY LOWER(name) clause leverages the
// existing idx_series_name_ci functional index instead of triggering a temp
// B-tree sort.
func BenchmarkListSeries_100(b *testing.B) {
	d := newBenchDB(b)
	ctx := b.Context()

	for i := range 100 {
		_, err := d.CreateSeries(ctx, fmt.Sprintf("Series %03d", i+1), nil, nil, nil)
		require.NoError(b, err, "CreateSeries")
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := d.ListSeries(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
