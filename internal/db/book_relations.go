package db

import (
	"context"
	"log/slog"
	"strings"

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
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		a, err := scanAuthor(ctx, rows)
		if err != nil {
			return nil, err
		}
		authors = append(authors, *a)
	}
	return authors, rows.Err()
}

// SetBookAuthors replaces all author associations for a book.
// Duplicate author IDs are silently deduplicated.
func (d *DB) SetBookAuthors(ctx context.Context, bookID string, authorIDs []string) error {
	slog.DebugContext(ctx, "db: setting book authors",
		slog.String(otelkeys.BookID, bookID),
		slog.Int(otelkeys.AuthorCount, len(authorIDs)),
	)
	seen := make(map[string]struct{}, len(authorIDs))
	unique := make([]string, 0, len(authorIDs))
	for _, id := range authorIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM book_authors WHERE book_id = $1`, bookID); err != nil {
		return err
	}

	for _, authorID := range unique {
		if _, err := tx.ExecContext(ctx, `INSERT INTO book_authors (book_id, author_id) VALUES ($1, $2)`, bookID, authorID); err != nil {
			return err
		}
	}

	return tx.Commit()
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
	defer rows.Close()

	var entries []BookSeriesEntry
	for rows.Next() {
		var entry BookSeriesEntry
		err := rows.Scan(&entry.Series.ID, &entry.Series.Name, &entry.Series.GoodreadsID, &entry.Series.HardcoverID, &entry.Series.GoogleBooksID, &entry.Series.CreatedAt, &entry.Series.UpdatedAt, &entry.Position)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book series entry", slog.Any(otelkeys.Error, err))
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// SetBookSeries replaces all series associations for a book.
// Duplicate series IDs are silently deduplicated (last position wins).
func (d *DB) SetBookSeries(ctx context.Context, bookID string, entries []BookSeriesInput) error {
	slog.DebugContext(ctx, "db: setting book series",
		slog.String(otelkeys.BookID, bookID),
		slog.Int(otelkeys.SeriesCount, len(entries)),
	)
	seen := make(map[string]struct{}, len(entries))
	unique := make([]BookSeriesInput, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		if _, ok := seen[entries[i].SeriesID]; !ok {
			seen[entries[i].SeriesID] = struct{}{}
			unique = append(unique, entries[i])
		}
	}

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `DELETE FROM book_series WHERE book_id = $1`, bookID); err != nil {
		return err
	}

	for _, entry := range unique {
		if _, err := tx.ExecContext(ctx, `INSERT INTO book_series (book_id, series_id, position) VALUES ($1, $2, $3)`, bookID, entry.SeriesID, entry.Position); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAuthorsForBooks returns authors grouped by book ID for the given book IDs.
func (d *DB) GetAuthorsForBooks(ctx context.Context, bookIDs []string) (map[string][]Author, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching authors for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	placeholders := make([]string, len(bookIDs))
	args := make([]any, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = dollarN(i + 1)
		args[i] = id
	}

	rows, err := d.QueryContext(ctx,
		`SELECT ba.book_id, a.id, a.name, a.goodreads_id, a.hardcover_id, a.google_books_id, a.image_url, a.created_at, a.updated_at
		FROM authors a INNER JOIN book_authors ba ON ba.author_id = a.id
		WHERE ba.book_id IN (`+strings.Join(placeholders, ",")+`)
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
		a, err := scanAuthor(ctx, prefixedScanner{row: rows, prefix: []any{&bookID}})
		if err != nil {
			return nil, err
		}
		result[bookID] = append(result[bookID], *a)
	}
	return result, rows.Err()
}
