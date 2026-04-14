package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// PasskeyCredential represents a row in the passkey_credentials table.
type PasskeyCredential struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Name           string    `json:"name"`
	CredentialID   string    `json:"-"`
	CredentialData string    `json:"-"`
	AAGUID         string    `json:"aaguid"`
	CreatedAt      Timestamp `json:"created_at"`
}

const passkeyCredentialColumns = `id, user_id, name, credential_id, credential_data, aaguid, created_at`

func scanPasskeyCredential(row interface{ Scan(...any) error }) (*PasskeyCredential, error) {
	return scanRow(row, func(c *PasskeyCredential) []any {
		return []any{&c.ID, &c.UserID, &c.Name, &c.CredentialID, &c.CredentialData, &c.AAGUID, &c.CreatedAt}
	})
}

// CreatePasskeyCredential inserts a new passkey credential and returns it.
func (d *DB) CreatePasskeyCredential(ctx context.Context, userID, name, credentialID, credentialData, aaguid string) (*PasskeyCredential, error) {
	slog.DebugContext(ctx, "db: creating passkey credential",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.PasskeyCredentialID, credentialID),
	)
	return scanPasskeyCredential(d.QueryRowContext(ctx,
		`INSERT INTO passkey_credentials (user_id, name, credential_id, credential_data, aaguid) VALUES ($1, $2, $3, $4, $5) RETURNING `+passkeyCredentialColumns,
		userID, name, credentialID, credentialData, aaguid,
	))
}

// GetPasskeyCredential returns a passkey credential by ID, scoped to the given user.
// Returns sql.ErrNoRows if not found.
func (d *DB) GetPasskeyCredential(ctx context.Context, id, userID string) (*PasskeyCredential, error) {
	slog.DebugContext(ctx, "db: fetching passkey credential",
		slog.String(otelkeys.PasskeyCredentialID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanPasskeyCredential(d.QueryRowContext(ctx,
		`SELECT `+passkeyCredentialColumns+` FROM passkey_credentials WHERE id = $1 AND user_id = $2`,
		id, userID,
	))
}

// GetPasskeyCredentialByCredentialID returns a passkey credential by its raw credential ID.
// Returns sql.ErrNoRows if not found.
func (d *DB) GetPasskeyCredentialByCredentialID(ctx context.Context, credentialID string) (*PasskeyCredential, error) {
	slog.DebugContext(ctx, "db: fetching passkey credential by credential ID")
	return scanPasskeyCredential(d.QueryRowContext(ctx,
		`SELECT `+passkeyCredentialColumns+` FROM passkey_credentials WHERE credential_id = $1`,
		credentialID,
	))
}

// ListPasskeyCredentials returns all passkey credentials for a user, ordered by creation time (newest first).
func (d *DB) ListPasskeyCredentials(ctx context.Context, userID string) ([]PasskeyCredential, error) {
	slog.DebugContext(ctx, "db: listing passkey credentials", slog.String(otelkeys.UserID, userID))
	rows, err := d.QueryContext(ctx,
		`SELECT `+passkeyCredentialColumns+` FROM passkey_credentials WHERE user_id = $1 ORDER BY created_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanPasskeyCredential)
}

// UpdatePasskeyCredentialData updates the stored credential JSON (e.g. after sign count update following authentication).
func (d *DB) UpdatePasskeyCredentialData(ctx context.Context, credentialID, credentialData string) error {
	slog.DebugContext(ctx, "db: updating passkey credential data")
	_, err := d.ExecContext(ctx,
		`UPDATE passkey_credentials SET credential_data = $1 WHERE credential_id = $2`,
		credentialData, credentialID,
	)
	return err
}

// DeletePasskeyCredential removes a passkey credential by ID, scoped to the given user.
// Returns sql.ErrNoRows if the credential does not exist or does not belong to the user.
func (d *DB) DeletePasskeyCredential(ctx context.Context, id, userID string) error {
	slog.DebugContext(ctx, "db: deleting passkey credential",
		slog.String(otelkeys.PasskeyCredentialID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return d.execAffected(ctx,
		`DELETE FROM passkey_credentials WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
}

// PasskeyChallenge represents a row in the passkey_challenges table.
// Challenges are ephemeral: they are created at the start of a WebAuthn ceremony
// and atomically retrieved and deleted when the ceremony completes or expires.
type PasskeyChallenge struct {
	ID          string    `json:"id"`
	UserID      *string   `json:"user_id"`
	SessionData string    `json:"-"`
	ExpiresAt   Timestamp `json:"expires_at"`
	CreatedAt   Timestamp `json:"created_at"`
}

const passkeyChallengeColumns = `id, user_id, session_data, expires_at, created_at`

func scanPasskeyChallenge(row interface{ Scan(...any) error }) (*PasskeyChallenge, error) {
	return scanRow(row, func(c *PasskeyChallenge) []any {
		return []any{&c.ID, &c.UserID, &c.SessionData, &c.ExpiresAt, &c.CreatedAt}
	})
}

// CreatePasskeyChallenge inserts a new challenge and returns it.
// userID is nil for login challenges (user not yet known).
func (d *DB) CreatePasskeyChallenge(ctx context.Context, userID *string, sessionData string, expiresAt time.Time) (*PasskeyChallenge, error) {
	slog.DebugContext(ctx, "db: creating passkey challenge")
	return scanPasskeyChallenge(d.QueryRowContext(ctx,
		`INSERT INTO passkey_challenges (user_id, session_data, expires_at) VALUES ($1, $2, $3) RETURNING `+passkeyChallengeColumns,
		userID, sessionData, expiresAt,
	))
}

// GetAndDeletePasskeyChallenge atomically retrieves and deletes a passkey challenge.
// Returns sql.ErrNoRows if the challenge does not exist.
func (d *DB) GetAndDeletePasskeyChallenge(ctx context.Context, id string) (*PasskeyChallenge, error) {
	slog.DebugContext(ctx, "db: getting and deleting passkey challenge", slog.String(otelkeys.PasskeySessionID, id))

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer deferRollback(ctx, tx)

	var c PasskeyChallenge
	err = tx.QueryRowContext(ctx,
		`SELECT `+passkeyChallengeColumns+` FROM passkey_challenges WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.UserID, &c.SessionData, &c.ExpiresAt, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("fetch challenge: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM passkey_challenges WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("delete challenge: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &c, nil
}

// DeleteExpiredPasskeyChallenges removes all expired passkey challenges.
func (d *DB) DeleteExpiredPasskeyChallenges(ctx context.Context) error {
	slog.DebugContext(ctx, "db: deleting expired passkey challenges")
	_, err := d.ExecContext(ctx, `DELETE FROM passkey_challenges WHERE expires_at < `+d.now())
	return err
}
