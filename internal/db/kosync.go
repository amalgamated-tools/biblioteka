package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ErrKOSyncUsernameExists is returned when a KOSync username is already taken by another user.
var ErrKOSyncUsernameExists = errors.New("kosync username already exists")

// KOSyncCredential represents a row in the kosync_credentials table.
type KOSyncCredential struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    Timestamp `json:"created_at"`
	UpdatedAt    Timestamp `json:"updated_at"`
}

const kosyncCredentialColumns = `id, user_id, username, password_hash, created_at, updated_at`

func scanKOSyncCredential(row interface{ Scan(...any) error }) (*KOSyncCredential, error) {
	var c KOSyncCredential
	err := row.Scan(&c.ID, &c.UserID, &c.Username, &c.PasswordHash, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetKOSyncCredentialByUserID returns the KOSync credential for a user, or sql.ErrNoRows if not found.
func (d *DB) GetKOSyncCredentialByUserID(ctx context.Context, userID string) (*KOSyncCredential, error) {
	slog.DebugContext(ctx, "db: fetching KOSync credential by user ID", slog.String(otelkeys.UserID, userID))
	return scanKOSyncCredential(d.QueryRowContext(ctx,
		`SELECT `+kosyncCredentialColumns+` FROM kosync_credentials WHERE user_id = $1`,
		userID,
	))
}

// GetKOSyncCredentialByUsername returns the KOSync credential for a username, or sql.ErrNoRows if not found.
func (d *DB) GetKOSyncCredentialByUsername(ctx context.Context, username string) (*KOSyncCredential, error) {
	slog.DebugContext(ctx, "db: fetching KOSync credential by username", slog.String(otelkeys.KOSyncUsername, username))
	return scanKOSyncCredential(d.QueryRowContext(ctx,
		`SELECT `+kosyncCredentialColumns+` FROM kosync_credentials WHERE LOWER(username) = $1`,
		username,
	))
}

// UpsertKOSyncCredential creates or updates the KOSync credential for a user.
// Returns ErrKOSyncUsernameExists if the username is taken by a different user.
func (d *DB) UpsertKOSyncCredential(ctx context.Context, userID, username, passwordHash string) (*KOSyncCredential, error) {
	slog.DebugContext(ctx, "db: upserting KOSync credential",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.KOSyncUsername, username),
	)

	query := `INSERT INTO kosync_credentials (user_id, username, password_hash, updated_at)
		VALUES ($1, $2, $3, ` + d.now() + `)
		ON CONFLICT (user_id) DO UPDATE SET username = $2, password_hash = $3, updated_at = ` + d.now() + `
		RETURNING ` + kosyncCredentialColumns

	cred, err := scanKOSyncCredential(d.QueryRowContext(ctx, query, userID, username, passwordHash))
	if err != nil && isKOSyncUsernameUniqueViolation(err) {
		return nil, ErrKOSyncUsernameExists
	}
	return cred, err
}

// isKOSyncUsernameUniqueViolation checks if an error is a unique constraint violation
// specifically on the username column of kosync_credentials.
func isKOSyncUsernameUniqueViolation(err error) bool {
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed: kosync_credentials.username"
	// PostgreSQL: "...violates unique constraint \"idx_kosync_credentials_username\""
	return strings.Contains(msg, "kosync_credentials.username") ||
		strings.Contains(msg, "idx_kosync_credentials_username")
}

// DeleteKOSyncCredential removes the KOSync credential for a user.
func (d *DB) DeleteKOSyncCredential(ctx context.Context, userID string) error {
	slog.DebugContext(ctx, "db: deleting KOSync credential", slog.String(otelkeys.UserID, userID))
	res, err := d.ExecContext(ctx, `DELETE FROM kosync_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ReadingProgress represents a row in the reading_progress table.
type ReadingProgress struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Document   string    `json:"document"`
	Progress   string    `json:"progress"`
	Percentage float64   `json:"percentage"`
	Device     *string   `json:"device,omitempty"`
	DeviceID   *string   `json:"device_id,omitempty"`
	CreatedAt  Timestamp `json:"created_at"`
	UpdatedAt  Timestamp `json:"updated_at"`
}

const readingProgressColumns = `id, user_id, document, progress, percentage, device, device_id, created_at, updated_at`

func scanReadingProgress(row interface{ Scan(...any) error }) (*ReadingProgress, error) {
	var p ReadingProgress
	err := row.Scan(&p.ID, &p.UserID, &p.Document, &p.Progress, &p.Percentage, &p.Device, &p.DeviceID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetReadingProgress returns the reading progress for a user and document, or sql.ErrNoRows if not found.
func (d *DB) GetReadingProgress(ctx context.Context, userID, document string) (*ReadingProgress, error) {
	slog.DebugContext(ctx, "db: fetching reading progress",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Document, document),
	)
	return scanReadingProgress(d.QueryRowContext(ctx,
		`SELECT `+readingProgressColumns+` FROM reading_progress WHERE user_id = $1 AND document = $2`,
		userID, document,
	))
}

// UpsertReadingProgress creates or updates the reading progress for a user and document.
func (d *DB) UpsertReadingProgress(ctx context.Context, userID, document, progress string, percentage float64, device, deviceID *string) (*ReadingProgress, error) {
	slog.DebugContext(ctx, "db: upserting reading progress",
		slog.String(otelkeys.UserID, userID),
		slog.String(otelkeys.Document, document),
	)

	query := `INSERT INTO reading_progress (user_id, document, progress, percentage, device, device_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, ` + d.now() + `)
		ON CONFLICT (user_id, document) DO UPDATE SET
			progress = $3, percentage = $4, device = $5, device_id = $6, updated_at = ` + d.now() + `
		RETURNING ` + readingProgressColumns

	return scanReadingProgress(d.QueryRowContext(ctx, query, userID, document, progress, percentage, device, deviceID))
}
