package db

import (
	"context"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ListBooksPaginated returns books ordered by title with pagination and total count.
func (d *DB) ListBooksPaginated(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list books paginated", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book and total", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}

	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&total); err != nil {
			slog.ErrorContext(ctx, "Failed to count total books", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
	}

	return books, total, nil
}

// ListRecentBooks returns books ordered by creation time (newest first) with pagination and total count.
func (d *DB) ListRecentBooks(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing recent books",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := "ORDER BY created_at DESC, rowid DESC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at DESC, id DESC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list recent books", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book and total", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}

	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&total); err != nil {
			slog.ErrorContext(ctx, "Failed to count total books", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
	}

	return books, total, nil
}

// ListBooksByAuthor returns all books for a specific author.
func (d *DB) ListBooksByAuthor(ctx context.Context, authorID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by author", slog.String(otelkeys.AuthorID, authorID))
	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1 `+orderBy,
		authorID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list books by author", slog.Any(otelkeys.Error, err))
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book", slog.Any(otelkeys.Error, err))
			return nil, err
		}
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, err
	}
	return books, nil
}

// ListBooksByAuthorPaginated returns books for a specific author with pagination and total count.
func (d *DB) ListBooksByAuthorPaginated(ctx context.Context, authorID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by author paginated",
		slog.String(otelkeys.AuthorID, authorID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		authorID, limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list books by author paginated", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book and total", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}

	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1`,
			authorID,
		).Scan(&total); err != nil {
			slog.ErrorContext(ctx, "Failed to count total books by author", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
	}

	return books, total, nil
}

// ListBooksBySeries returns all books in a specific series, ordered by position.
func (d *DB) ListBooksBySeries(ctx context.Context, seriesID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by series", slog.String(otelkeys.SeriesID, seriesID))
	nullsLast := "ORDER BY bs.position ASC, b.title ASC"
	if d.Dialect == DialectPostgres {
		nullsLast = "ORDER BY bs.position ASC NULLS LAST, b.title ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1 `+nullsLast,
		seriesID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list books by series", slog.Any(otelkeys.Error, err))
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book", slog.Any(otelkeys.Error, err))
			return nil, err
		}
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, err
	}
	return books, nil
}

// ListBooksBySeriesPaginated returns books in a specific series with pagination and total count.
func (d *DB) ListBooksBySeriesPaginated(ctx context.Context, seriesID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by series paginated",
		slog.String(otelkeys.SeriesID, seriesID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	nullsLast := "ORDER BY bs.position ASC, b.title ASC"
	if d.Dialect == DialectPostgres {
		nullsLast = "ORDER BY bs.position ASC NULLS LAST, b.title ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1 `+nullsLast+` LIMIT $2 OFFSET $3`,
		seriesID, limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list books by series paginated", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book and total", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}

	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1`,
			seriesID,
		).Scan(&total); err != nil {
			slog.ErrorContext(ctx, "Failed to count total books by series", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
	}

	return books, total, nil
}

// SearchBooks searches books by title or description with pagination and total count.
func (d *DB) SearchBooks(ctx context.Context, query string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: searching books",
		slog.String(otelkeys.Query, query),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	likePattern := "%" + escaped + "%"

	var whereClause string
	if d.Dialect == DialectPostgres {
		whereClause = `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\')`
	} else {
		whereClause = `WHERE (title LIKE $1 ESCAPE '\' OR description LIKE $1 ESCAPE '\')`
	}

	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books `+whereClause+` `+orderBy+` LIMIT $2 OFFSET $3`,
		likePattern, limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to search books", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan book and total", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate book rows", slog.Any(otelkeys.Error, err))
		return nil, 0, err
	}

	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books `+whereClause, likePattern).Scan(&total); err != nil {
			slog.ErrorContext(ctx, "Failed to count total search results", slog.Any(otelkeys.Error, err))
			return nil, 0, err
		}
	}

	return books, total, nil
}
