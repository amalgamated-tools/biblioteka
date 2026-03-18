package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strconv"

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
	NumPages        *int      `json:"num_pages"`
	CoverImageURL   *string   `json:"cover_image_url"`
	CreatedAt       Timestamp `json:"created_at"`
	UpdatedAt       Timestamp `json:"updated_at"`
}

const bookColumns = `id, title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, num_pages, cover_image_url, created_at, updated_at`

func scanBook(row interface{ Scan(...any) error }) (*Book, error) {
	var b Book
	err := row.Scan(&b.ID, &b.Title, &b.Description, &b.ASIN, &b.ISBN10, &b.ISBN13, &b.GoodreadsID, &b.HardcoverID, &b.GoogleBooksID, &b.PublicationDate, &b.Publisher, &b.Language, &b.NumPages, &b.CoverImageURL, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// scanBookAndTotal scans book columns plus a trailing COUNT(*) OVER() total.
func scanBookAndTotal(row interface{ Scan(...any) error }) (*Book, int, error) {
	var b Book
	var total int
	err := row.Scan(&b.ID, &b.Title, &b.Description, &b.ASIN, &b.ISBN10, &b.ISBN13, &b.GoodreadsID, &b.HardcoverID, &b.GoogleBooksID, &b.PublicationDate, &b.Publisher, &b.Language, &b.NumPages, &b.CoverImageURL, &b.CreatedAt, &b.UpdatedAt, &total)
	if err != nil {
		return nil, 0, err
	}
	return &b, total, nil
}

// CreateBook inserts a new book and returns it.
func (d *DB) CreateBook(ctx context.Context, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*Book, error) {
	slog.DebugContext(ctx, "db: creating book", slog.String(otelkeys.Title, title))
	b, err := scanBook(d.QueryRowContext(ctx,
		`INSERT INTO books (title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, num_pages, cover_image_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING `+bookColumns,
		title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, numPages, coverImageURL,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CreateBookWithFile atomically creates a book and its associated file record
// within a single transaction. If either insert fails the transaction is rolled back.
func (d *DB) CreateBookWithFile(ctx context.Context, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string, fileType, fileName string, fileSize int64, fileHash *string, filePath string) (*Book, *BookFile, error) {
	slog.DebugContext(ctx, "db: creating book with file",
		slog.String(otelkeys.Title, title),
		slog.String(otelkeys.FileName, fileName),
	)

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	b, err := scanBook(tx.QueryRowContext(ctx,
		`INSERT INTO books (title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, num_pages, cover_image_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING `+bookColumns,
		title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, numPages, coverImageURL,
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
	slog.DebugContext(ctx, "db: fetching book", slog.String(otelkeys.ID, id))
	return scanBook(d.QueryRowContext(ctx,
		`SELECT `+bookColumns+` FROM books WHERE id = $1`,
		id,
	))
}

// ListBooks returns all books ordered by title.
func (d *DB) ListBooks(ctx context.Context) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books")
	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books `+orderBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, *b)
	}
	return books, rows.Err()
}

// ListBooksByLibrary returns all books in a specific library.
func (d *DB) ListBooksByLibrary(ctx context.Context, libraryID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by library", slog.String(otelkeys.LibraryID, libraryID))
	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT b.id, b.title, b.description, b.asin, b.isbn10, b.isbn13, b.goodreads_id, b.hardcover_id, b.google_books_id, b.publication_date, b.publisher, b.language, b.num_pages, b.cover_image_url, b.created_at, b.updated_at FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1 `+orderBy,
		libraryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, *b)
	}
	return books, rows.Err()
}

// ListBooksByLibraryPaginated returns books in a specific library with pagination and total count.
func (d *DB) ListBooksByLibraryPaginated(ctx context.Context, libraryID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by library paginated",
		slog.String(otelkeys.LibraryID, libraryID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		libraryID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	var total int
	for rows.Next() {
		b, t, err := scanBookAndTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = t
		books = append(books, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// When offset exceeds total rows, the window function returns nothing.
	// Fall back to a COUNT query so the caller can report the true total.
	if len(books) == 0 && offset > 0 {
		if err := d.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1`,
			libraryID,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	return books, total, nil
}

// UpdateBook updates a book's fields and returns the updated book.
func (d *DB) UpdateBook(ctx context.Context, id, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*Book, error) {
	slog.DebugContext(ctx, "db: updating book",
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.Title, title),
	)
	b, err := scanBook(d.QueryRowContext(ctx,
		`UPDATE books SET title = $1, description = $2, asin = $3, isbn10 = $4, isbn13 = $5, goodreads_id = $6, hardcover_id = $7, google_books_id = $8, publication_date = $9, publisher = $10, language = $11, num_pages = $12, cover_image_url = $13, updated_at = `+d.now()+` WHERE id = $14 RETURNING `+bookColumns,
		title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, numPages, coverImageURL, id,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteBook removes a book by ID.
func (d *DB) DeleteBook(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting book", slog.String(otelkeys.ID, id))
	res, err := d.ExecContext(ctx, `DELETE FROM books WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	res, err := d.ExecContext(ctx,
		`DELETE FROM library_books WHERE library_id = $1 AND book_id = $2`,
		libraryID, bookID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// bookColumnsWithPrefix returns book columns with a table alias prefix.
func bookColumnsWithPrefix(prefix string) string {
	return prefix + "id, " + prefix + "title, " + prefix + "description, " + prefix + "asin, " + prefix + "isbn10, " + prefix + "isbn13, " + prefix + "goodreads_id, " + prefix + "hardcover_id, " + prefix + "google_books_id, " + prefix + "publication_date, " + prefix + "publisher, " + prefix + "language, " + prefix + "num_pages, " + prefix + "cover_image_url, " + prefix + "created_at, " + prefix + "updated_at"
}

// dollarN returns a PostgreSQL-style positional placeholder ($1, $2, ...).
// SQLite also accepts dollar-sign placeholders.
func dollarN(n int) string {
	return "$" + strconv.Itoa(n)
}
