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
	require.NotEqual(t, "", cred.ID)
	require.Equal(t, user.ID, cred.UserID)
	require.Equal(t, "alice", cred.Username)
	require.Equal(t, "hashval", cred.PasswordHash)

	// Fetch by userID
	fetched, err := d.GetOPDSCredentialByUserID(ctx, user.ID)
	require.NoError(t, err, "GetOPDSCredentialByUserID")
	require.Equal(t, "alice", fetched.Username)

	// Fetch by username (lowercase, as the middleware always lowercases before calling)
	fetched2, err := d.GetOPDSCredentialByUsername(ctx, "alice")
	require.NoError(t, err, "GetOPDSCredentialByUsername")
	require.Equal(t, user.ID, fetched2.UserID)
}

func TestOPDSCredential_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForOPDS(t, d, "bob@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user.ID, "bob", "hash1")
	require.NoError(t, err, "first upsert")

	updated, err := d.UpsertOPDSCredential(ctx, user.ID, "bob2", "hash2")
	require.NoError(t, err, "second upsert")
	require.Equal(t, "bob2", updated.Username)
	require.Equal(t, "hash2", updated.PasswordHash)
}

func TestOPDSCredential_UsernameConflict(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForOPDS(t, d, "user1@example.com")
	user2 := createTestUserForOPDS(t, d, "user2@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user1.ID, "shared", "hash1")
	require.NoError(t, err, "first upsert")

	_, err = d.UpsertOPDSCredential(ctx, user2.ID, "shared", "hash2")
	require.ErrorIs(t, err, ErrOPDSUsernameExists)
}

func TestOPDSCredential_GetByUserID_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetOPDSCredentialByUserID(ctx, "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestOPDSCredential_GetByUsername_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetOPDSCredentialByUsername(ctx, "nobody")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestOPDSCredential_Delete(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForOPDS(t, d, "del@example.com")
	ctx := t.Context()

	_, err := d.UpsertOPDSCredential(ctx, user.ID, "delme", "hash")
	require.NoError(t, err, "upsert")

	require.NoError(t, d.DeleteOPDSCredential(ctx, user.ID), "DeleteOPDSCredential")

	_, err = d.GetOPDSCredentialByUserID(ctx, user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestOPDSCredential_Delete_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	err := d.DeleteOPDSCredential(ctx, "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
