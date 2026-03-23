package db

import "context"

// listPaginated is a generic helper that runs a COUNT(*) query against table
// and then fetches a paginated page using the provided columns, ORDER BY
// clause, limit, and offset. scan is called for every row returned by the
// SELECT and must match the column list exactly.
//
// table and columns must be hardcoded SQL identifiers — never pass
// user-supplied input.
func listPaginated[T any](
	ctx context.Context,
	d *DB,
	table, columns, orderBy string,
	limit, offset int,
	scan func(interface{ Scan(...any) error }) (*T, error),
) ([]T, int, error) {
	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := d.QueryContext(ctx,
		`SELECT `+columns+` FROM `+table+` `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}
