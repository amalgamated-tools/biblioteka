package db

import (
	"context"
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

// normalizeName trims leading/trailing whitespace and collapses internal runs
// of whitespace to a single space. It preserves the caller's capitalization.
func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	return collapseSpaces.ReplaceAllString(name, " ")
}

// NormalizeAuthorName normalizes an author name by trimming whitespace and
// collapsing internal runs to a single space. Names like "McCaffrey" or
// "de la Cruz" are stored as-is.
func NormalizeAuthorName(name string) string { return normalizeName(name) }

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

type authorListQuery struct{}

func (authorListQuery) table() string   { return "authors" }
func (authorListQuery) columns() string { return authorColumns }
func (authorListQuery) orderBy(d *DB) string {
	return d.dialectOrderBy("name", "ASC")
}

func scanAuthor(row interface{ Scan(...any) error }) (*Author, error) {
	var a Author
	err := row.Scan(&a.ID, &a.Name, &a.GoodreadsID, &a.HardcoverID, &a.GoogleBooksID, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (d *DB) CreateAuthor(ctx context.Context, name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	return namedEntityCreate(ctx, "author", name, NormalizeAuthorName, ErrInvalidAuthorName, ErrAuthorNameExists,
		func(ctx context.Context, n string) (*Author, error) {
			return scanAuthor(d.QueryRowContext(ctx,
				`INSERT INTO authors (name, goodreads_id, hardcover_id, google_books_id, image_url) VALUES ($1, $2, $3, $4, $5) RETURNING `+authorColumns,
				n, goodreadsID, hardcoverID, googleBooksID, imageURL,
			))
		},
	)
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
	return listAll(ctx, d, authorListQuery{}, scanAuthor)
}

// ListAuthorsPaginated returns authors ordered by name with pagination and total count.
func (d *DB) ListAuthorsPaginated(ctx context.Context, limit, offset int) ([]Author, int, error) {
	slog.DebugContext(ctx, "db: listing authors paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	return listPaginated(ctx, d, authorListQuery{}, limit, offset, scanAuthor)
}

func (d *DB) UpdateAuthor(ctx context.Context, id, name string, goodreadsID, hardcoverID, googleBooksID, imageURL *string) (*Author, error) {
	return namedEntityUpdate(ctx, "author", id, name, NormalizeAuthorName, ErrInvalidAuthorName, ErrAuthorNameExists,
		func(ctx context.Context, entityID, n string) (*Author, error) {
			return scanAuthor(d.QueryRowContext(ctx,
				`UPDATE authors SET name = $1, goodreads_id = $2, hardcover_id = $3, google_books_id = $4, image_url = $5, updated_at = `+d.now()+` WHERE id = $6 RETURNING `+authorColumns,
				n, goodreadsID, hardcoverID, googleBooksID, imageURL, entityID,
			))
		},
	)
}

// FindOrCreateAuthor looks up an author by name (case-insensitive) and returns
// it, creating a new one if it doesn't exist. Handles concurrent insert races
// gracefully.
func (d *DB) FindOrCreateAuthor(ctx context.Context, name string) (*Author, error) {
	return findOrCreate(ctx, name, "author",
		NormalizeAuthorName, ErrInvalidAuthorName, ErrAuthorNameExists,
		d.GetAuthorByName,
		func(ctx context.Context, n string) (*Author, error) {
			return d.CreateAuthor(ctx, n, nil, nil, nil, nil)
		},
	)
}

func (d *DB) DeleteAuthor(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting author", slog.String(otelkeys.ID, id))
	return d.execAffected(ctx, `DELETE FROM authors WHERE id = $1`, id)
}
