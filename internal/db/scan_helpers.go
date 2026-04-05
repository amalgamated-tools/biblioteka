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
	defer rows.Close()
	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
