package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// invalidListQuery implements listQuery with a table name that is not in
// allowedListTables, used to test the security whitelist.
type invalidListQuery struct{}

func (invalidListQuery) table() string        { return "nonexistent_table" }
func (invalidListQuery) columns() string      { return "id" }
func (invalidListQuery) orderBy(_ *DB) string { return "ORDER BY id ASC" }

// ---- listAll ----

func TestListAll_EmptyTable(t *testing.T) {
	d := newTestDB(t)

	results, err := listAll(t.Context(), d, authorListQuery{}, scanAuthor)
	require.NoError(t, err, "listAll(authors, empty) error")
	require.Len(t, results, 0)
}

func TestListAll_WithData(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Zoe Author", "Alice Author", "Midge Author"} {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q)", name)
	}

	results, err := listAll(t.Context(), d, authorListQuery{}, scanAuthor)
	require.NoError(t, err, "listAll(authors) error")
	require.Len(t, results, 3)
	// Alphabetical order: Alice, Midge, Zoe.
	require.Equal(t, "Alice Author", results[0].Name)
	require.Equal(t, "Midge Author", results[1].Name)
	require.Equal(t, "Zoe Author", results[2].Name)
}

func TestListAll_InvalidTableRejected(t *testing.T) {
	d := newTestDB(t)

	_, err := listAll(t.Context(), d, invalidListQuery{}, func(row interface{ Scan(...any) error }) (*struct{}, error) {
		return &struct{}{}, nil
	})
	require.Error(t, err, "expected error for unknown table, got nil")
}

// ---- listPaginated ----

func TestListPaginated_EmptyTable(t *testing.T) {
	d := newTestDB(t)

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 10, 0, scanAuthor)
	require.NoError(t, err, "listPaginated(empty) error")
	require.Equal(t, 0, total)
	require.Len(t, results, 0)
	// Must return an empty non-nil slice.
	require.NotNil(t, results)
}

func TestListPaginated_FirstPage(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author A", "Author B", "Author C", "Author D", "Author E"} {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q)", name)
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 2, 0, scanAuthor)
	require.NoError(t, err, "listPaginated(page1) error")
	require.Equal(t, 5, total)
	require.Len(t, results, 2)
	require.Equal(t, "Author A", results[0].Name)
}

func TestListPaginated_SecondPage(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author A", "Author B", "Author C", "Author D", "Author E"} {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q)", name)
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 2, 2, scanAuthor)
	require.NoError(t, err, "listPaginated(page2) error")
	require.Equal(t, 5, total)
	require.Len(t, results, 2)
	require.Equal(t, "Author C", results[0].Name)
}

func TestListPaginated_NegativeOffsetClampedToZero(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author X", "Author Y"} {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q)", name)
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 10, -5, scanAuthor)
	require.NoError(t, err, "listPaginated(offset=-5) error")
	require.Equal(t, 2, total)
	// Negative offset clamped to 0 → both rows returned.
	require.Len(t, results, 2)
}

func TestListPaginated_ZeroLimitReturnsTotal(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Alpha", "Beta"} {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q)", name)
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 0, 0, scanAuthor)
	require.NoError(t, err, "listPaginated(limit=0) error")
	require.Equal(t, 2, total)
	require.Len(t, results, 0)
	require.NotNil(t, results)
}

// ---- countFallback ----

func TestCountFallback_OffsetExceeded(t *testing.T) {
	d := newTestDB(t)
	_, err := d.CreateBook(t.Context(), BookInput{Title: "Only Book"})
	require.NoError(t, err)

	var total int
	err = countFallback(t.Context(), d, &total, 0, 10, `SELECT COUNT(*) FROM books`)
	require.NoError(t, err)
	require.Equal(t, 1, total, "countFallback should report the actual total when offset exceeds row count")
}

func TestCountFallback_NoOpWhenRowsReturned(t *testing.T) {
	d := newTestDB(t)
	var total int
	// rowCount > 0 — fallback must not fire, total stays 0
	err := countFallback(t.Context(), d, &total, 3, 0, `SELECT COUNT(*) FROM books`)
	require.NoError(t, err)
	require.Equal(t, 0, total, "countFallback should be a no-op when rows were returned")
}

func TestCountFallback_NoOpWhenOffsetIsZero(t *testing.T) {
	d := newTestDB(t)
	var total int
	// rowCount == 0 AND offset == 0 — empty table, fallback must not fire
	err := countFallback(t.Context(), d, &total, 0, 0, `SELECT COUNT(*) FROM books`)
	require.NoError(t, err)
	require.Equal(t, 0, total, "countFallback should be a no-op when offset is zero")
}

func TestListPaginated_InvalidTableRejected(t *testing.T) {
	d := newTestDB(t)

	_, _, err := listPaginated(t.Context(), d, invalidListQuery{}, 10, 0,
		func(row interface{ Scan(...any) error }) (*struct{}, error) {
			return &struct{}{}, nil
		},
	)
	require.Error(t, err, "expected error for unknown table, got nil")
}
