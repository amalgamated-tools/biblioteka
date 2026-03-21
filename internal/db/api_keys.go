package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// APIKey represents a row in the api_keys table.
type APIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	KeyPrefix  string     `json:"key_prefix"`
	LastUsedAt *Timestamp `json:"last_used_at"`
	CreatedAt  Timestamp  `json:"created_at"`
}

const apiKeyColumns = `id, user_id, name, key_hash, key_prefix, last_used_at, created_at`

func scanAPIKey(ctx context.Context, row interface{ Scan(...any) error }) (*APIKey, error) {
	var k APIKey
	err := row.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to scan API key", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}
	return &k, nil
}

// CreateAPIKey inserts a new API key and returns it.
func (d *DB) CreateAPIKey(ctx context.Context, userID, name, keyHash, keyPrefix string) (*APIKey, error) {
	slog.DebugContext(ctx, "db: creating api key",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Name, name),
	)
	return scanAPIKey(ctx, d.QueryRowContext(ctx,
		`INSERT INTO api_keys (user_id, name, key_hash, key_prefix) VALUES ($1, $2, $3, $4) RETURNING `+apiKeyColumns,
		userID, name, keyHash, keyPrefix,
	))
}

// ListAPIKeys returns all API keys for a user, ordered by creation time (newest first).
func (d *DB) ListAPIKeys(ctx context.Context, userID string) ([]APIKey, error) {
	slog.DebugContext(ctx, "db: listing api keys", slog.String(otelkeys.UserID, userID))
	rows, err := d.QueryContext(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to query API keys", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		k, err := scanAPIKey(ctx, rows)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to scan API key from list", slog.Any(otelkeys.Error, err))
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, *k)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "Failed to iterate API key rows", slog.Any(otelkeys.Error, err))
		return nil, fmt.Errorf("failed to iterate API key rows: %w", err)
	}
	return keys, nil
}

// GetAPIKey returns a single API key by ID and user ID.
func (d *DB) GetAPIKey(ctx context.Context, id, userID string) (*APIKey, error) {
	slog.DebugContext(ctx, "db: fetching api key", slog.String(otelkeys.ID, id))
	return scanAPIKey(ctx, d.QueryRowContext(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// DeleteAPIKey removes an API key by ID, scoped to the given user.
// Returns sql.ErrNoRows if the key doesn't exist or doesn't belong to the user.
func (d *DB) DeleteAPIKey(ctx context.Context, id, userID string) error {
	slog.DebugContext(ctx, "db: deleting api key",
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.UserID, userID),
	)
	res, err := d.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to delete API key", slog.Any(otelkeys.Error, err))
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		slog.DebugContext(
			ctx,
			"No API key deleted (not found or not owned by user)",
			slog.String(otelkeys.ID, id),
			slog.String(otelkeys.UserID, userID),
		)
		return sql.ErrNoRows
	}
	return nil
}

// GetAPIKeyByHash returns an API key by its SHA-256 hash. Used during authentication.
func (d *DB) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	slog.DebugContext(ctx, "db: looking up api key by hash")
	return scanAPIKey(ctx, d.QueryRowContext(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE key_hash = $1`,
		keyHash,
	))
}

// TouchAPIKeyLastUsed updates the last_used_at timestamp for the given API key.
func (d *DB) TouchAPIKeyLastUsed(ctx context.Context, id string) error {
	_, err := d.ExecContext(ctx, `UPDATE api_keys SET last_used_at = `+d.now()+` WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to touch API key last_used_at",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.ID, id),
		)
		return fmt.Errorf("failed to touch API key last_used_at: %w", err)
	}
	return nil
}

// ValidateAPIKey looks up an API key by hash and returns the user ID and key ID.
// Returns sql.ErrNoRows if no matching key is found.
func (d *DB) ValidateAPIKey(ctx context.Context, keyHash string) (string, string, error) {
	k, err := d.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to validate API key", slog.Any(otelkeys.Error, err))
		return "", "", err
	}
	return k.UserID, k.ID, nil
}
