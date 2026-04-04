package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"
)

func createTestUser(t *testing.T, d *DB) *User {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", t.Name()+"@example.com", "password1")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestCreateAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	key, err := d.CreateAPIKey(t.Context(), user.ID, "CI Pipeline", "hash123", "bib_abcd")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}
	if key.ID == "" {
		t.Error("expected non-empty ID")
	}
	if key.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", key.UserID, user.ID)
	}
	if key.Name != "CI Pipeline" {
		t.Errorf("Name = %q, want %q", key.Name, "CI Pipeline")
	}
	if key.KeyHash != "hash123" {
		t.Errorf("KeyHash = %q, want %q", key.KeyHash, "hash123")
	}
	if key.KeyPrefix != "bib_abcd" {
		t.Errorf("KeyPrefix = %q, want %q", key.KeyPrefix, "bib_abcd")
	}
	if key.LastUsedAt != nil {
		t.Errorf("LastUsedAt = %v, want nil", key.LastUsedAt)
	}
	if key.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestListAPIKeys_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	keys, err := d.ListAPIKeys(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys() error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("len = %d, want 0", len(keys))
	}
}

func TestListAPIKeys_ReturnsUserKeysOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other@example.com", "password2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	// Create keys for both users.
	if _, err := d.CreateAPIKey(t.Context(), user1.ID, "Key A", "hashA", "prefixA"); err != nil {
		t.Fatalf("CreateAPIKey A: %v", err)
	}
	if _, err := d.CreateAPIKey(t.Context(), user1.ID, "Key B", "hashB", "prefixB"); err != nil {
		t.Fatalf("CreateAPIKey B: %v", err)
	}
	if _, err := d.CreateAPIKey(t.Context(), u2.ID, "Key C", "hashC", "prefixC"); err != nil {
		t.Fatalf("CreateAPIKey C: %v", err)
	}

	keys, err := d.ListAPIKeys(t.Context(), user1.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys() error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("len = %d, want 2", len(keys))
	}

	// user2 should only see their own key.
	keys2, err := d.ListAPIKeys(t.Context(), u2.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys(user2) error: %v", err)
	}
	if len(keys2) != 1 {
		t.Errorf("user2 len = %d, want 1", len(keys2))
	}
}

func TestGetAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Fetch Me", "hashFetch", "prefFetch")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetAPIKey() error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "Fetch Me" {
		t.Errorf("Name = %q, want %q", got.Name, "Fetch Me")
	}
}

func TestGetAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other2@example.com", "password2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Private", "hashPriv", "prefPriv")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	_, err = d.GetAPIKey(t.Context(), created.ID, u2.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Delete Me", "hashDel", "prefDel")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	if err := d.DeleteAPIKey(t.Context(), created.ID, user.ID); err != nil {
		t.Fatalf("DeleteAPIKey() error: %v", err)
	}

	// Should be gone.
	_, err = d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("after delete: err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteAPIKey(t.Context(), "nonexistent", user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteAPIKey_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)

	u2, err := d.CreateUser(t.Context(), "Other", "other3@example.com", "password2")
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}

	created, err := d.CreateAPIKey(t.Context(), user1.ID, "Protected", "hashProt", "prefProt")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	err = d.DeleteAPIKey(t.Context(), created.ID, u2.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}

	// Key should still exist for original user.
	_, err = d.GetAPIKey(t.Context(), created.ID, user1.ID)
	if err != nil {
		t.Errorf("key should still exist: %v", err)
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
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	userID, keyID, err := d.ValidateAPIKey(t.Context(), keyHash)
	if err != nil {
		t.Fatalf("ValidateAPIKey() error: %v", err)
	}
	if userID != user.ID {
		t.Errorf("userID = %q, want %q", userID, user.ID)
	}
	if keyID != created.ID {
		t.Errorf("keyID = %q, want %q", keyID, created.ID)
	}
}

func TestValidateAPIKey_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, _, err := d.ValidateAPIKey(t.Context(), "nonexistenthash")
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestTouchAPIKeyLastUsed(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Touch Me", "hashTouch", "prefTouch")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}
	if created.LastUsedAt != nil {
		t.Fatal("LastUsedAt should be nil initially")
	}

	if err := d.TouchAPIKeyLastUsed(t.Context(), created.ID); err != nil {
		t.Fatalf("TouchAPIKeyLastUsed() error: %v", err)
	}

	// Re-fetch and verify last_used_at is now set.
	got, err := d.GetAPIKey(t.Context(), created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetAPIKey() error: %v", err)
	}
	if got.LastUsedAt == nil {
		t.Error("LastUsedAt should be non-nil after touch")
	}
}

func TestGetAPIKeyByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateAPIKey(t.Context(), user.ID, "Hash Lookup", "uniqueHash42", "prefHash")
	if err != nil {
		t.Fatalf("CreateAPIKey() error: %v", err)
	}

	got, err := d.GetAPIKeyByHash(t.Context(), "uniqueHash42")
	if err != nil {
		t.Fatalf("GetAPIKeyByHash() error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestGetAPIKeyByHash_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAPIKeyByHash(t.Context(), "nosuchhash")
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}
