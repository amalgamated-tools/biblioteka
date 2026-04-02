package db

import "context"

const protocolCredentialColumns = `id, user_id, username, password_hash, created_at, updated_at`

// ProtocolCredential represents a row in a protocol credential table.
type ProtocolCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    Timestamp `json:"created_at"`
	UpdatedAt    Timestamp `json:"updated_at"`
}

type protocolCredentialTable struct {
	name              string
	usernameUniqueCol string
	usernameUniqueIdx string
}

func scanProtocolCredential(row interface{ Scan(...any) error }) (*ProtocolCredential, error) {
	return scanRow(row, func(c *ProtocolCredential) []any {
		return []any{&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt}
	})
}

func (d *DB) getProtocolCredentialByUserID(ctx context.Context, tableName, userID string) (*ProtocolCredential, error) {
	return scanProtocolCredential(d.QueryRowContext(ctx,
		`SELECT `+protocolCredentialColumns+` FROM `+tableName+` WHERE user_id = $1`,
		userID,
	))
}

func (d *DB) getProtocolCredentialByUsername(ctx context.Context, tableName, username string) (*ProtocolCredential, error) {
	return scanProtocolCredential(d.QueryRowContext(ctx,
		`SELECT `+protocolCredentialColumns+` FROM `+tableName+` WHERE LOWER(username) = $1`,
		username,
	))
}

func (d *DB) upsertProtocolCredential(
	ctx context.Context,
	table protocolCredentialTable,
	userID, username, passwordHash string,
	errUsernameExists error,
) (*ProtocolCredential, error) {
	query := `INSERT INTO ` + table.name + ` (user_id, username, password_hash, updated_at)
		VALUES ($1, $2, $3, ` + d.now() + `)
		ON CONFLICT (user_id) DO UPDATE SET username = $2, password_hash = $3, updated_at = ` + d.now() + `
		RETURNING ` + protocolCredentialColumns

	cred, err := scanProtocolCredential(d.QueryRowContext(ctx, query, userID, username, passwordHash))
	if err != nil && isColumnUniqueViolation(err, table.usernameUniqueCol, table.usernameUniqueIdx) {
		return nil, errUsernameExists
	}
	return cred, err
}

func (d *DB) deleteProtocolCredential(ctx context.Context, tableName, userID string) error {
	return d.execAffected(ctx, `DELETE FROM `+tableName+` WHERE user_id = $1`, userID)
}
