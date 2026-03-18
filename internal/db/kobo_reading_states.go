package db

import (
	"context"
	"log/slog"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// KoboReadingState represents a user's reading progress for a book.
type KoboReadingState struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	BookID         string    `json:"book_id"`
	Status         string    `json:"status"` // ReadyToRead, Reading, Finished
	PercentRead    *float64  `json:"percent_read"`
	LocationValue  *string   `json:"location_value"`
	LocationType   *string   `json:"location_type"`
	LocationSource *string   `json:"location_source"`
	CreatedAt      Timestamp `json:"created_at"`
	UpdatedAt      Timestamp `json:"updated_at"`
}

const koboReadingStateColumns = `id, user_id, book_id, status, percent_read, location_value, location_type, location_source, created_at, updated_at`

func scanKoboReadingState(row interface{ Scan(...any) error }) (*KoboReadingState, error) {
	var s KoboReadingState
	err := row.Scan(&s.ID, &s.UserID, &s.BookID, &s.Status, &s.PercentRead,
		&s.LocationValue, &s.LocationType, &s.LocationSource, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetKoboReadingState returns the reading state for a specific user+book pair,
// or sql.ErrNoRows if not found.
func (d *DB) GetKoboReadingState(ctx context.Context, userID, bookID string) (*KoboReadingState, error) {
	slog.DebugContext(ctx, "db: fetching kobo reading state",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.BookID, bookID),
	)
	return scanKoboReadingState(d.QueryRowContext(ctx,
		`SELECT `+koboReadingStateColumns+` FROM kobo_reading_states WHERE user_id = $1 AND book_id = $2`,
		userID, bookID,
	))
}

// UpsertKoboReadingState creates or updates the reading state for a user+book pair.
func (d *DB) UpsertKoboReadingState(ctx context.Context, userID, bookID, status string, percentRead *float64, locationValue, locationType, locationSource *string) (*KoboReadingState, error) {
	slog.DebugContext(ctx, "db: upserting kobo reading state",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.Status, status),
	)

	q := `INSERT INTO kobo_reading_states (user_id, book_id, status, percent_read, location_value, location_type, location_source)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)
	      ON CONFLICT (user_id, book_id) DO UPDATE SET
	          status = EXCLUDED.status,
	          percent_read = EXCLUDED.percent_read,
	          location_value = EXCLUDED.location_value,
	          location_type = EXCLUDED.location_type,
	          location_source = EXCLUDED.location_source,
	          updated_at = ` + d.now() + `
	      RETURNING ` + koboReadingStateColumns

	return scanKoboReadingState(d.QueryRowContext(ctx, q,
		userID, bookID, status, percentRead, locationValue, locationType, locationSource,
	))
}

// ListKoboReadingStatesSince returns reading states updated after the given time for a user.
// If since is the zero time, all reading states for the user are returned.
func (d *DB) ListKoboReadingStatesSince(ctx context.Context, userID string, since time.Time) ([]KoboReadingState, error) {
	slog.DebugContext(ctx, "db: listing kobo reading states since",
		slog.String(otelkeys.UserID, userID),
	)

	if since.IsZero() {
		rows, err := d.QueryContext(ctx,
			`SELECT `+koboReadingStateColumns+` FROM kobo_reading_states WHERE user_id = $1 ORDER BY updated_at ASC`,
			userID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanKoboReadingStateRows(rows)
	}

	var sinceParam any
	if d.Dialect == DialectPostgres {
		sinceParam = since
	} else {
		// SQLite stores datetimes as "YYYY-MM-DD HH:MM:SS"; use matching format
		// to ensure correct string-based datetime comparison.
		sinceParam = since.UTC().Format("2006-01-02 15:04:05")
	}
	rows, err := d.QueryContext(ctx,
		`SELECT `+koboReadingStateColumns+` FROM kobo_reading_states WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at ASC`,
		userID, sinceParam,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKoboReadingStateRows(rows)
}

type koboReadingStateScanner interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanKoboReadingStateRows(rows koboReadingStateScanner) ([]KoboReadingState, error) {
	var states []KoboReadingState
	for rows.Next() {
		s, err := scanKoboReadingState(rows)
		if err != nil {
			return nil, err
		}
		states = append(states, *s)
	}
	return states, rows.Err()
}
