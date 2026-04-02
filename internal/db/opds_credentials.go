package db

import (
	"context"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ErrOPDSUsernameExists is returned when an OPDS username is already taken by another user.
var ErrOPDSUsernameExists = errors.New("opds username already exists")

// OPDSCredential represents a row in the opds_credentials table.
type OPDSCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    Timestamp `json:"created_at"`
	UpdatedAt    Timestamp `json:"updated_at"`
}

const opdsCredentialColumns = `id, user_id, username, password_hash, created_at, updated_at`

func scanOPDSCredential(row interface{ Scan(...any) error }) (*OPDSCredential, error) {
	return scanRow(row, func(c *OPDSCredential) []any {
		return []any{&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt}
	})
}

// GetOPDSCredentialByUserID returns the OPDS credential for a user, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUserID(ctx context.Context, userID string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: fetching OPDS credential by user ID", slog.String(otelkeys.UserID, userID))
	return scanOPDSCredential(d.QueryRowContext(ctx,
		`SELECT `+opdsCredentialColumns+` FROM opds_credentials WHERE user_id = $1`,
		userID,
	))
}

// GetOPDSCredentialByUsername returns the OPDS credential for a username, or sql.ErrNoRows if not found.
func (d *DB) GetOPDSCredentialByUsername(ctx context.Context, username string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: fetching OPDS credential by username", slog.String(otelkeys.OPDSUsername, username))
	return scanOPDSCredential(d.QueryRowContext(ctx,
		`SELECT `+opdsCredentialColumns+` FROM opds_credentials WHERE LOWER(username) = $1`,
		username,
	))
}

// UpsertOPDSCredential creates or updates the OPDS credential for a user.
// Returns ErrOPDSUsernameExists if the username is taken by a different user.
func (d *DB) UpsertOPDSCredential(ctx context.Context, userID, username, passwordHash string) (*OPDSCredential, error) {
	slog.DebugContext(ctx, "db: upserting OPDS credential",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.OPDSUsername, username),
	)

	query := `INSERT INTO opds_credentials (user_id, username, password_hash, updated_at)
		VALUES ($1, $2, $3, ` + d.now() + `)
		ON CONFLICT (user_id) DO UPDATE SET username = $2, password_hash = $3, updated_at = ` + d.now() + `
		RETURNING ` + opdsCredentialColumns

	cred, err := scanOPDSCredential(d.QueryRowContext(ctx, query, userID, username, passwordHash))
	if err != nil && isColumnUniqueViolation(err, "opds_credentials.username", "idx_opds_credentials_username") {
		return nil, ErrOPDSUsernameExists
	}
	return cred, err
}

// DeleteOPDSCredential removes the OPDS credential for a user.
func (d *DB) DeleteOPDSCredential(ctx context.Context, userID string) error {
	slog.DebugContext(ctx, "db: deleting OPDS credential", slog.String(otelkeys.UserID, userID))
	return d.execAffected(ctx, `DELETE FROM opds_credentials WHERE user_id = $1`, userID)
}
