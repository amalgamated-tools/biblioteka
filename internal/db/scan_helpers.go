package db

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// scanRow creates a new T, passes a pointer to it to fill (which returns the
// destination pointers for row.Scan), and returns a pointer to the populated
// value.
//
// This eliminates the per-entity scan boilerplate:
//
//	func scanFoo(row interface{ Scan(...any) error }) (*Foo, error) {
//	    var f Foo
//	    if err := row.Scan(&f.X, &f.Y, ...); err != nil {
//	        return nil, err
//	    }
//	    return &f, nil
//	}
//
// which can now be written as:
//
//	func scanFoo(row interface{ Scan(...any) error }) (*Foo, error) {
//	    return scanRow(row, func(f *Foo) []any {
//	        return []any{&f.X, &f.Y, ...}
//	    })
//	}
//
// For the rows-iteration pattern used in List* functions, see collectRows.
// For the find-or-create pattern used in FindOrCreate* functions, see findOrCreate.
func scanRow[T any](row interface{ Scan(...any) error }, fill func(*T) []any) (*T, error) {
	var v T
	if err := row.Scan(fill(&v)...); err != nil {
		return nil, err
	}
	return &v, nil
}

// collectRows iterates rows, scans each one using scan, and returns the
// collected slice. It always closes rows before returning.
//
// This eliminates the per-entity rows-iteration boilerplate:
//
//	defer rows.Close()
//	var items []T
//	for rows.Next() {
//	    item, err := scan(rows)
//	    if err != nil {
//	        return nil, err
//	    }
//	    items = append(items, *item)
//	}
//	return items, rows.Err()
//
// which can now be written as:
//
//	return collectRows(rows, scanFoo)
func collectRows[T any](rows *sql.Rows, scan func(interface{ Scan(...any) error }) (*T, error)) ([]T, error) {
	return collectRowsWithCap(rows, scan, 0)
}

// collectRowsWithCap is like collectRows but pre-allocates the result slice
// with the given capacity hint. Use this on hot paths where the maximum number
// of rows is known in advance (e.g., paginated queries with a known limit).
func collectRowsWithCap[T any](rows *sql.Rows, scan func(interface{ Scan(...any) error }) (*T, error), cap int) ([]T, error) {
	defer rows.Close()
	var items []T
	if cap > 0 {
		items = make([]T, 0, cap)
	}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// collectRowsAndTotal iterates rows, scans each one using scan (which also
// returns a window-function total), and returns the collected slice and the
// total seen on the last row. It always closes rows before returning.
//
// This eliminates the per-entity rows-iteration boilerplate for paginated
// queries that embed a COUNT(*) OVER() column:
//
//	defer rows.Close()
//	var items []T
//	var total int
//	for rows.Next() {
//	    item, t, err := scan(rows)
//	    if err != nil {
//	        return nil, 0, err
//	    }
//	    total = t
//	    items = append(items, *item)
//	}
//	if err := rows.Err(); err != nil {
//	    return nil, 0, err
//	}
//
// which can now be written as:
//
//	return collectRowsAndTotal(rows, scanFooAndTotal)
func collectRowsAndTotal[T any](rows *sql.Rows, scan func(interface{ Scan(...any) error }) (*T, int, error)) ([]T, int, error) {
	defer rows.Close()
	var items []T
	var total int
	for rows.Next() {
		item, t, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		total = t
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// collectRowsGrouped iterates rows, calling scan for each row. scan must
// return a string group key, a pointer to the scanned value, and an error.
// Results are accumulated into a map keyed by the returned string. It always
// closes rows before returning.
//
// This is the grouped variant of collectRows, used by batch-fetch helpers
// such as batchFetchByBookID that group results by an association key.
func collectRowsGrouped[T any](
	rows *sql.Rows,
	scan func(interface{ Scan(...any) error }) (string, *T, error),
	mapCap int,
) (map[string][]T, error) {
	defer rows.Close()

	var result map[string][]T
	if mapCap > 0 {
		result = make(map[string][]T, mapCap)
	} else {
		result = make(map[string][]T)
	}

	for rows.Next() {
		key, v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		if v == nil {
			continue
		}
		result[key] = append(result[key], *v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// batchFetchByBookID is the shared skeleton for all GetXForBooks methods.
// It handles the early-return, debug logging, IN-clause construction, query
// execution, and grouped collection, delegating the SQL string and per-row
// scanning to the caller.
//
// buildQuery receives the IN-clause placeholder string (e.g. "$1,$2,$3") and
// must return the full SQL statement to execute.
//
// scan receives each *sql.Rows and must return the book ID that the row
// belongs to, a pointer to the scanned value, and an error.
func batchFetchByBookID[T any](
	ctx context.Context,
	d *DB,
	bookIDs []string,
	logMsg string,
	buildQuery func(inClause string) string,
	scan func(interface{ Scan(...any) error }) (string, *T, error),
) (map[string][]T, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, logMsg, slog.Int(otelkeys.BookCount, len(bookIDs)))

	inClause, args := buildInClause(bookIDs, 1)

	rows, err := d.QueryContext(ctx, buildQuery(inClause), args...)
	if err != nil {
		return nil, err
	}
	return collectRowsGrouped(rows, scan, len(bookIDs))
}
