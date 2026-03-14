package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
)

var ErrSeriesNameExists = errors.New("series name already exists")

type Series struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	GoodreadsID   *string   `json:"goodreads_id"`
	HardcoverID   *string   `json:"hardcover_id"`
	GoogleBooksID *string   `json:"google_books_id"`
	CreatedAt     Timestamp `json:"created_at"`
	UpdatedAt     Timestamp `json:"updated_at"`
}

const seriesColumns = `id, name, goodreads_id, hardcover_id, google_books_id, created_at, updated_at`

func scanSeries(row interface{ Scan(...any) error }) (*Series, error) {
	var s Series
	err := row.Scan(&s.ID, &s.Name, &s.GoodreadsID, &s.HardcoverID, &s.GoogleBooksID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateSeries(ctx context.Context, name string, goodreadsID, hardcoverID, googleBooksID *string) (*Series, error) {
	slog.DebugContext(ctx, "db: creating series", slog.String("name", name))
	s, err := scanSeries(d.QueryRowContext(ctx,
		`INSERT INTO series (name, goodreads_id, hardcover_id, google_books_id) VALUES ($1, $2, $3, $4) RETURNING `+seriesColumns,
		name, goodreadsID, hardcoverID, googleBooksID,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSeriesNameExists
		}
		return nil, err
	}
	return s, nil
}

func (d *DB) GetSeries(ctx context.Context, id string) (*Series, error) {
	slog.DebugContext(ctx, "db: fetching series", slog.String("id", id))
	return scanSeries(d.QueryRowContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE id = $1`,
		id,
	))
}

func (d *DB) ListSeries(ctx context.Context) ([]Series, error) {
	slog.DebugContext(ctx, "db: listing series")
	orderBy := "ORDER BY name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY name ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+seriesColumns+` FROM series `+orderBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *s)
	}
	return list, rows.Err()
}

func (d *DB) UpdateSeries(ctx context.Context, id, name string, goodreadsID, hardcoverID, googleBooksID *string) (*Series, error) {
	slog.DebugContext(ctx, "db: updating series", slog.String("id", id), slog.String("name", name))
	s, err := scanSeries(d.QueryRowContext(ctx,
		`UPDATE series SET name = $1, goodreads_id = $2, hardcover_id = $3, google_books_id = $4, updated_at = `+d.now()+` WHERE id = $5 RETURNING `+seriesColumns,
		name, goodreadsID, hardcoverID, googleBooksID, id,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSeriesNameExists
		}
		return nil, err
	}
	return s, nil
}

func (d *DB) DeleteSeries(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting series", slog.String("id", id))
	res, err := d.ExecContext(ctx, `DELETE FROM series WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
