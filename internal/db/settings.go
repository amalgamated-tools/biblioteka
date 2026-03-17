package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Setting holds a configuration key-value pair for bulk saves.
type Setting struct {
	Key   string
	Value string
}

type settingExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

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
	return d.setSetting(ctx, d.DB, key, value)
}

// SetSettings atomically upserts multiple setting key-value pairs.
func (d *DB) SetSettings(ctx context.Context, settings []Setting) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin settings transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "db: failed to roll back settings transaction", slog.Any(otelkeys.Error, rollbackErr))
		}
	}()

	for _, setting := range settings {
		slog.DebugContext(ctx, "db: saving setting", slog.String(otelkeys.Key, setting.Key))
		if err := d.setSetting(ctx, tx, setting.Key, setting.Value); err != nil {
			return fmt.Errorf("db: saving setting %q: %w", setting.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit settings transaction: %w", err)
	}
	committed = true
	return nil
}

func (d *DB) setSetting(ctx context.Context, execer settingExecer, key, value string) error {
	_, err := execer.ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, `+d.now()+`)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value,
	)
	return err
}
