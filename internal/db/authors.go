package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var (
	ErrAuthorNameExists  = errors.New("author name already exists")
	ErrInvalidAuthorName = errors.New("author name cannot be blank")
)

var collapseSpaces = regexp.MustCompile(`\s+`)

// NormalizeAuthorName trims leading/trailing whitespace and collapses internal
// runs of whitespace to a single space. It preserves the caller's
// capitalization so names like "McCaffrey" or "de la Cruz" are stored as-is.
func NormalizeAuthorName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	return collapseSpaces.ReplaceAllString(name, " ")
}

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

func (d *DB) CreateAuthor(ctx context.Context, name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	name = NormalizeAuthorName(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting author with blank name after normalization")
		return nil, ErrInvalidAuthorName
	}
	slog.DebugContext(ctx, "db: creating author", slog.String(otelkeys.Name, name))

	a, err := scanAuthor(d.QueryRowContext(ctx,
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

func (d *DB) GetAuthor(ctx context.Context, id string) (*Author, error) {
	slog.DebugContext(ctx, "db: fetching author", slog.String(otelkeys.ID, id))
	return scanAuthor(d.QueryRowContext(ctx,
		`SELECT `+authorColumns+` FROM authors WHERE id = $1`,
		id,
	))
}

// GetAuthorByName looks up an author by name using case-insensitive matching.
// The stored name preserves the original capitalization provided by the caller.
func (d *DB) GetAuthorByName(ctx context.Context, name string) (*Author, error) {
	name = NormalizeAuthorName(name)
	slog.DebugContext(ctx, "db: fetching author by name", slog.String(otelkeys.Name, name))
	return scanAuthor(d.QueryRowContext(ctx,
		`SELECT `+authorColumns+` FROM authors WHERE LOWER(name) = LOWER($1)`,
		name,
	))
}

func (d *DB) ListAuthors(ctx context.Context) ([]Author, error) {
	slog.DebugContext(ctx, "db: listing authors")
	orderBy := "ORDER BY name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY name ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+authorColumns+` FROM authors `+orderBy,
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

// ListAuthorsPaginated returns authors ordered by name with pagination and total count.
func (d *DB) ListAuthorsPaginated(ctx context.Context, limit, offset int) ([]Author, int, error) {
	slog.DebugContext(ctx, "db: listing authors paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	var total int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM authors`).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "ORDER BY name ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY name ASC, id ASC"
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+authorColumns+` FROM authors `+orderBy+` LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		a, err := scanAuthor(rows)
		if err != nil {
			return nil, 0, err
		}
		authors = append(authors, *a)
	}
	return authors, total, rows.Err()
}

func (d *DB) UpdateAuthor(ctx context.Context, id, name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	name = NormalizeAuthorName(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting author update with blank name after normalization")
		return nil, ErrInvalidAuthorName
	}
	slog.DebugContext(ctx, "db: updating author",
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.Name, name),
	)
	a, err := scanAuthor(d.QueryRowContext(ctx,
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

// FindOrCreateAuthor looks up an author by name (case-insensitive) and returns
// it, creating a new one if it doesn't exist. Handles concurrent insert races
// gracefully.
func (d *DB) FindOrCreateAuthor(ctx context.Context, name string) (*Author, error) {
	name = NormalizeAuthorName(name)
	if name == "" {
		return nil, ErrInvalidAuthorName
	}
	slog.DebugContext(ctx, "db: find or create author", slog.String(otelkeys.Name, name))

	// Look up using the same case-insensitive predicate as GetAuthorByName.
	a, err := d.GetAuthorByName(ctx, name)
	if err == nil {
		return a, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	// Not found — insert.
	a, err = d.CreateAuthor(ctx, name, nil, nil, nil, nil)
	if err == nil {
		return a, nil
	}
	if err != ErrAuthorNameExists {
		return nil, err
	}
	// Concurrent insert won the race — fetch.
	return d.GetAuthorByName(ctx, name)
}

func (d *DB) DeleteAuthor(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting author", slog.String(otelkeys.ID, id))
	res, err := d.ExecContext(ctx, `DELETE FROM authors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
