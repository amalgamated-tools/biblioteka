package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestUserForKOSync(t *testing.T, d *DB, email string) *User {
	t.Helper()
	user, err := d.CreateUser(t.Context(), "Test User", email, "hashedpw")
	require.NoError(t, err, "CreateUser(%q)", email)
	return user
}

// ---- KOSyncCredential tests ----

func TestKOSyncCredential_UpsertAndGet(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "alice@example.com")
	ctx := t.Context()

	cred, err := d.UpsertKOSyncCredential(ctx, user.ID, "alice", "hashval")
	require.NoError(t, err, "UpsertKOSyncCredential")
	require.NotEqual(t, "", cred.ID)
	require.Equal(t, user.ID, cred.UserID)
	require.Equal(t, "alice", cred.Username)
	require.Equal(t, "hashval", cred.PasswordHash)

	// Fetch by userID
	fetched, err := d.GetKOSyncCredentialByUserID(ctx, user.ID)
	require.NoError(t, err, "GetKOSyncCredentialByUserID")
	require.Equal(t, "alice", fetched.Username)

	// Fetch by username (lowercase, as the middleware always lowercases before calling)
	fetched2, err := d.GetKOSyncCredentialByUsername(ctx, "alice")
	require.NoError(t, err, "GetKOSyncCredentialByUsername")
	require.Equal(t, user.ID, fetched2.UserID)
}

func TestKOSyncCredential_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "bob@example.com")
	ctx := t.Context()

	_, err := d.UpsertKOSyncCredential(ctx, user.ID, "bob", "hash1")
	require.NoError(t, err, "first upsert")

	updated, err := d.UpsertKOSyncCredential(ctx, user.ID, "bob2", "hash2")
	require.NoError(t, err, "second upsert")
	require.Equal(t, "bob2", updated.Username)
	require.Equal(t, "hash2", updated.PasswordHash)
}

func TestKOSyncCredential_UsernameConflict(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForKOSync(t, d, "user1@example.com")
	user2 := createTestUserForKOSync(t, d, "user2@example.com")
	ctx := t.Context()

	_, err := d.UpsertKOSyncCredential(ctx, user1.ID, "shared", "hash1")
	require.NoError(t, err, "first upsert")

	_, err = d.UpsertKOSyncCredential(ctx, user2.ID, "shared", "hash2")
	require.ErrorIs(t, err, ErrKOSyncUsernameExists)
}

func TestKOSyncCredential_GetByUserID_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetKOSyncCredentialByUserID(ctx, "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestKOSyncCredential_GetByUsername_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	_, err := d.GetKOSyncCredentialByUsername(ctx, "nobody")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestKOSyncCredential_Delete(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "del@example.com")
	ctx := t.Context()

	_, err := d.UpsertKOSyncCredential(ctx, user.ID, "delme", "hash")
	require.NoError(t, err, "upsert")

	require.NoError(t, d.DeleteKOSyncCredential(ctx, user.ID), "DeleteKOSyncCredential")

	_, err = d.GetKOSyncCredentialByUserID(ctx, user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestKOSyncCredential_Delete_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	err := d.DeleteKOSyncCredential(ctx, "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// ---- ReadingProgress tests ----

func TestReadingProgress_UpsertAndGet(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "reader@example.com")
	ctx := t.Context()

	device := "MyKindle"
	deviceID := "device-001"
	p, err := d.UpsertReadingProgress(ctx, user.ID, "doc123", "/body/p[1]", 0.25, &device, &deviceID)
	require.NoError(t, err, "UpsertReadingProgress")
	require.NotEqual(t, "", p.ID)
	require.Equal(t, "doc123", p.Document)
	require.Equal(t, "/body/p[1]", p.Progress)
	require.Equal(t, 0.25, p.Percentage)
	require.NotNil(t, p.Device)
	require.Equal(t, "MyKindle", *p.Device)
	require.NotNil(t, p.DeviceID)
	require.Equal(t, "device-001", *p.DeviceID)

	// Fetch
	fetched, err := d.GetReadingProgress(ctx, user.ID, "doc123")
	require.NoError(t, err, "GetReadingProgress")
	require.Equal(t, "/body/p[1]", fetched.Progress)
}

func TestReadingProgress_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "reader2@example.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user.ID, "doc456", "/body/p[1]", 0.1, nil, nil)
	require.NoError(t, err, "first upsert")

	updated, err := d.UpsertReadingProgress(ctx, user.ID, "doc456", "/body/p[10]", 0.5, nil, nil)
	require.NoError(t, err, "second upsert")
	require.Equal(t, "/body/p[10]", updated.Progress)
	require.Equal(t, 0.5, updated.Percentage)
}

func TestReadingProgress_GetNotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "nobody@example.com")
	ctx := t.Context()

	_, err := d.GetReadingProgress(ctx, user.ID, "missing-doc")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestReadingProgress_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForKOSync(t, d, "user1@books.com")
	user2 := createTestUserForKOSync(t, d, "user2@books.com")
	ctx := t.Context()

	_, err := d.UpsertReadingProgress(ctx, user1.ID, "shared-doc", "/body/p[5]", 0.3, nil, nil)
	require.NoError(t, err, "upsert user1")

	// user2 should see no progress for the same document
	_, err = d.GetReadingProgress(ctx, user2.ID, "shared-doc")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
