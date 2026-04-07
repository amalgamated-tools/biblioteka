package db

import (
	"context"
	"database/sql"
	"errors"
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
