package db

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ProtocolCredential represents a row in a protocol credential table
// (opds_credentials, kosync_credentials). All protocol credential tables
// share the same six-column schema.
type ProtocolCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    Timestamp `json:"created_at"`
	UpdatedAt    Timestamp `json:"updated_at"`
}

const protocolCredentialColumns = `id, user_id, username, password_hash, created_at, updated_at`

func scanProtocolCredential(row interface{ Scan(...any) error }) (*ProtocolCredential, error) {
	return scanRow(row, func(c *ProtocolCredential) []any {
		return []any{&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt}
	})
}

// protocolCredentialConfig captures the table-specific details for a protocol
// credential. The CRUD query patterns are identical across protocols; only the
// table/index names, sentinel error, and log attributes differ.
//
// IMPORTANT: table, tableCol, and indexName are interpolated into SQL — they
// MUST be compile-time string literals, never user-supplied values.
type protocolCredentialConfig struct {
	table        string                 // e.g. "opds_credentials"
	tableCol     string                 // e.g. "opds_credentials.username"
	indexName    string                 // e.g. "idx_opds_credentials_username"
	errExists    error                  // e.g. ErrOPDSUsernameExists
	logPrefix    string                 // e.g. "OPDS"
	usernameAttr func(string) slog.Attr // builds the protocol-specific username log attribute
}

func getCredentialByUserID(d *DB, ctx context.Context, cfg protocolCredentialConfig, userID string) (*ProtocolCredential, error) {
	slog.DebugContext(ctx, "db: fetching "+cfg.logPrefix+" credential by user ID",
		slog.String(otelkeys.UserID, userID),
	)
	return scanProtocolCredential(d.QueryRowContext(ctx,
		`SELECT `+protocolCredentialColumns+` FROM `+cfg.table+` WHERE user_id = $1`,
		userID,
	))
}

func getCredentialByUsername(d *DB, ctx context.Context, cfg protocolCredentialConfig, username string) (*ProtocolCredential, error) {
	slog.DebugContext(ctx, "db: fetching "+cfg.logPrefix+" credential by username",
		cfg.usernameAttr(username),
	)
	return scanProtocolCredential(d.QueryRowContext(ctx,
		`SELECT `+protocolCredentialColumns+` FROM `+cfg.table+` WHERE LOWER(username) = $1`,
		username,
	))
}

func upsertCredential(d *DB, ctx context.Context, cfg protocolCredentialConfig, userID, username, passwordHash string) (*ProtocolCredential, error) {
	slog.DebugContext(ctx, "db: upserting "+cfg.logPrefix+" credential",
		slog.String(otelkeys.UserID, userID),
		cfg.usernameAttr(username),
	)

	query := `INSERT INTO ` + cfg.table + ` (user_id, username, password_hash, updated_at)
		VALUES ($1, $2, $3, ` + d.now() + `)
		ON CONFLICT (user_id) DO UPDATE SET username = $2, password_hash = $3, updated_at = ` + d.now() + `
		RETURNING ` + protocolCredentialColumns

	cred, err := scanProtocolCredential(d.QueryRowContext(ctx, query, userID, username, passwordHash))
	if err != nil && isColumnUniqueViolation(err, cfg.tableCol, cfg.indexName) {
		return nil, cfg.errExists
	}
	return cred, err
}

func deleteCredential(d *DB, ctx context.Context, cfg protocolCredentialConfig, userID string) error {
	slog.DebugContext(ctx, "db: deleting "+cfg.logPrefix+" credential",
		slog.String(otelkeys.UserID, userID),
	)
	return d.execAffected(ctx, `DELETE FROM `+cfg.table+` WHERE user_id = $1`, userID)
}
