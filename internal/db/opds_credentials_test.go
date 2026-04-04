package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestUserForOPDS(t *testing.T, d *DB, email string) *User {
	t.Helper()
	user, err := d.CreateUser(t.Context(), "Test User", email, "hashedpw")
	require.NoError(t, err, "CreateUser(%q)", email)
	return user
}

func TestOPDSCredential_UpsertAndGet(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForOPDS(t, d, "alice@example.com")
	ctx := t.Context()

	cred, err := d.UpsertOPDSCredential(ctx, user.ID, "alice", "hashval")
	require.NoError(t, err, "UpsertOPDSCredential")
	if cred.ID == "" {
		t.Error("cred.ID is empty")
	}
	if cred.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", cred.UserID, user.ID)
	}
	if cred.Username != "alice" {
		t.Errorf("Username = %q, want %q", cred.Username, "alice")
	}
	if cred.PasswordHash != "hashval" {
		t.Errorf("PasswordHash = %q, want %q", cred.PasswordHash, "hashval")
	}

	// Fetch by userID
	fetched, err := d.GetOPDSCredentialByUserID(ctx, user.ID)
	require.NoError(t, err, "GetOPDSCredentialByUserID")
	if fetched.Username != "alice" {
		t.Errorf("fetched Username = %q, want %q", fetched.Username, "alice")
	}

	// Fetch by username (lowercase, as the middleware always lowercases before calling)
	fetched2, err := d.GetOPDSCredentialByUsername(ctx, "alice")
	require.NoError(t, err, "GetOPDSCredentialByUsername")
	if fetched2.UserID != user.ID {
		t.Errorf("fetched2 UserID = %q, want %q", fetched2.UserID, user.ID)
	}
}

func TestOPDSCredential_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForOPDS(t, d, "bob@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user.ID, "bob", "hash1")
	require.NoError(t, err, "first upsert")

	updated, err := d.UpsertOPDSCredential(ctx, user.ID, "bob2", "hash2")
	require.NoError(t, err, "second upsert")
	if updated.Username != "bob2" {
		t.Errorf("updated Username = %q, want %q", updated.Username, "bob2")
	}
	if updated.PasswordHash != "hash2" {
		t.Errorf("updated PasswordHash = %q, want %q", updated.PasswordHash, "hash2")
	}
}

func TestOPDSCredential_UsernameConflict(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForOPDS(t, d, "user1@example.com")
	user2 := createTestUserForOPDS(t, d, "user2@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user1.ID, "shared", "hash1")
	require.NoError(t, err, "first upsert")

	_, err = d.UpsertOPDSCredential(ctx, user2.ID, "shared", "hash2")
	if err != ErrOPDSUsernameExists {
		t.Errorf("expected ErrOPDSUsernameExists, got %v", err)
	}
}

func TestOPDSCredential_GetByUserID_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetOPDSCredentialByUserID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestOPDSCredential_GetByUsername_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetOPDSCredentialByUsername(ctx, "nobody")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestOPDSCredential_Delete(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForOPDS(t, d, "del@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user.ID, "delme", "hash")
	require.NoError(t, err, "upsert")

	if err := d.DeleteOPDSCredential(ctx, user.ID); err != nil {
		require.NoError(t, err, "DeleteOPDSCredential")
	}

	_, err = d.GetOPDSCredentialByUserID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestOPDSCredential_Delete_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	err := d.DeleteOPDSCredential(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
