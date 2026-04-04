package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScanProtocolCredential_HappyPath(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`
		CREATE TABLE protocol_credentials_test (
			id TEXT,
			user_id TEXT,
			username TEXT,
			password_hash TEXT,
			created_at TEXT,
			updated_at TEXT
		)
	`)
	require.NoError(t, err, "create table")

	_, err = d.Exec(`
		INSERT INTO protocol_credentials_test (id, user_id, username, password_hash, created_at, updated_at)
		VALUES ('cred-1', 'user-1', 'reader', 'hash', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z')
	`)
	require.NoError(t, err, "insert row")

	row := d.QueryRowContext(t.Context(),
		`SELECT `+protocolCredentialColumns+` FROM protocol_credentials_test WHERE id = $1`,
		"cred-1",
	)

	cred, err := scanProtocolCredential(row)
	require.NoError(t, err, "scanProtocolCredential")

	if cred.ID != "cred-1" {
		t.Errorf("ID = %q, want %q", cred.ID, "cred-1")
	}
	if cred.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", cred.UserID, "user-1")
	}
	if cred.Username != "reader" {
		t.Errorf("Username = %q, want %q", cred.Username, "reader")
	}
	if cred.PasswordHash != "hash" {
		t.Errorf("PasswordHash = %q, want %q", cred.PasswordHash, "hash")
	}
	wantCreatedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !cred.CreatedAt.Time.Equal(wantCreatedAt) {
		t.Errorf("CreatedAt = %q, want %q", cred.CreatedAt.Time.Format(time.RFC3339), wantCreatedAt.Format(time.RFC3339))
	}
	wantUpdatedAt := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !cred.UpdatedAt.Time.Equal(wantUpdatedAt) {
		t.Errorf("UpdatedAt = %q, want %q", cred.UpdatedAt.Time.Format(time.RFC3339), wantUpdatedAt.Format(time.RFC3339))
	}
}

func TestScanProtocolCredential_PropagatesNoRowsError(t *testing.T) {
	d := memDB(t)
	_, err := d.Exec(`CREATE TABLE protocol_credentials_test_norows (id TEXT)`)
	require.NoError(t, err, "create table")
	row := d.QueryRowContext(t.Context(), `SELECT id FROM protocol_credentials_test_norows WHERE id = $1`, "missing")

	cred, err := scanProtocolCredential(row)
	require.ErrorIs(t, err, sql.ErrNoRows)
	if cred != nil {
		require.Failf(t, "failed", "expected nil credential on error, got %+v", *cred)
	}
}

func TestScanProtocolCredential_PropagatesScanError(t *testing.T) {
	d := memDB(t)
	// Create a table with only one column — scanning 6 fields from it triggers a real scan error.
	_, err := d.Exec(`CREATE TABLE protocol_credentials_test_scan (id TEXT)`)
	require.NoError(t, err, "create table")
	_, err = d.Exec(`INSERT INTO protocol_credentials_test_scan (id) VALUES ('row-1')`)
	require.NoError(t, err, "insert row")

	row := d.QueryRowContext(t.Context(),
		`SELECT id FROM protocol_credentials_test_scan WHERE id = $1`, "row-1")

	cred, err := scanProtocolCredential(row)
	require.Error(t, err, "expected scan error, got nil")
	if errors.Is(err, sql.ErrNoRows) {
		require.Fail(t, "expected a scan error, not sql.ErrNoRows")
	}
	if cred != nil {
		require.Failf(t, "failed", "expected nil credential on error, got %+v", *cred)
	}
}
