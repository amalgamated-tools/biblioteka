package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---- CreateKoboToken ----

func TestCreateKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	token, err := d.CreateKoboToken(t.Context(), user.ID, "My Device", "hash-abc123")
	require.NoError(t, err, "CreateKoboToken() error")
	if token.ID == "" {
		t.Error("token.ID is empty")
	}
	if token.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", token.UserID, user.ID)
	}
	if token.Name != "My Device" {
		t.Errorf("Name = %q, want My Device", token.Name)
	}
	if token.TokenHash != "hash-abc123" {
		t.Errorf("TokenHash = %q, want hash-abc123", token.TokenHash)
	}
	if token.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

// ---- GetKoboToken ----

func TestGetKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Tablet", "hash-tablet")
	require.NoError(t, err, "CreateKoboToken() error")

	got, err := d.GetKoboToken(t.Context(), created.ID, user.ID)
	require.NoError(t, err, "GetKoboToken() error")
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Name != "Tablet" {
		t.Errorf("Name = %q, want Tablet", got.Name)
	}
	if got.TokenHash != "hash-tablet" {
		t.Errorf("TokenHash = %q, want hash-tablet", got.TokenHash)
	}
}

func TestGetKoboToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	_, err := d.GetKoboToken(t.Context(), "nonexistent-id", user.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestGetKoboToken_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Other Token User", "other_tok@example.com", "pass2")
	require.NoError(t, err, "CreateUser(user2)")

	created, err := d.CreateKoboToken(t.Context(), user1.ID, "Private Token", "hash-private")
	require.NoError(t, err, "CreateKoboToken()")

	_, err = d.GetKoboToken(t.Context(), created.ID, user2.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows when fetching another user's token", err)
	}
}

// ---- GetKoboTokenByHash ----

func TestGetKoboTokenByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Hash Lookup", "unique-hash-xyz")
	require.NoError(t, err, "CreateKoboToken()")

	got, err := d.GetKoboTokenByHash(t.Context(), "unique-hash-xyz")
	require.NoError(t, err, "GetKoboTokenByHash() error")
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.TokenHash != "unique-hash-xyz" {
		t.Errorf("TokenHash = %q, want unique-hash-xyz", got.TokenHash)
	}
}

func TestGetKoboTokenByHash_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetKoboTokenByHash(t.Context(), "no-such-hash")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

// ---- ListKoboTokens ----

func TestListKoboTokens_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	tokens, err := d.ListKoboTokens(t.Context(), user.ID)
	require.NoError(t, err, "ListKoboTokens() error")
	if len(tokens) != 0 {
		t.Errorf("len(tokens) = %d, want 0", len(tokens))
	}
}

func TestListKoboTokens_ReturnsUserTokensOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "User 2 Kobo", "u2kobo@example.com", "pass2")
	require.NoError(t, err, "CreateUser(user2)")

	if _, err := d.CreateKoboToken(t.Context(), user1.ID, "Token A", "hash-A"); err != nil {
		require.NoError(t, err, "CreateKoboToken(A)")
	}
	if _, err := d.CreateKoboToken(t.Context(), user1.ID, "Token B", "hash-B"); err != nil {
		require.NoError(t, err, "CreateKoboToken(B)")
	}
	if _, err := d.CreateKoboToken(t.Context(), user2.ID, "Token C", "hash-C"); err != nil {
		require.NoError(t, err, "CreateKoboToken(C)")
	}

	tokens1, err := d.ListKoboTokens(t.Context(), user1.ID)
	require.NoError(t, err, "ListKoboTokens(user1) error")
	if len(tokens1) != 2 {
		t.Errorf("user1 len = %d, want 2", len(tokens1))
	}

	tokens2, err := d.ListKoboTokens(t.Context(), user2.ID)
	require.NoError(t, err, "ListKoboTokens(user2) error")
	if len(tokens2) != 1 {
		t.Errorf("user2 len = %d, want 1", len(tokens2))
	}
}

func TestListKoboTokens_OrderedNewestFirst(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	t1, err := d.CreateKoboToken(t.Context(), user.ID, "First Token", "hash-first")
	require.NoError(t, err, "CreateKoboToken(first)")
	// Sleep to guarantee distinct created_at timestamps in SQLite (second precision).
	time.Sleep(1100 * time.Millisecond)
	t2, err := d.CreateKoboToken(t.Context(), user.ID, "Second Token", "hash-second")
	require.NoError(t, err, "CreateKoboToken(second)")

	tokens, err := d.ListKoboTokens(t.Context(), user.ID)
	require.NoError(t, err, "ListKoboTokens() error")
	if len(tokens) != 2 {
		require.Failf(t, "failed", "len(tokens) = %d, want 2", len(tokens))
	}

	// Ordered by created_at DESC then id DESC: second token should come first.
	if tokens[0].ID != t2.ID {
		t.Errorf("tokens[0].ID = %q, want %q (newest first)", tokens[0].ID, t2.ID)
	}
	if tokens[1].ID != t1.ID {
		t.Errorf("tokens[1].ID = %q, want %q (oldest second)", tokens[1].ID, t1.ID)
	}
}

// ---- DeleteKoboToken ----

func TestDeleteKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Delete Me", "hash-del")
	require.NoError(t, err, "CreateKoboToken()")

	require.NoError(t, d.DeleteKoboToken(t.Context(), created.ID, user.ID), "DeleteKoboToken() error")

	_, err = d.GetKoboToken(t.Context(), created.ID, user.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("after delete: err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteKoboToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteKoboToken(t.Context(), "nonexistent-token-id", user.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteKoboToken_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Unauthorized", "unauth_tok@example.com", "pass2")
	require.NoError(t, err, "CreateUser(user2)")

	created, err := d.CreateKoboToken(t.Context(), user1.ID, "Protected Token", "hash-protected")
	require.NoError(t, err, "CreateKoboToken()")

	// user2 cannot delete user1's token.
	err = d.DeleteKoboToken(t.Context(), created.ID, user2.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows when deleting another user's token", err)
	}

	// Token should still exist for user1.
	_, err = d.GetKoboToken(t.Context(), created.ID, user1.ID)
	if err != nil {
		t.Errorf("token should still exist for user1: %v", err)
	}
}
