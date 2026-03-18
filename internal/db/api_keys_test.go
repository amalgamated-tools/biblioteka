package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"
)

func createTestUser(t *testing.T, d *DB) *User {
	t.Helper()
	u, err := d.CreateUser(context.Background(), "Test User", t.Name()+"@example.com", "password1")
	if err != nil {
		failNowf(t, "create user: %v", err)
	}
	return u
}

func TestCreateAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	key, err := d.CreateAPIKey(t.Context(), user.ID, "CI Pipeline", "hash123", "bib_abcd")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}
	if key.ID == "" {
		fail(t, "expected non-empty ID")
	}
	if key.UserID != user.ID {
		failf(t, "UserID = %q, want %q", key.UserID, user.ID)
	}
	if key.Name != "CI Pipeline" {
		failf(t, "Name = %q, want %q", key.Name, "CI Pipeline")
	}
	if key.KeyHash != "hash123" {
		failf(t, "KeyHash = %q, want %q", key.KeyHash, "hash123")
	}
	if key.KeyPrefix != "bib_abcd" {
		failf(t, "KeyPrefix = %q, want %q", key.KeyPrefix, "bib_abcd")
	}
	if key.LastUsedAt != nil {
		failf(t, "LastUsedAt = %v, want nil", key.LastUsedAt)
	}
	if key.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
	}
}

func TestListAPIKeys_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	keys, err := d.ListAPIKeys(t.Context(), user.ID)
	if err != nil {
		failNowf(t, "ListAPIKeys() error: %v", err)
	}
	if len(keys) != 0 {
		failf(t, "len = %d, want 0", len(keys))
	}
}

func TestListAPIKeys_ReturnsUserKeysOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other@example.com", "password2")
	if err != nil {
		failNowf(t, "create user2: %v", err)
	}

	// Create keys for both users.
	if _, err := d.CreateAPIKey(t.Context(), user1.ID, "Key A", "hashA", "prefixA"); err != nil {
		failNowf(t, "CreateAPIKey A: %v", err)
	}
	if _, err := d.CreateAPIKey(t.Context(), user1.ID, "Key B", "hashB", "prefixB"); err != nil {
		failNowf(t, "CreateAPIKey B: %v", err)
	}
	if _, err := d.CreateAPIKey(t.Context(), u2.ID, "Key C", "hashC", "prefixC"); err != nil {
		failNowf(t, "CreateAPIKey C: %v", err)
	}

	keys, err := d.ListAPIKeys(t.Context(), user1.ID)
	if err != nil {
		failNowf(t, "ListAPIKeys() error: %v", err)
	}
	if len(keys) != 2 {
		failf(t, "len = %d, want 2", len(keys))
	}

	// user2 should only see their own key.
	keys2, err := d.ListAPIKeys(t.Context(), u2.ID)
	if err != nil {
		failNowf(t, "ListAPIKeys(user2) error: %v", err)
	}
	if len(keys2) != 1 {
		failf(t, "user2 len = %d, want 1", len(keys2))
	}
}

func TestGetAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Fetch Me", "hashFetch", "prefFetch")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != nil {
		failNowf(t, "GetAPIKey() error: %v", err)
	}
	if got.ID != created.ID {
		failf(t, "ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "Fetch Me" {
		failf(t, "Name = %q, want %q", got.Name, "Fetch Me")
	}
}

func TestGetAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other2@example.com", "password2")
	if err != nil {
		failNowf(t, "create user2: %v", err)
	}

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Private", "hashPriv", "prefPriv")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	_, err = d.GetAPIKey(t.Context(), created.ID, u2.ID)
	if err != sql.ErrNoRows {
		failf(t, "err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Delete Me", "hashDel", "prefDel")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	if err := d.DeleteAPIKey(t.Context(), created.ID, user.ID); err != nil {
		failNowf(t, "DeleteAPIKey() error: %v", err)
	}

	// Should be gone.
	_, err = d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != sql.ErrNoRows {
		failf(t, "after delete: err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteAPIKey(t.Context(), "nonexistent", user.ID)
	if err != sql.ErrNoRows {
		failf(t, "err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other3@example.com", "password2")
	if err != nil {
		failNowf(t, "create user2: %v", err)
	}

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Protected", "hashProt", "prefProt")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	err = d.DeleteAPIKey(t.Context(), created.ID, u2.ID)
	if err != sql.ErrNoRows {
		failf(t, "err = %v, want sql.ErrNoRows", err)
	}

	// Key should still exist for original user.
	_, err = d.GetAPIKey(t.Context(), created.ID, user1.ID)
	if err != nil {
		failf(t, "key should still exist: %v", err)
	}
}

func TestValidateAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	fullKey := "bib_deadbeef12345678"
	h := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(h[:])

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Auth Key", keyHash, fullKey[:8])
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	userID, keyID, err := d.ValidateAPIKey(t.Context(), keyHash)
	if err != nil {
		failNowf(t, "ValidateAPIKey() error: %v", err)
	}
	if userID != user.ID {
		failf(t, "userID = %q, want %q", userID, user.ID)
	}
	if keyID != created.ID {
		failf(t, "keyID = %q, want %q", keyID, created.ID)
	}
}

func TestValidateAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, _, err := d.ValidateAPIKey(t.Context(), "nonexistenthash")
	if err != sql.ErrNoRows {
		failf(t, "err = %v, want sql.ErrNoRows", err)
	}
}

func TestTouchAPIKeyLastUsed(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Touch Me", "hashTouch", "prefTouch")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}
	if created.LastUsedAt != nil {
		failNow(t, "LastUsedAt should be nil initially")
	}

	if err := d.TouchAPIKeyLastUsed(t.Context(), created.ID); err != nil {
		failNowf(t, "TouchAPIKeyLastUsed() error: %v", err)
	}

	// Re-fetch and verify last_used_at is now set.
	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != nil {
		failNowf(t, "GetAPIKey() error: %v", err)
	}
	if got.LastUsedAt == nil {
		fail(t, "LastUsedAt should be non-nil after touch")
	}
}

func TestGetAPIKeyByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Hash Lookup", "uniqueHash42", "prefHash")
	if err != nil {
		failNowf(t, "CreateAPIKey() error: %v", err)
	}

	got, err := d.GetAPIKeyByHash(t.Context(), "uniqueHash42")
	if err != nil {
		failNowf(t, "GetAPIKeyByHash() error: %v", err)
	}
	if got.ID != created.ID {
		failf(t, "ID = %q, want %q", got.ID, created.ID)
	}
}

func TestGetAPIKeyByHash_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAPIKeyByHash(t.Context(), "nosuchhash")
	if err != sql.ErrNoRows {
		failf(t, "err = %v, want sql.ErrNoRows", err)
	}
}
