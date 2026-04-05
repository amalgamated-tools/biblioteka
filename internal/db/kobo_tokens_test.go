package db

import (
	"database/sql"
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
	require.NotEqual(t, "", token.ID)
	require.Equal(t, user.ID, token.UserID)
	require.Equal(t, "My Device", token.Name)
	require.Equal(t, "hash-abc123", token.TokenHash)
	require.False(t, token.CreatedAt.IsZero())
}

// ---- GetKoboToken ----

func TestGetKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Tablet", "hash-tablet")
	require.NoError(t, err, "CreateKoboToken() error")

	got, err := d.GetKoboToken(t.Context(), created.ID, user.ID)
	require.NoError(t, err, "GetKoboToken() error")
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "Tablet", got.Name)
	require.Equal(t, "hash-tablet", got.TokenHash)
}

func TestGetKoboToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	_, err := d.GetKoboToken(t.Context(), "nonexistent-id", user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetKoboToken_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Other Token User", "other_tok@example.com", "pass2")
	require.NoError(t, err, "CreateUser(user2)")

	created, err := d.CreateKoboToken(t.Context(), user1.ID, "Private Token", "hash-private")
	require.NoError(t, err, "CreateKoboToken()")

	_, err = d.GetKoboToken(t.Context(), created.ID, user2.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// ---- GetKoboTokenByHash ----

func TestGetKoboTokenByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Hash Lookup", "unique-hash-xyz")
	require.NoError(t, err, "CreateKoboToken()")

	got, err := d.GetKoboTokenByHash(t.Context(), "unique-hash-xyz")
	require.NoError(t, err, "GetKoboTokenByHash() error")
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "unique-hash-xyz", got.TokenHash)
}

func TestGetKoboTokenByHash_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetKoboTokenByHash(t.Context(), "no-such-hash")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// ---- ListKoboTokens ----

func TestListKoboTokens_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	tokens, err := d.ListKoboTokens(t.Context(), user.ID)
	require.NoError(t, err, "ListKoboTokens() error")
	require.Len(t, tokens, 0)
}

func TestListKoboTokens_ReturnsUserTokensOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "User 2 Kobo", "u2kobo@example.com", "pass2")
	require.NoError(t, err, "CreateUser(user2)")

	_, err = d.CreateKoboToken(t.Context(), user1.ID, "Token A", "hash-A")
	require.NoError(t, err, "CreateKoboToken(A)")
	_, err = d.CreateKoboToken(t.Context(), user1.ID, "Token B", "hash-B")
	require.NoError(t, err, "CreateKoboToken(B)")
	_, err = d.CreateKoboToken(t.Context(), user2.ID, "Token C", "hash-C")
	require.NoError(t, err, "CreateKoboToken(C)")

	tokens1, err := d.ListKoboTokens(t.Context(), user1.ID)
	require.NoError(t, err, "ListKoboTokens(user1) error")
	require.Len(t, tokens1, 2)

	tokens2, err := d.ListKoboTokens(t.Context(), user2.ID)
	require.NoError(t, err, "ListKoboTokens(user2) error")
	require.Len(t, tokens2, 1)
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
	require.Len(t, tokens, 2)

	// Ordered by created_at DESC then id DESC: second token should come first.
	require.Equal(t, t2.ID, tokens[0].ID)
	require.Equal(t, t1.ID, tokens[1].ID)
}

// ---- DeleteKoboToken ----

func TestDeleteKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Delete Me", "hash-del")
	require.NoError(t, err, "CreateKoboToken()")

	require.NoError(t, d.DeleteKoboToken(t.Context(), created.ID, user.ID), "DeleteKoboToken() error")

	_, err = d.GetKoboToken(t.Context(), created.ID, user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteKoboToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteKoboToken(t.Context(), "nonexistent-token-id", user.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
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
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Token should still exist for user1.
	_, err = d.GetKoboToken(t.Context(), created.ID, user1.ID)
	require.NoError(t, err)
}
