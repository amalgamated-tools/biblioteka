package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// deferRollback is intended to be used with defer to roll back a transaction.
// It silently ignores sql.ErrTxDone (which means the transaction was already
// committed or rolled back) and logs a warning for any other rollback error.
func deferRollback(ctx context.Context, tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.WarnContext(ctx, "failed to rollback transaction", slog.Any(otelkeys.Error, err))
	}
}

// WithTx executes fn inside a database transaction. If fn returns a non-nil
// error the transaction is rolled back; otherwise it is committed.
func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer deferRollback(ctx, tx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
