package db

import "context"

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
// input into those methods.
func listPaginated[T any, Q paginatedQuery](
	ctx context.Context,
	d *DB,
	query Q,
	limit, offset int,
	scan func(interface{ Scan(...any) error }) (*T, error),
) ([]T, int, error) {
	table := query.table()
	columns := query.columns()
	orderBy := query.orderBy(d)
	items := make([]T, 0)
	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return items, 0, nil
	}

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
