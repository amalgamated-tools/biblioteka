package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, d *DB) *User {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", t.Name()+"@example.com", "password1")
	require.NoError(t, err, "create user")
	return u
}

func TestCreateAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	key, err := d.CreateAPIKey(t.Context(), user.ID, "CI Pipeline", "hash123", "bib_abcd")
	require.NoError(t, err, "CreateAPIKey() error")
	require.NotEqual(t, "", key.ID)
	require.Equal(t, user.ID, key.UserID)
	require.Equal(t, "CI Pipeline", key.Name)
	require.Equal(t, "hash123", key.KeyHash)
	require.Equal(t, "bib_abcd", key.KeyPrefix)
	require.Nil(t, key.LastUsedAt)
	require.False(t, key.CreatedAt.IsZero())
}

func TestListAPIKeys_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	keys, err := d.ListAPIKeys(t.Context(), user.ID)
	require.NoError(t, err, "ListAPIKeys() error")
	require.Len(t, keys, 0)
}

func TestListAPIKeys_ReturnsUserKeysOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other@example.com", "password2")
	require.NoError(t, err, "create user2")

	// Create keys for both users.
	_, err = d.CreateAPIKey(t.Context(), user1.ID, "Key A", "hashA", "prefixA")
	require.NoError(t, err, "CreateAPIKey A")
	_, err = d.CreateAPIKey(t.Context(), user1.ID, "Key B", "hashB", "prefixB")
	require.NoError(t, err, "CreateAPIKey B")
	_, err = d.CreateAPIKey(t.Context(), u2.ID, "Key C", "hashC", "prefixC")
	require.NoError(t, err, "CreateAPIKey C")

	keys, err := d.ListAPIKeys(t.Context(), user1.ID)
	require.NoError(t, err, "ListAPIKeys() error")
	require.Len(t, keys, 2)

	// user2 should only see their own key.
	keys2, err := d.ListAPIKeys(t.Context(), u2.ID)
	require.NoError(t, err, "ListAPIKeys(user2) error")
	require.Len(t, keys2, 1)
}

func TestGetAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Fetch Me", "hashFetch", "prefFetch")
	require.NoError(t, err, "CreateAPIKey() error")

	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	require.NoError(t, err, "GetAPIKey() error")
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Fetch Me", got.Name)
}

func TestGetAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other2@example.com", "password2")
	require.NoError(t, err, "create user2")

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Private", "hashPriv", "prefPriv")
	require.NoError(t, err, "CreateAPIKey() error")

	_, err = d.GetAPIKey(t.Context(), created.ID, u2.ID)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeleteAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Delete Me", "hashDel", "prefDel")
	require.NoError(t, err, "CreateAPIKey() error")

	require.NoError(t, d.DeleteAPIKey(t.Context(), created.ID, user.ID), "DeleteAPIKey() error")

	// Should be gone.
	_, err = d.GetAPIKey(t.Context(), created.ID, user.ID)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteAPIKey(t.Context(), "nonexistent", user.ID)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeleteAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other3@example.com", "password2")
	require.NoError(t, err, "create user2")

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Protected", "hashProt", "prefProt")
	require.NoError(t, err, "CreateAPIKey() error")

	err = d.DeleteAPIKey(t.Context(), created.ID, u2.ID)
	require.Equal(t, sql.ErrNoRows, err)

	// Key should still exist for original user.
	_, err = d.GetAPIKey(t.Context(), created.ID, user1.ID)
	require.NoError(t, err)
}

func TestValidateAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	fullKey := "bib_deadbeef12345678"
	h := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(h[:])

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Auth Key", keyHash, fullKey[:8])
	require.NoError(t, err, "CreateAPIKey() error")

	userID, keyID, err := d.ValidateAPIKey(t.Context(), keyHash)
	require.NoError(t, err, "ValidateAPIKey() error")
	require.Equal(t, user.ID, userID)
	require.Equal(t, created.ID, keyID)
}

func TestValidateAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, _, err := d.ValidateAPIKey(t.Context(), "nonexistenthash")
	require.Equal(t, sql.ErrNoRows, err)
}

func TestTouchAPIKeyLastUsed(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Touch Me", "hashTouch", "prefTouch")
	require.NoError(t, err, "CreateAPIKey() error")
	require.Nil(t, created.LastUsedAt)

	require.NoError(t, d.TouchAPIKeyLastUsed(t.Context(), created.ID), "TouchAPIKeyLastUsed() error")

	// Re-fetch and verify last_used_at is now set.
	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	require.NoError(t, err, "GetAPIKey() error")
	require.NotNil(t, got.LastUsedAt)
}

func TestGetAPIKeyByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Hash Lookup", "uniqueHash42", "prefHash")
	require.NoError(t, err, "CreateAPIKey() error")

	got, err := d.GetAPIKeyByHash(t.Context(), "uniqueHash42")
	require.NoError(t, err, "GetAPIKeyByHash() error")
	require.Equal(t, created.ID, got.ID)
}

func TestGetAPIKeyByHash_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAPIKeyByHash(t.Context(), "nosuchhash")
	require.Equal(t, sql.ErrNoRows, err)
}
