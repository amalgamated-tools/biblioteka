package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func backfillKoboTokenHashes(ctx context.Context, d *DB) error {
	hasTokenHash, err := hasColumn(ctx, d, "kobo_tokens", "token_hash")
	if err != nil || !hasTokenHash {
		return err
	}

	hasToken, err := hasColumn(ctx, d, "kobo_tokens", "token")
	if err != nil || !hasToken {
		return err
	}

	rows, err := d.QueryContext(ctx, `SELECT id, token FROM kobo_tokens WHERE token_hash IS NULL OR token_hash = ''`)
	if err != nil {
		return err
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
			return err
		}
		if token == "" {
			continue
		}
		hash := hashKoboToken(token)
		updates = append(updates, update{id: id, hash: hash})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, u := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE kobo_tokens SET token_hash = $1, token = $1 WHERE id = $2`, u.hash, u.id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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
		return exists, err
	case DialectSQLite:
		var count int
		if err := d.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`,
			table, column,
		).Scan(&count); err != nil {
			return false, err
		}
		return count > 0, nil
	default:
		return false, fmt.Errorf("unsupported dialect %q", d.Dialect)
	}
}
