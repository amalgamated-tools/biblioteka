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
	if len(results) != 0 {
		t.Errorf("len = %d, want 0", len(results))
	}
}

func TestListAll_WithData(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Zoe Author", "Alice Author", "Midge Author"} {
		if _, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateAuthor(%q)", name)
		}
	}

	results, err := listAll(t.Context(), d, authorListQuery{}, scanAuthor)
	require.NoError(t, err, "listAll(authors) error")
	if len(results) != 3 {
		require.Failf(t, "failed", "len = %d, want 3", len(results))
	}
	// Alphabetical order: Alice, Midge, Zoe.
	if results[0].Name != "Alice Author" {
		t.Errorf("results[0].Name = %q, want Alice Author", results[0].Name)
	}
	if results[1].Name != "Midge Author" {
		t.Errorf("results[1].Name = %q, want Midge Author", results[1].Name)
	}
	if results[2].Name != "Zoe Author" {
		t.Errorf("results[2].Name = %q, want Zoe Author", results[2].Name)
	}
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
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(results) != 0 {
		t.Errorf("len = %d, want 0", len(results))
	}
	// Must return an empty non-nil slice.
	if results == nil {
		t.Error("results = nil, want empty slice")
	}
}

func TestListPaginated_FirstPage(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author A", "Author B", "Author C", "Author D", "Author E"} {
		if _, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateAuthor(%q)", name)
		}
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 2, 0, scanAuthor)
	require.NoError(t, err, "listPaginated(page1) error")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(results) != 2 {
		require.Failf(t, "failed", "len = %d, want 2", len(results))
	}
	if results[0].Name != "Author A" {
		t.Errorf("results[0].Name = %q, want Author A", results[0].Name)
	}
}

func TestListPaginated_SecondPage(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author A", "Author B", "Author C", "Author D", "Author E"} {
		if _, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateAuthor(%q)", name)
		}
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 2, 2, scanAuthor)
	require.NoError(t, err, "listPaginated(page2) error")
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(results) != 2 {
		require.Failf(t, "failed", "len = %d, want 2", len(results))
	}
	if results[0].Name != "Author C" {
		t.Errorf("results[0].Name = %q, want Author C", results[0].Name)
	}
}

func TestListPaginated_NegativeOffsetClampedToZero(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Author X", "Author Y"} {
		if _, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateAuthor(%q)", name)
		}
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 10, -5, scanAuthor)
	require.NoError(t, err, "listPaginated(offset=-5) error")
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	// Negative offset clamped to 0 → both rows returned.
	if len(results) != 2 {
		t.Errorf("len = %d, want 2 (negative offset treated as 0)", len(results))
	}
}

func TestListPaginated_ZeroLimitReturnsTotal(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"Alpha", "Beta"} {
		if _, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil); err != nil {
			require.NoError(t, err, "CreateAuthor(%q)", name)
		}
	}

	results, total, err := listPaginated(t.Context(), d, authorListQuery{}, 0, 0, scanAuthor)
	require.NoError(t, err, "listPaginated(limit=0) error")
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(results) != 0 {
		t.Errorf("len = %d, want 0 for limit=0", len(results))
	}
	if results == nil {
		t.Error("results = nil, want empty slice")
	}
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
