package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// KoboToken represents a row in the kobo_tokens table.
type KoboToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"token_hash"`
	CreatedAt Timestamp `json:"created_at"`
}

const koboTokenColumns = `id, user_id, name, token_hash, created_at`

func scanKoboToken(row interface{ Scan(...any) error }) (*KoboToken, error) {
	return scanRow(row, func(t *KoboToken) []any {
		return []any{&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.CreatedAt}
	})
}

// CreateKoboToken inserts a new Kobo sync token hash and returns it.
func (d *DB) CreateKoboToken(ctx context.Context, userID, name, tokenHash string) (*KoboToken, error) {
	slog.DebugContext(ctx, "db: creating kobo token",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Name, name),
	)
	// Store the hash in the legacy token column to avoid persisting raw secrets.
	return scanKoboToken(d.QueryRowContext(ctx,
		`INSERT INTO kobo_tokens (user_id, name, token, token_hash) VALUES ($1, $2, $3, $4) RETURNING `+koboTokenColumns,
		userID, name, tokenHash, tokenHash,
	))
}

// GetKoboToken returns a Kobo token by ID scoped to the given user, or sql.ErrNoRows if not found.
func (d *DB) GetKoboToken(ctx context.Context, id, userID string) (*KoboToken, error) {
	slog.DebugContext(ctx, "db: fetching kobo token", slog.String(otelkeys.ID, id))
	return scanKoboToken(d.QueryRowContext(ctx,
		`SELECT `+koboTokenColumns+` FROM kobo_tokens WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// GetKoboTokenByHash returns a Kobo token record by its hash, or sql.ErrNoRows if not found.
func (d *DB) GetKoboTokenByHash(ctx context.Context, tokenHash string) (*KoboToken, error) {
	slog.DebugContext(ctx, "db: looking up kobo token by hash")
	return scanKoboToken(d.QueryRowContext(ctx,
		`SELECT `+koboTokenColumns+` FROM kobo_tokens WHERE token_hash = $1`,
		tokenHash,
	))
}

// ListKoboTokens returns all Kobo tokens for a user ordered by creation time (newest first).
func (d *DB) ListKoboTokens(ctx context.Context, userID string) ([]KoboToken, error) {
	slog.DebugContext(ctx, "db: listing kobo tokens", slog.String(otelkeys.UserID, userID))
	rows, err := d.QueryContext(ctx,
		`SELECT `+koboTokenColumns+` FROM kobo_tokens WHERE user_id = $1 ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanKoboToken)
}

// DeleteKoboToken removes a Kobo token by ID, scoped to the given user.
// Returns sql.ErrNoRows if the token doesn't exist or doesn't belong to the user.
func (d *DB) DeleteKoboToken(ctx context.Context, id, userID string) error {
	slog.DebugContext(ctx, "db: deleting kobo token",
		slog.String(otelkeys.ID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return d.execAffected(ctx, `DELETE FROM kobo_tokens WHERE id = $1 AND user_id = $2`, id, userID)
}
