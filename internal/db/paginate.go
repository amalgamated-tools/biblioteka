package db

import (
	"context"
	"fmt"
)

// allowedPaginatedTables is the set of tables that listPaginated may query.
// Any table not in this set is rejected at runtime to prevent accidental SQL
// injection if a caller ever passes a dynamic value.
var allowedPaginatedTables = map[string]bool{
	"authors": true,
	"series":  true,
}

type paginatedQuery interface {
	table() string
	columns() string
	orderBy(*DB) string
}

// listPaginated is a generic helper that runs a COUNT(*) query against table
// and then fetches a paginated page using the provided columns, ORDER BY
// clause, limit, and offset. scan is called for every row returned by the
// SELECT and must match the column list exactly.
//
// query must be a package-defined type whose methods return hardcoded SQL
// identifiers and dialect-derived ORDER BY clauses. Never pass user-supplied
// input into those methods. table is additionally validated against
// allowedPaginatedTables at runtime.
func listPaginated[T any](
	ctx context.Context,
	d *DB,
	query paginatedQuery,
	limit, offset int,
	scan func(interface{ Scan(...any) error }) (*T, error),
) ([]T, int, error) {
	if limit <= 0 {
		return make([]T, 0), 0, nil
	}
	if offset < 0 {
		offset = 0
	}

	table := query.table()

	if !allowedPaginatedTables[table] {
		return nil, 0, fmt.Errorf("listPaginated: unknown table %q", table)
	}

	columns := query.columns()
	orderBy := query.orderBy(d)
	items := make([]T, 0, limit)

	var total int
	// safe: table validated against allowedPaginatedTables above
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return items, 0, nil
	}

	// safe: table, columns, and orderBy are hardcoded caller-provided identifiers
	rows, err := d.QueryContext(ctx,
		`SELECT `+columns+` FROM `+table+` `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}
