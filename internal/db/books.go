package db

import (
	"context"
	"database/sql"
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
func (d *DB) CreateBook(title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*Book, error) {
	b, err := scanBook(d.QueryRow(
		`INSERT INTO books (title, description, asin, isbn10, isbn13, goodreads_id, hardcover_id, google_books_id, publication_date, publisher, language, num_pages, cover_image_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING `+bookColumns,
		title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, numPages, coverImageURL,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GetBook returns a book by ID, or sql.ErrNoRows if not found.
func (d *DB) GetBook(id string) (*Book, error) {
	return scanBook(d.QueryRow(
		`SELECT `+bookColumns+` FROM books WHERE id = $1`,
		id,
	))
}

// ListBooks returns all books ordered by title.
func (d *DB) ListBooks() ([]Book, error) {
	orderBy := "ORDER BY title ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY title ASC, id ASC"
	}
	rows, err := d.Query(
		`SELECT ` + bookColumns + ` FROM books ` + orderBy,
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
func (d *DB) ListBooksByLibrary(libraryID string) ([]Book, error) {
	orderBy := "ORDER BY b.title ASC"
	rows, err := d.Query(
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

// UpdateBook updates a book's fields and returns the updated book.
func (d *DB) UpdateBook(id, title string, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language *string, numPages *int, coverImageURL *string) (*Book, error) {
	b, err := scanBook(d.QueryRow(
		`UPDATE books SET title = $1, description = $2, asin = $3, isbn10 = $4, isbn13 = $5, goodreads_id = $6, hardcover_id = $7, google_books_id = $8, publication_date = $9, publisher = $10, language = $11, num_pages = $12, cover_image_url = $13, updated_at = `+d.now()+` WHERE id = $14 RETURNING `+bookColumns,
		title, description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, numPages, coverImageURL, id,
	))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteBook removes a book by ID.
func (d *DB) DeleteBook(id string) error {
	res, err := d.Exec(`DELETE FROM books WHERE id = $1`, id)
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
func (d *DB) AddBookToLibrary(libraryID, bookID string) error {
	_, err := d.Exec(
		`INSERT INTO library_books (library_id, book_id) VALUES ($1, $2)`,
		libraryID, bookID,
	)
	return err
}

// RemoveBookFromLibrary removes the association between a book and a library.
func (d *DB) RemoveBookFromLibrary(libraryID, bookID string) error {
	res, err := d.Exec(
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
func (d *DB) GetBookAuthors(bookID string) ([]Author, error) {
	rows, err := d.Query(
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
func (d *DB) SetBookAuthors(bookID string, authorIDs []string) error {
	seen := make(map[string]struct{}, len(authorIDs))
	unique := make([]string, 0, len(authorIDs))
	for _, id := range authorIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}

	ctx := context.Background()
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
func (d *DB) GetBookSeries(bookID string) ([]BookSeriesEntry, error) {
	rows, err := d.Query(
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
func (d *DB) SetBookSeries(bookID string, entries []BookSeriesInput) error {
	seen := make(map[string]struct{}, len(entries))
	unique := make([]BookSeriesInput, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		if _, ok := seen[entries[i].SeriesID]; !ok {
			seen[entries[i].SeriesID] = struct{}{}
			unique = append(unique, entries[i])
		}
	}

	ctx := context.Background()
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
