package db

import (
	"context"
	"fmt"
	"log/slog"
)

// CheckFTSIntegrity verifies the FTS5 index is consistent with the books table.
// Returns nil when the index is healthy. Returns an error if the index is
// corrupted or if the check cannot be run.
//
// On PostgreSQL this is a no-op and always returns nil because PostgreSQL uses
// pg_trgm GIN indexes which are maintained automatically by the database engine.
func (d *DB) CheckFTSIntegrity(ctx context.Context) error {
	if d.Dialect != DialectSQLite {
		return nil
	}
	slog.DebugContext(ctx, "db: checking FTS5 index integrity")
	if _, err := d.ExecContext(ctx, `INSERT INTO books_fts(books_fts) VALUES ('integrity-check')`); err != nil {
		return fmt.Errorf("FTS5 integrity check failed: %w", err)
	}
	return nil
}

// RebuildFTS rebuilds the FTS5 index from the current books table contents.
// This is a full, synchronous rebuild; all indexed data is discarded and
// re-derived from the content table (books). Use after running VACUUM on the
// SQLite database, or whenever CheckFTSIntegrity reports corruption.
//
// On PostgreSQL this is a no-op because pg_trgm GIN indexes are maintained
// automatically.
func (d *DB) RebuildFTS(ctx context.Context) error {
	if d.Dialect != DialectSQLite {
		return nil
	}
	slog.InfoContext(ctx, "db: rebuilding FTS5 index")
	if _, err := d.ExecContext(ctx, `INSERT INTO books_fts(books_fts) VALUES ('rebuild')`); err != nil {
		return fmt.Errorf("FTS5 rebuild failed: %w", err)
	}
	slog.InfoContext(ctx, "db: FTS5 index rebuild complete")
	return nil
}
