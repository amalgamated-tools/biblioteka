package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// GetSetting retrieves a setting value by key.
// Returns sql.ErrNoRows if the key does not exist.
func (d *DB) GetSetting(ctx context.Context, key string) (string, error) {
	slog.DebugContext(ctx, "db: fetching setting", slog.String(otelkeys.Key, key))
	var value string
	err := d.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	return value, err
}

// SetSetting upserts a setting key-value pair.
func (d *DB) SetSetting(ctx context.Context, key, value string) error {
	slog.DebugContext(ctx, "db: saving setting", slog.String(otelkeys.Key, key))
	_, err := d.ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, `+d.now()+`)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value,
	)
	return err
}
