package db

import "database/sql"

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

// collectRowsGrouped iterates rows, scans each one using scan (which returns a
// grouping key), and returns a map of key -> []T.
//
// It always closes rows before returning.
func collectRowsGrouped[T any](
	rows *sql.Rows,
	scan func(interface{ Scan(...any) error }) (string, *T, error),
	cap int,
) (map[string][]T, error) {
	defer rows.Close()

	var result map[string][]T
	if cap > 0 {
		result = make(map[string][]T, cap)
	} else {
		result = make(map[string][]T)
	}

	for rows.Next() {
		key, item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result[key] = append(result[key], *item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
