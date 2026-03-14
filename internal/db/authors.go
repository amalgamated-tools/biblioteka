package db

import (
	"database/sql"
	"errors"
	"log/slog"
)

var ErrAuthorNameExists = errors.New("author name already exists")

type Author struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	GoodreadsID   *string   `json:"goodreads_id"`
	HardcoverID   *string   `json:"hardcover_id"`
	GoogleBooksID *string   `json:"google_books_id"`
	ImageURL      *string   `json:"image_url"`
	CreatedAt     Timestamp `json:"created_at"`
	UpdatedAt     Timestamp `json:"updated_at"`
}

const authorColumns = `id, name, goodreads_id, hardcover_id, google_books_id, image_url, created_at, updated_at`

func scanAuthor(row interface{ Scan(...any) error }) (*Author, error) {
	var a Author
	err := row.Scan(&a.ID, &a.Name, &a.GoodreadsID, &a.HardcoverID, &a.GoogleBooksID, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *DB) CreateAuthor(name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	slog.Debug("db: creating author", slog.String("name", name))
	a, err := scanAuthor(d.QueryRow(
		`INSERT INTO authors (name, goodreads_id, hardcover_id, google_books_id, image_url) VALUES ($1, $2, $3, $4, $5) RETURNING `+authorColumns,
		name, goodreadsID, hardcoverID, googleBooksID, imageURL,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAuthorNameExists
		}
		return nil, err
	}
	return a, nil
}

func (d *DB) GetAuthor(id string) (*Author, error) {
	slog.Debug("db: fetching author", slog.String("id", id))
	return scanAuthor(d.QueryRow(
		`SELECT `+authorColumns+` FROM authors WHERE id = $1`,
		id,
	))
}

func (d *DB) ListAuthors() ([]Author, error) {
	slog.Debug("db: listing authors")
	orderBy := "ORDER BY name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY name ASC, id ASC"
	}
	rows, err := d.Query(
		`SELECT ` + authorColumns + ` FROM authors ` + orderBy,
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

func (d *DB) UpdateAuthor(id, name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	slog.Debug("db: updating author", slog.String("id", id), slog.String("name", name))
	a, err := scanAuthor(d.QueryRow(
		`UPDATE authors SET name = $1, goodreads_id = $2, hardcover_id = $3, google_books_id = $4, image_url = $5, updated_at = `+d.now()+` WHERE id = $6 RETURNING `+authorColumns,
		name, goodreadsID, hardcoverID, googleBooksID, imageURL, id,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAuthorNameExists
		}
		return nil, err
	}
	return a, nil
}

func (d *DB) DeleteAuthor(id string) error {
	slog.Debug("db: deleting author", slog.String("id", id))
	res, err := d.Exec(`DELETE FROM authors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
