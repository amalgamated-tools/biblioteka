package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Book represents a row in the books table.
type Book struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	ASIN            *string   `json:"asin"`
	ISBN10          *string   `json:"isbn10"`
	ISBN13          *string   `json:"isbn13"`
	GoodreadsID     *string   `json:"goodreads_id"`
	HardcoverID     *string   `json:"hardcover_id"`
	GoogleBooksID   *string   `json:"google_books_id"`
	PublicationDate *string   `json:"publication_date"`
	Publisher       *string   `json:"publisher"`
	Language        *string   `json:"language"`
	CoverImageURL   *string   `json:"cover_image_url"`
	CreatedAt       Timestamp `json:"created_at"`
	UpdatedAt       Timestamp `json:"updated_at"`
}

const bookColumns = `id, title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, cover_image_url, created_at, updated_at`

// scanBook scans a book row into a Book struct.
func scanBook(row interface{ Scan(...any) error }) (*Book, error) {
	return scanRow(row, func(b *Book) []any {
		return []any{&b.ID, &b.Title, &b.Description, &b.ASIN, &b.ISBN10, &b.ISBN13, &b.GoodreadsID, &b.HardcoverID, &b.GoogleBooksID, &b.PublicationDate, &b.Publisher, &b.Language, &b.CoverImageURL, &b.CreatedAt, &b.UpdatedAt}
	})
}

// scanBookAndTotal scans book columns plus a trailing COUNT(*) OVER() total.
func scanBookAndTotal(row interface{ Scan(...any) error }) (*Book, int, error) {
	var b Book
	var total int
	err := row.Scan(&b.ID, &b.Title, &b.Description, &b.ASIN, &b.ISBN10, &b.ISBN13, &b.GoodreadsID, &b.HardcoverID, &b.GoogleBooksID, &b.PublicationDate, &b.Publisher, &b.Language, &b.CoverImageURL, &b.CreatedAt, &b.UpdatedAt, &total)
	if err != nil {
		return nil, 0, err
	}
	return &b, total, nil
}

// BookInput holds the fields used to create or update a book record.
type BookInput struct {
	Title           string
	Description     *string
	ASIN            *string
	ISBN10          *string
	ISBN13          *string
	GoodreadsID     *string
	HardcoverID     *string
	GoogleBooksID   *string
	PublicationDate *string
	Publisher       *string
	Language        *string
	CoverImageURL   *string
}

// CreateBook inserts a new book and returns it.
func (d *DB) CreateBook(ctx context.Context, input BookInput) (*Book, error) {
	slog.DebugContext(ctx, "db: creating book", slog.String(otelkeys.Title, input.Title))
	b, err := scanBook(d.QueryRowContext(ctx,
		`INSERT INTO books (title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, cover_image_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING `+bookColumns,
		input.Title, input.Description, input.ASIN, input.ISBN10, input.ISBN13, input.GoodreadsID, input.HardcoverID, input.GoogleBooksID, input.PublicationDate, input.Publisher, input.Language, input.CoverImageURL,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CreateBookWithFile atomically creates a book and its associated file record
// within a single transaction. If either insert fails the transaction is rolled back.
func (d *DB) CreateBookWithFile(ctx context.Context, input BookInput, fileType, fileName string, fileSize int64, fileHash *string, filePath string) (*Book, *BookFile, error) {
	slog.DebugContext(ctx, "db: creating book with file",
		slog.String(otelkeys.Title, input.Title),
		slog.String(otelkeys.FileName, fileName),
	)

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer deferRollback(ctx, tx)

	b, err := scanBook(tx.QueryRowContext(ctx,
		`INSERT INTO books (title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, cover_image_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING `+bookColumns,
		input.Title, input.Description, input.ASIN, input.ISBN10, input.ISBN13, input.GoodreadsID, input.HardcoverID, input.GoogleBooksID, input.PublicationDate, input.Publisher, input.Language, input.CoverImageURL,
	))
	if err != nil {
		return nil, nil, err
	}

	bf, err := scanBookFile(tx.QueryRowContext(ctx,
		`INSERT INTO book_files (book_id, file_type, file_name, file_size, file_hash, file_path) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+bookFileColumns,
		b.ID, fileType, fileName, fileSize, fileHash, filePath,
	))
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return b, bf, nil
}

// GetBook returns a book by ID, or sql.ErrNoRows if not found.
func (d *DB) GetBook(ctx context.Context, id string) (*Book, error) {
	slog.DebugContext(ctx, "db: fetching book", slog.String(otelkeys.BookID, id))
	return scanBook(d.QueryRowContext(ctx,
		`SELECT `+bookColumns+` FROM books WHERE id = $1`,
		id,
	))
}

// FindBookByExternalID returns the first book that matches any of the given
// external identifiers, checked in priority order: ISBN-13, ISBN-10, ASIN,
// Goodreads ID. Returns sql.ErrNoRows when no match is found.
func (d *DB) FindBookByExternalID(ctx context.Context, isbn13, isbn10, asin, goodreadsID *string) (*Book, error) {
	valueOrEmpty := func(v *string) string {
		if v == nil {
			return ""
		}
		return *v
	}

	isbn13Value := valueOrEmpty(isbn13)
	isbn10Value := valueOrEmpty(isbn10)
	asinValue := valueOrEmpty(asin)
	goodreadsIDValue := valueOrEmpty(goodreadsID)

	// Skip the query entirely when no identifiers are provided.
	if isbn13Value == "" && isbn10Value == "" && asinValue == "" && goodreadsIDValue == "" {
		return nil, sql.ErrNoRows
	}

	book, err := scanBook(d.QueryRowContext(ctx,
		`SELECT `+bookColumns+` FROM books
		WHERE ($1 <> '' AND isbn13 = $1)
			OR ($2 <> '' AND isbn10 = $2)
			OR ($3 <> '' AND asin = $3)
			OR ($4 <> '' AND goodreads_id = $4)
		ORDER BY CASE
			WHEN $1 <> '' AND isbn13 = $1 THEN 1
			WHEN $2 <> '' AND isbn10 = $2 THEN 2
			WHEN $3 <> '' AND asin = $3 THEN 3
			WHEN $4 <> '' AND goodreads_id = $4 THEN 4
			ELSE 5
		END
		LIMIT 1`,
		isbn13Value,
		isbn10Value,
		asinValue,
		goodreadsIDValue,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("find book by external id: %w", err)
	}
	return book, nil
}

// ListBooks returns all books ordered by title.
func (d *DB) ListBooks(ctx context.Context) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books")
	orderBy := d.dialectOrderBy("title", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books `+orderBy,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBook)
}

// ListBooksByLibrary returns all books in a specific library.
func (d *DB) ListBooksByLibrary(ctx context.Context, libraryID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by library", slog.String(otelkeys.LibraryID, libraryID))
	orderBy := d.dialectOrderBy("b.title", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1 `+orderBy,
		libraryID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBook)
}

// ListBooksByLibraryPaginated returns books in a specific library with pagination and total count.
func (d *DB) ListBooksByLibraryPaginated(ctx context.Context, libraryID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by library paginated",
		slog.String(otelkeys.LibraryID, libraryID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := d.dialectOrderBy("b.title", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		libraryID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	books, total, err := collectRowsAndTotal(rows, scanBookAndTotal)
	if err != nil {
		return nil, 0, err
	}
	// When offset exceeds total rows, the window function returns nothing.
	// Fall back to a COUNT query so the caller can report the true total.
	if err := countFallback(ctx, d, &total, len(books), offset,
		`SELECT COUNT(*) FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1`,
		libraryID,
	); err != nil {
		return nil, 0, err
	}
	return books, total, nil
}

// UpdateBook updates a book's fields and returns the updated book.
func (d *DB) UpdateBook(ctx context.Context, id string, input BookInput) (*Book, error) {
	slog.DebugContext(ctx, "db: updating book",
		slog.String(otelkeys.BookID, id),
		slog.String(otelkeys.Title, input.Title),
	)
	b, err := scanBook(d.QueryRowContext(ctx,
		`UPDATE books SET title = $1, description = $2, asin = $3, isbn10 = $4, isbn13 = $5, goodreads_id = $6, hardcover_id = $7, google_books_id = $8, publication_date = $9, publisher = $10, language = $11, cover_image_url = $12, updated_at = `+d.now()+` WHERE id = $13 RETURNING `+bookColumns,
		input.Title, input.Description, input.ASIN, input.ISBN10, input.ISBN13, input.GoodreadsID, input.HardcoverID, input.GoogleBooksID, input.PublicationDate, input.Publisher, input.Language, input.CoverImageURL, id,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteBook removes a book by ID.
func (d *DB) DeleteBook(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting book", slog.String(otelkeys.BookID, id))
	return d.execAffected(ctx, `DELETE FROM books WHERE id = $1`, id)
}

// AddBookToLibrary creates an association between a book and a library.
func (d *DB) AddBookToLibrary(ctx context.Context, libraryID, bookID string) error {
	slog.DebugContext(ctx, "db: adding book to library",
		slog.String(otelkeys.LibraryID, libraryID),
		slog.String(otelkeys.BookID, bookID),
	)
	_, err := d.ExecContext(ctx,
		`INSERT INTO library_books (library_id, book_id) VALUES ($1, $2)`,
		libraryID, bookID,
	)
	return err
}

// RemoveBookFromLibrary removes the association between a book and a library.
func (d *DB) RemoveBookFromLibrary(ctx context.Context, libraryID, bookID string) error {
	slog.DebugContext(ctx, "db: removing book from library",
		slog.String(otelkeys.LibraryID, libraryID),
		slog.String(otelkeys.BookID, bookID),
	)
	return d.execAffected(ctx,
		`DELETE FROM library_books WHERE library_id = $1 AND book_id = $2`,
		libraryID, bookID,
	)
}

// bookColumnsWithPrefix returns book columns with a table alias prefix.
func bookColumnsWithPrefix(prefix string) string {
	cols := strings.Split(bookColumns, ",")
	for i, c := range cols {
		cols[i] = prefix + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

// ListBooksModifiedSince returns up to limit books updated after since, ordered by
// updated_at ascending so callers can track the high-water mark. When since is the
// zero time all books are returned (initial sync). For non-zero since values, callers
// should also pass the last seen book ID as lastID to avoid skipping rows that share
// the same updated_at across pages.
func (d *DB) ListBooksModifiedSince(ctx context.Context, since time.Time, lastID string, limit int) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books modified since",
		slog.Int(otelkeys.Limit, limit),
	)
	// Use a stable, deterministic ordering that matches the tie-breaker used in the
	// WHERE clause for pagination.
	const orderBy = "ORDER BY updated_at ASC, id ASC"
	if since.IsZero() {
		rows, err := d.QueryContext(ctx,
			`SELECT `+bookColumns+` FROM books `+orderBy+` LIMIT $1`,
			limit,
		)
		if err != nil {
			return nil, err
		}
		return collectRows(rows, scanBook)
	}

	var sinceParam any
	if d.Dialect == DialectPostgres {
		sinceParam = since
	} else {
		// SQLite stores datetimes as "YYYY-MM-DD HH:MM:SS"; use matching format
		// to ensure correct string-based datetime comparison.
		sinceParam = since.UTC().Format("2006-01-02 15:04:05")
	}

	// For incremental sync, include a tie-breaker on ID so that rows sharing the same
	// updated_at value but appearing on different pages are not skipped.
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books WHERE (updated_at > $1 OR (updated_at = $1 AND id > $2)) `+orderBy+` LIMIT $3`,
		sinceParam, lastID, limit,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBook)
}

// GetSeriesForBooks returns series entries (with position) grouped by book ID for the given book IDs.
func (d *DB) GetSeriesForBooks(ctx context.Context, bookIDs []string) (map[string][]BookSeriesEntry, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching series for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	inClause, args := buildInClause(bookIDs, 1)

	rows, err := d.QueryContext(ctx,
		`SELECT bs.book_id, s.id, s.name, s.goodreads_id, s.hardcover_id, s.google_books_id, s.created_at, s.updated_at, bs.position
		FROM series s INNER JOIN book_series bs ON bs.series_id = s.id
		WHERE bs.book_id IN (`+inClause+`)
		ORDER BY s.name ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}

	return collectRowsGrouped(rows, func(row interface{ Scan(...any) error }) (string, *BookSeriesEntry, error) {
		var bookID string
		entry, err := scanBookSeriesEntry(prefixedScanner{row: row, prefix: []any{&bookID}})
		if err != nil {
			return "", nil, err
		}
		return bookID, entry, nil
	}, len(bookIDs))
}
