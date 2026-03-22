package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
)

func backfillKoboTokenHashes(ctx context.Context, d *DB) error {
	hasTokenHash, err := hasColumn(ctx, d, "kobo_tokens", "token_hash")
	if err != nil || !hasTokenHash {
		slog.ErrorContext(ctx, "kobo_tokens table does not have token_hash column; cannot backfill token hashes")
		return fmt.Errorf("kobo_tokens table does not have token_hash column; cannot backfill token hashes")
	}

	hasToken, err := hasColumn(ctx, d, "kobo_tokens", "token")
	if err != nil || !hasToken {
		slog.ErrorContext(ctx, "kobo_tokens table does not have token column; cannot backfill token hashes")
		return fmt.Errorf("kobo_tokens table does not have token column; cannot backfill token hashes")
	}

	rows, err := d.QueryContext(ctx, `SELECT id, token FROM kobo_tokens WHERE token_hash IS NULL OR token_hash = ''`)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query kobo_tokens for backfilling token hashes", slog.Any("error", err))
		return fmt.Errorf("failed to query kobo_tokens for backfilling token hashes: %w", err)
	}
	defer rows.Close()

	type update struct {
		id   string
		hash string
	}

	var updates []update
	for rows.Next() {
		var id, token string
		if err := rows.Scan(&id, &token); err != nil {
			slog.ErrorContext(ctx, "Failed to scan kobo_tokens row for backfilling token hashes", slog.Any("error", err))
			return fmt.Errorf("failed to scan kobo_tokens row for backfilling token hashes: %w", err)
		}
		if token == "" {
			continue
		}
		hash := hashKoboToken(token)
		updates = append(updates, update{id: id, hash: hash})
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Error iterating kobo_tokens rows for backfilling token hashes", slog.Any("error", err))
		return fmt.Errorf("error iterating kobo_tokens rows for backfilling token hashes: %w", err)
	}

	if len(updates) == 0 {
		return nil
	}

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to begin transaction for backfilling kobo token hashes", slog.Any("error", err))
		return fmt.Errorf("failed to begin transaction for backfilling kobo token hashes: %w", err)
	}

	for _, u := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE kobo_tokens SET token_hash = $1, token = $1 WHERE id = $2`, u.hash, u.id); err != nil {
			_ = tx.Rollback()
			slog.ErrorContext(ctx, "Failed to update kobo_tokens for backfilling token hashes", slog.Any("error", err))
			return fmt.Errorf("failed to update kobo_tokens for backfilling token hashes: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "Failed to commit transaction for backfilling kobo token hashes", slog.Any("error", err))
		return fmt.Errorf("failed to commit transaction for backfilling kobo token hashes: %w", err)
	}
	return nil
}

func hashKoboToken(token string) string {
	h := sha256.Sum256([]byte(token)) // #nosec G401 -- not a password; high-entropy token
	return hex.EncodeToString(h[:])
}

func hasColumn(ctx context.Context, d *DB, table, column string) (bool, error) {
	switch d.Dialect {
	case DialectPostgres:
		var exists bool
		err := d.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)`,
			table, column,
		).Scan(&exists)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to query information_schema for column existence", slog.Any("error", err), slog.String("table", table), slog.String("column", column))
			return false, fmt.Errorf("failed to query information_schema for column existence: %w", err)
		}
		return exists, nil
	case DialectSQLite:
		rows, err := d.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			slog.ErrorContext(ctx, "Failed to query SQLite PRAGMA for table info", slog.Any("error", err), slog.String("table", table))
			return false, fmt.Errorf("failed to query SQLite PRAGMA for table info: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				cid     int
				name    string
				colType string
				notNull int
				dflt    sql.NullString
				pk      int
			)
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
				slog.ErrorContext(ctx, "Failed to scan SQLite PRAGMA table info row", slog.Any("error", err), slog.String("table", table))
				return false, fmt.Errorf("failed to scan SQLite PRAGMA table info row: %w", err)
			}
			if name == column {
				return true, nil
			}
		}
		if err := rows.Err(); err != nil {
			slog.ErrorContext(ctx, "Error iterating SQLite PRAGMA table info rows", slog.Any("error", err), slog.String("table", table))
			return false, fmt.Errorf("error iterating SQLite PRAGMA table info rows: %w", err)
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported dialect %q", d.Dialect)
	}
}
