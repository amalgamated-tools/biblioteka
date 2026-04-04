package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var (
	ErrSeriesNameExists  = errors.New("series name already exists")
	ErrInvalidSeriesName = errors.New("invalid series name")
)

// NormalizeSeriesName normalizes a series name by trimming whitespace and
// collapsing internal runs to a single space while preserving capitalization.
func NormalizeSeriesName(name string) string { return normalizeName(name) }

// Series represents a row in the series table.
type Series struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	GoodreadsID   *string   `json:"goodreads_id"`
	HardcoverID   *string   `json:"hardcover_id"`
	GoogleBooksID *string   `json:"google_books_id"`
	CreatedAt     Timestamp `json:"created_at"`
	UpdatedAt     Timestamp `json:"updated_at"`
}

const seriesColumns = `id, name, goodreads_id, hardcover_id, google_books_id, created_at, updated_at`

type seriesListQuery struct{}

func (seriesListQuery) table() string   { return "series" }
func (seriesListQuery) columns() string { return seriesColumns }
func (seriesListQuery) orderBy(d *DB) string {
	return d.dialectOrderBy("name", "ASC")
}

func scanSeries(row interface{ Scan(...any) error }) (*Series, error) {
	var s Series
	err := row.Scan(&s.ID, &s.Name, &s.GoodreadsID, &s.HardcoverID, &s.GoogleBooksID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CreateSeries inserts a new series with the given name and optional external
// IDs. The name is normalized before storage. Returns ErrSeriesNameExists if
// a series with an equivalent normalized name already exists.
func (d *DB) CreateSeries(ctx context.Context, name string, goodreadsID, hardcoverID, googleBooksID *string) (*Series, error) {
	return namedEntityCreate(ctx, "series", name, NormalizeSeriesName, ErrInvalidSeriesName, ErrSeriesNameExists,
		func(ctx context.Context, n string) (*Series, error) {
			return scanSeries(d.QueryRowContext(ctx,
				`INSERT INTO series (name, goodreads_id, hardcover_id, google_books_id) VALUES ($1, $2, $3, $4) RETURNING `+seriesColumns,
				n, goodreadsID, hardcoverID, googleBooksID,
			))
		},
	)
}

// GetSeries retrieves a series by its UUID. Returns sql.ErrNoRows if not found.
func (d *DB) GetSeries(ctx context.Context, id string) (*Series, error) {
	slog.DebugContext(ctx, "db: fetching series", slog.String(otelkeys.ID, id))
	return scanSeries(d.QueryRowContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE id = $1`,
		id,
	))
}

// GetSeriesByName looks up a series by name using case-insensitive matching
// after normalizing whitespace. The returned Series.Name preserves the
// capitalization stored in the database from when the row was created or
// last updated, which may differ from the capitalization used for lookup.
// Callers do not need to pre-normalize the input; this method handles it.
func (d *DB) GetSeriesByName(ctx context.Context, name string) (*Series, error) {
	name = NormalizeSeriesName(name)
	slog.DebugContext(ctx, "db: fetching series by name", slog.String(otelkeys.Name, name))
	return scanSeries(d.QueryRowContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE LOWER(name) = LOWER($1)`,
		name,
	))
}

// ListSeries returns all series ordered by name.
func (d *DB) ListSeries(ctx context.Context) ([]Series, error) {
	slog.DebugContext(ctx, "db: listing series")
	return listAll(ctx, d, seriesListQuery{}, scanSeries)
}

// ListSeriesPaginated returns series ordered by name with pagination and total count.
func (d *DB) ListSeriesPaginated(ctx context.Context, limit, offset int) ([]Series, int, error) {
	slog.DebugContext(ctx, "db: listing series paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)
	return listPaginated(ctx, d, seriesListQuery{}, limit, offset, scanSeries)
}

// UpdateSeries replaces the name and external IDs of the series identified by
// id. The name is normalized. Returns sql.ErrNoRows if the series does not
// exist, or ErrSeriesNameExists if the new name conflicts with another series.
func (d *DB) UpdateSeries(ctx context.Context, id, name string, goodreadsID, hardcoverID, googleBooksID *string) (*Series, error) {
	return namedEntityUpdate(ctx, "series", id, name, NormalizeSeriesName, ErrInvalidSeriesName, ErrSeriesNameExists,
		func(ctx context.Context, entityID, n string) (*Series, error) {
			return scanSeries(d.QueryRowContext(ctx,
				`UPDATE series SET name = $1, goodreads_id = $2, hardcover_id = $3, google_books_id = $4, updated_at = `+d.now()+` WHERE id = $5 RETURNING `+seriesColumns,
				n, goodreadsID, hardcoverID, googleBooksID, entityID,
			))
		},
	)
}

// FindOrCreateSeries looks up a series by name (case-insensitive) and returns
// it, creating a new one if it doesn't exist. Handles concurrent insert races
// gracefully.
func (d *DB) FindOrCreateSeries(ctx context.Context, name string) (*Series, error) {
	return findOrCreate(ctx, name, "series",
		NormalizeSeriesName, ErrInvalidSeriesName, ErrSeriesNameExists,
		d.GetSeriesByName,
		func(ctx context.Context, n string) (*Series, error) {
			return d.CreateSeries(ctx, n, nil, nil, nil)
		},
	)
}

// DeleteSeries removes the series with the given ID. Returns sql.ErrNoRows if
// no matching series exists.
func (d *DB) DeleteSeries(ctx context.Context, id string) error {
	slog.DebugContext(ctx, "db: deleting series", slog.String(otelkeys.ID, id))
	return d.execAffected(ctx, `DELETE FROM series WHERE id = $1`, id)
}
