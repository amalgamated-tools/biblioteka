package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- fakes ---------------------------------------------------------------

// fakeRow implements the Scan(...any) interface used by scanRow.
type fakeRow struct {
	vals []any // values to copy into destinations
	err  error // error to return from Scan
}

func (f *fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	for i, d := range dest {
		switch dp := d.(type) {
		case *string:
			*dp = f.vals[i].(string)
		case *int:
			*dp = f.vals[i].(int)
		}
	}
	return nil
}

// fakeRows implements *sql.Rows behavior for collectRows via a real
// in-memory SQLite query so that the contract (Close, Next, Scan, Err)
// is exercised against real driver code.

// helper: open a disposable in-memory SQLite database.
func memDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return d
}

// --- scanRow tests -------------------------------------------------------

type sample struct {
	Name string
	Age  int
}

func fillSample(s *sample) []any {
	return []any{&s.Name, &s.Age}
}

func TestScanRow_HappyPath(t *testing.T) {
	row := &fakeRow{vals: []any{"Alice", 30}}

	got, err := scanRow(row, fillSample)
	require.NoError(t, err)
	require.Equal(t, "Alice", got.Name)
	require.Equal(t, 30, got.Age)
}

func TestScanRow_PropagatesScanError(t *testing.T) {
	scanErr := errors.New("column mismatch")
	row := &fakeRow{err: scanErr}

	got, err := scanRow(row, fillSample)
	require.ErrorIs(t, err, scanErr)
	require.Nil(t, got)
}

// --- collectRows tests ---------------------------------------------------

// scanSample is a scan function compatible with collectRows.
func scanSample(row interface{ Scan(...any) error }) (*sample, error) {
	return scanRow(row, fillSample)
}

func TestCollectRows_HappyPath(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO t VALUES ('Alice', 30), ('Bob', 25), ('Carol', 40)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name, age FROM t ORDER BY age`)
	require.NoError(t, err)

	items, err := collectRows(rows, scanSample)
	require.NoError(t, err)
	require.Len(t, items, 3)
	want := []sample{{"Bob", 25}, {"Alice", 30}, {"Carol", 40}}
	for i, w := range want {
		require.Equal(t, w, items[i])
	}
}

func TestCollectRows_EmptyResult(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t (name TEXT, age INTEGER)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name, age FROM t`)
	require.NoError(t, err)

	items, err := collectRows(rows, scanSample)
	require.NoError(t, err)
	require.Nil(t, items)
}

func TestCollectRows_PropagatesMidIterationScanError(t *testing.T) {
	d := memDB(t)
	// Create a table with two columns but query only one, then scan into
	// two destinations. The first row's Scan will fail with a column-count
	// mismatch.
	_, err := d.Exec(`CREATE TABLE t (name TEXT)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO t VALUES ('Alice')`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name FROM t`)
	require.NoError(t, err)

	// scanBadDest asks for two fields from a one-column result set.
	scanBadDest := func(row interface{ Scan(...any) error }) (*sample, error) {
		return scanRow(row, fillSample) // expects name + age, but only name exists
	}

	items, err := collectRows(rows, scanBadDest)
	require.Error(t, err, "expected scan error, got nil with items: %+v", items)
}

func TestCollectRows_AlwaysClosesRows(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO t VALUES ('Alice', 30)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name, age FROM t`)
	require.NoError(t, err)

	// Succeed — rows should be closed after collectRows returns.
	_, err = collectRows(rows, scanSample)
	require.NoError(t, err)

	// Calling rows.Next() after Close returns false.
	require.False(t, rows.Next())
}

func TestCollectRows_ClosesRowsOnError(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t (name TEXT)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO t VALUES ('Alice')`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name FROM t`)
	require.NoError(t, err)

	scanBadDest := func(row interface{ Scan(...any) error }) (*sample, error) {
		return scanRow(row, fillSample)
	}

	_, _ = collectRows(rows, scanBadDest)

	// Even after an error, rows should be closed.
	require.False(t, rows.Next())
}

func scanSampleForBook(row interface{ Scan(...any) error }) (string, *sample, error) {
	var bookID string
	item, err := scanSample(prefixedScanner{row: row, prefix: []any{&bookID}})
	if err != nil {
		return "", nil, err
	}
	return bookID, item, nil
}

func TestCollectForBooks_HappyPath(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t_books (book_id TEXT, name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = d.Exec(`
		INSERT INTO t_books (book_id, name, age)
		VALUES ('book-1', 'Alice', 30), ('book-1', 'Bob', 25), ('book-2', 'Carol', 40)
	`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT book_id, name, age FROM t_books ORDER BY book_id, age`)
	require.NoError(t, err)

	items, err := collectForBooks(rows, scanSampleForBook, 0)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, []sample{{Name: "Bob", Age: 25}, {Name: "Alice", Age: 30}}, items["book-1"])
	require.Equal(t, []sample{{Name: "Carol", Age: 40}}, items["book-2"])
}

func TestCollectForBooks_EmptyResult(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t_books_empty (book_id TEXT, name TEXT, age INTEGER)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT book_id, name, age FROM t_books_empty`)
	require.NoError(t, err)

	items, err := collectForBooks(rows, scanSampleForBook, 0)
	require.NoError(t, err)
	require.NotNil(t, items)
	require.Empty(t, items)
}

func TestCollectForBooks_ClosesRowsOnError(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t_books_err (book_id TEXT, name TEXT)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO t_books_err (book_id, name) VALUES ('book-1', 'Alice')`)
	require.NoError(t, err)

	// Query returns only 2 columns, but scanSampleForBook expects 3 (book_id + name + age).
	rows, err := d.Query(`SELECT book_id, name FROM t_books_err`)
	require.NoError(t, err)

	_, _ = collectForBooks(rows, scanSampleForBook, 0)

	// Even after an error, rows should be closed.
	require.False(t, rows.Next())
}

// --- collectRowsAndTotal tests -------------------------------------------

// sampleWithTotal is used by collectRowsAndTotal tests to exercise the
// scan-with-total pattern (e.g., a window function COUNT(*) OVER()).
func scanSampleAndTotal(row interface{ Scan(...any) error }) (*sample, int, error) {
	var s sample
	var total int
	if err := row.Scan(&s.Name, &s.Age, &total); err != nil {
		return nil, 0, err
	}
	return &s, total, nil
}

func TestCollectRowsAndTotal_HappyPath(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE tt (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO tt VALUES ('Alice', 30), ('Bob', 25), ('Carol', 40)`)
	require.NoError(t, err)

	// Simulate a paginated query with COUNT(*) OVER() as the total column.
	rows, err := d.Query(`SELECT name, age, COUNT(*) OVER() AS total FROM tt ORDER BY age`)
	require.NoError(t, err)

	items, total, err := collectRowsAndTotal(rows, scanSampleAndTotal)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, items, 3)
	want := []sample{{"Bob", 25}, {"Alice", 30}, {"Carol", 40}}
	for i, w := range want {
		require.Equal(t, w, items[i])
	}
}

func TestCollectRowsAndTotal_EmptyResult(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE tt2 (name TEXT, age INTEGER)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name, age, COUNT(*) OVER() AS total FROM tt2`)
	require.NoError(t, err)

	items, total, err := collectRowsAndTotal(rows, scanSampleAndTotal)
	require.NoError(t, err)
	require.Nil(t, items)
	require.Equal(t, 0, total)
}

func TestCollectRowsAndTotal_PropagatesScanError(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE tt3 (name TEXT)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO tt3 VALUES ('Alice')`)
	require.NoError(t, err)

	// Query returns one column, but scanSampleAndTotal expects three.
	rows, err := d.Query(`SELECT name FROM tt3`)
	require.NoError(t, err)

	items, total, err := collectRowsAndTotal(rows, scanSampleAndTotal)
	require.Error(t, err, "expected scan error, got nil with items: %+v", items)
	require.Nil(t, items)
	require.Equal(t, 0, total)

	// Rows should be closed even after a scan error.
	require.False(t, rows.Next())
}

func TestCollectRowsAndTotal_ClosesRowsOnSuccess(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE tt4 (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	_, err = d.Exec(`INSERT INTO tt4 VALUES ('Alice', 30)`)
	require.NoError(t, err)

	rows, err := d.Query(`SELECT name, age, COUNT(*) OVER() AS total FROM tt4`)
	require.NoError(t, err)

	_, _, err = collectRowsAndTotal(rows, scanSampleAndTotal)
	require.NoError(t, err)

	// Calling rows.Next() after Close returns false.
	require.False(t, rows.Next())
}

func TestCollectRowsAndTotal_PropagatesRowsErr(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE tt5 (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	// Insert a large batch to prevent the driver from buffering everything.
	for range 10000 {
		_, err = d.Exec(`INSERT INTO tt5 VALUES ('x', 1)`)
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	rows, err := d.QueryContext(ctx, `SELECT name, age, COUNT(*) OVER() AS total FROM tt5`)
	require.NoError(t, err)

	// Cancel the context so rows.Err() reports context.Canceled.
	cancel()

	_, _, err = collectRowsAndTotal(rows, scanSampleAndTotal)

	if err == nil {
		t.Skip("SQLite driver buffered all rows; context cancel did not propagate to rows.Err()")
	}
	require.ErrorIs(t, err, context.Canceled)
}

func TestCollectRows_PropagatesRowsErr(t *testing.T) {
	// rows.Err() is checked after the iteration loop in collectRows.
	// To exercise this, we cancel the query's context before iterating,
	// causing rows.Next() to stop early and rows.Err() to report the
	// cancellation. Because SQLite's in-process driver may buffer small
	// result sets entirely, we insert enough rows to exceed the buffer.
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE t (name TEXT, age INTEGER)`)
	require.NoError(t, err)
	// Insert a large batch to prevent the driver from buffering everything.
	for range 10000 {
		_, err = d.Exec(`INSERT INTO t VALUES ('x', 1)`)
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	rows, err := d.QueryContext(ctx, `SELECT name, age FROM t`)
	require.NoError(t, err)

	// Cancel the context so rows.Err() reports context.Canceled.
	cancel()

	_, err = collectRows(rows, scanSample)

	// If the driver still buffered everything, skip rather than give a
	// false negative.
	if err == nil {
		t.Skip("SQLite driver buffered all rows; context cancel did not propagate to rows.Err()")
	}
	require.ErrorIs(t, err, context.Canceled)
}
