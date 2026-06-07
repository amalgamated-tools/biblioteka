package db

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// prefixedScanner wraps a sql.Rows and prepends extra scan destinations
// before delegating to a helper like scanAuthor. This lets batch queries
// that select a leading book_id column reuse the same scan helper as
// single-row queries.
type prefixedScanner struct {
	row    interface{ Scan(...any) error }
	prefix []any
}

func (p prefixedScanner) Scan(dest ...any) error {
	all := make([]any, 0, len(p.prefix)+len(dest))
	all = append(all, p.prefix...)
	all = append(all, dest...)
	return p.row.Scan(all...)
}

// BookSeriesEntry represents a book's membership in a series with its position.
type BookSeriesEntry struct {
	Series   Series   `json:"series"`
	Position *float64 `json:"position"`
}

// BookSeriesInput represents input for setting a book's series membership.
type BookSeriesInput struct {
	SeriesID string   `json:"series_id"`
	Position *float64 `json:"position"`
}

// GetBookAuthors returns all authors for a book.
func (d *DB) GetBookAuthors(ctx context.Context, bookID string) ([]Author, error) {
	slog.DebugContext(ctx, "db: fetching book authors", slog.String(otelkeys.BookID, bookID))
	rows, err := d.QueryContext(ctx,
		`SELECT a.id, a.name, a.goodreads_id, a.hardcover_id, a.google_books_id, a.image_url, a.created_at, a.updated_at FROM authors a INNER JOIN book_authors ba ON ba.author_id = a.id WHERE ba.book_id = $1 ORDER BY a.name ASC`,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanAuthor)
}

// SetBookAuthors replaces all author associations for a book.
// Duplicate author IDs are silently deduplicated.
func (d *DB) SetBookAuthors(ctx context.Context, bookID string, authorIDs []string) error {
	slog.DebugContext(ctx, "db: setting book authors",
		slog.String(otelkeys.BookID, bookID),
		slog.Int(otelkeys.AuthorCount, len(authorIDs)),
	)
	return d.setBookStringIDs(
		ctx,
		bookID,
		authorIDs,
		`DELETE FROM book_authors WHERE book_id = $1`,
		`INSERT INTO book_authors (book_id, author_id) VALUES ($1, $2)`,
	)
}

// GetBookSeries returns all series entries for a book.
func (d *DB) GetBookSeries(ctx context.Context, bookID string) ([]BookSeriesEntry, error) {
	slog.DebugContext(ctx, "db: fetching book series", slog.String(otelkeys.BookID, bookID))
	rows, err := d.QueryContext(ctx,
		`SELECT s.id, s.name, s.goodreads_id, s.hardcover_id, s.google_books_id, s.created_at, s.updated_at, bs.position FROM series s INNER JOIN book_series bs ON bs.series_id = s.id WHERE bs.book_id = $1 ORDER BY s.name ASC`,
		bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBookSeriesEntry)
}

// scanBookSeriesEntry scans a book series entry row into a BookSeriesEntry struct.
func scanBookSeriesEntry(row interface{ Scan(...any) error }) (*BookSeriesEntry, error) {
	return scanRow(row, func(entry *BookSeriesEntry) []any {
		return []any{&entry.Series.ID, &entry.Series.Name, &entry.Series.GoodreadsID, &entry.Series.HardcoverID, &entry.Series.GoogleBooksID, &entry.Series.CreatedAt, &entry.Series.UpdatedAt, &entry.Position}
	})
}

// setBookStringIDs atomically replaces all string-ID associations for a book.
func (d *DB) setBookStringIDs(ctx context.Context, bookID string, entityIDs []string, deleteQuery, insertQuery string) error {
	seen := make(map[string]struct{}, len(entityIDs))
	unique := make([]string, 0, len(entityIDs))
	for _, id := range entityIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}

	return d.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, deleteQuery, bookID); err != nil {
			return err
		}

		for _, entityID := range unique {
			if _, err := tx.ExecContext(ctx, insertQuery, bookID, entityID); err != nil {
				return err
			}
		}

		return nil
	})
}

// SetBookSeries replaces all series associations for a book.
// Duplicate series IDs are silently deduplicated (last position wins).
// This intentionally does not use setBookStringIDs because it stores positions
// and applies last-position-wins deduplication.
func (d *DB) SetBookSeries(ctx context.Context, bookID string, entries []BookSeriesInput) error {
	slog.DebugContext(ctx, "db: setting book series",
		slog.String(otelkeys.BookID, bookID),
		slog.Int(otelkeys.SeriesCount, len(entries)),
	)
	seen := make(map[string]struct{}, len(entries))
	unique := make([]BookSeriesInput, 0, len(entries))
	for _, v := range slices.Backward(entries) {
		if _, ok := seen[v.SeriesID]; !ok {
			seen[v.SeriesID] = struct{}{}
			unique = append(unique, v)
		}
	}

	return d.WithTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM book_series WHERE book_id = $1`, bookID); err != nil {
			return err
		}

		for _, entry := range unique {
			if _, err := tx.ExecContext(ctx, `INSERT INTO book_series (book_id, series_id, position) VALUES ($1, $2, $3)`, bookID, entry.SeriesID, entry.Position); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetAuthorsForBooks returns authors grouped by book ID for the given book IDs.
func (d *DB) GetAuthorsForBooks(ctx context.Context, bookIDs []string) (map[string][]Author, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching authors for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	inClause, args := buildInClause(bookIDs, 1)

	rows, err := d.QueryContext(ctx,
		`SELECT ba.book_id, a.id, a.name, a.goodreads_id, a.hardcover_id, a.google_books_id, a.image_url, a.created_at, a.updated_at
		FROM authors a INNER JOIN book_authors ba ON ba.author_id = a.id
		WHERE ba.book_id IN (`+inClause+`)
		ORDER BY a.name ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Author, len(bookIDs))
	for rows.Next() {
		var bookID string
		a, err := scanAuthor(prefixedScanner{row: rows, prefix: []any{&bookID}})
		if err != nil {
			return nil, err
		}
		result[bookID] = append(result[bookID], *a)
	}
	return result, rows.Err()
}
