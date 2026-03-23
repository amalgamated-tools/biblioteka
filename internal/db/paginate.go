package db

import (
	"context"
	"fmt"
)

// listPaginated is a generic helper that runs a COUNT(*) query against table
// and then fetches a paginated page using the provided columns, ORDER BY
// clause, limit, and offset. scan is called for every row returned by the
// SELECT and must match the column list exactly.
//
// table, columns, and orderBy must come from package-defined constants or
// helpers such as dialectOrderBy. Never pass user-supplied input.
func listPaginated[T any](
	ctx context.Context,
	d *DB,
	table, columns, orderBy string,
	limit, offset int,
	scan func(interface{ Scan(...any) error }) (*T, error),
) ([]T, int, error) {
	if err := validatePaginatedQuery(d, table, columns, orderBy); err != nil {
		return nil, 0, err
	}

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

func validatePaginatedQuery(d *DB, table, columns, orderBy string) error {
	switch {
	case table == "authors" && columns == authorColumns && orderBy == d.dialectOrderBy("name", "ASC"):
		return nil
	case table == "series" && columns == seriesColumns && orderBy == d.dialectOrderBy("name", "ASC"):
		return nil
	default:
		return fmt.Errorf("listPaginated: unsupported query parts for table %q", table)
	}
}
