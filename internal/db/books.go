package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strconv"
	"strings"

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

	var total int
	if err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM library_books WHERE library_id = $1`,
		libraryID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN library_books lb ON lb.book_id = b.id WHERE lb.library_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		libraryID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
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

// BookSeriesEntry represents a book's membership in a series with its position.
type BookSeriesEntry struct {
	Series   Series   `json:"series"`
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
		a, err := scanAuthor(rows)
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
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// BookSeriesInput represents input for setting a book's series membership.
type BookSeriesInput struct {
	SeriesID string   `json:"series_id"`
	Position *float64 `json:"position"`
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

// ListBooksPaginated returns books ordered by title with pagination and total count.
func (d *DB) ListBooksPaginated(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
}

// ListRecentBooks returns books ordered by creation time (newest first) with pagination and total count.
func (d *DB) ListRecentBooks(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing recent books",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY created_at DESC, rowid DESC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at DESC, id DESC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
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

// ListBooksByAuthorPaginated returns books for a specific author with pagination and total count.
func (d *DB) ListBooksByAuthorPaginated(ctx context.Context, authorID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by author paginated",
		slog.String(otelkeys.AuthorID, authorID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1`,
		authorID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY b.title ASC, b.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY b.title ASC, b.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		authorID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
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

// ListBooksBySeriesPaginated returns books in a specific series with pagination and total count.
func (d *DB) ListBooksBySeriesPaginated(ctx context.Context, seriesID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by series paginated",
		slog.String(otelkeys.SeriesID, seriesID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1`,
		seriesID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	nullsLast := "ORDER BY bs.position ASC, b.title ASC"
	if d.Dialect == DialectPostgres {
		nullsLast = "ORDER BY bs.position ASC NULLS LAST, b.title ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1 `+nullsLast+` LIMIT $2 OFFSET $3`,
		seriesID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
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

	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM books `+whereClause, likePattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM books `+whereClause+` `+orderBy+` LIMIT $2 OFFSET $3`,
		likePattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, 0, err
		}
		books = append(books, *b)
	}
	return books, total, rows.Err()
}

// bookColumnsWithPrefix returns book columns with a table alias prefix.
func bookColumnsWithPrefix(prefix string) string {
	return prefix + "id, " + prefix + "title, " + prefix + "description, " + prefix + "asin, " + prefix + "isbn10, " + prefix + "isbn13, " + prefix + "goodreads_id, " + prefix + "hardcover_id, " + prefix + "google_books_id, " + prefix + "publication_date, " + prefix + "publisher, " + prefix + "language, " + prefix + "num_pages, " + prefix + "cover_image_url, " + prefix + "created_at, " + prefix + "updated_at"
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
		var a Author
		if err := rows.Scan(&bookID, &a.ID, &a.Name, &a.GoodreadsID, &a.HardcoverID, &a.GoogleBooksID, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		result[bookID] = append(result[bookID], a)
	}
	return result, rows.Err()
}

// GetFilesForBooks returns book files grouped by book ID for the given book IDs.
func (d *DB) GetFilesForBooks(ctx context.Context, bookIDs []string) (map[string][]BookFile, error) {
	if len(bookIDs) == 0 {
		return nil, nil
	}
	slog.DebugContext(ctx, "db: batch fetching files for books", slog.Int(otelkeys.BookCount, len(bookIDs)))

	placeholders := make([]string, len(bookIDs))
	args := make([]any, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = dollarN(i + 1)
		args[i] = id
	}

	orderBy := "ORDER BY bf.file_name ASC, bf.rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY bf.file_name ASC, bf.id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookFileColumnsWithPrefix("bf.")+` FROM book_files bf WHERE bf.book_id IN (`+strings.Join(placeholders, ",")+`) `+orderBy,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]BookFile, len(bookIDs))
	for rows.Next() {
		var bf BookFile
		if err := rows.Scan(&bf.ID, &bf.BookID, &bf.FileType, &bf.FileName, &bf.FileSize, &bf.FileHash, &bf.FilePath, &bf.CreatedAt, &bf.UpdatedAt); err != nil {
			return nil, err
		}
		result[bf.BookID] = append(result[bf.BookID], bf)
	}
	return result, rows.Err()
}

// dollarN returns a PostgreSQL-style positional placeholder ($1, $2, ...).
// SQLite also accepts dollar-sign placeholders.
func dollarN(n int) string {
	return "$" + strconv.Itoa(n)
}

// bookFileColumnsWithPrefix returns book_files columns with a table alias prefix.
func bookFileColumnsWithPrefix(prefix string) string {
	return prefix + "id, " + prefix + "book_id, " + prefix + "file_type, " + prefix + "file_name, " + prefix + "file_size, " + prefix + "file_hash, " + prefix + "file_path, " + prefix + "created_at, " + prefix + "updated_at"
}
