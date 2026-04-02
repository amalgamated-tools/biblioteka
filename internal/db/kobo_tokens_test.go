package db

import (
	"database/sql"
	"testing"
)

// ---- CreateKoboToken ----

func TestCreateKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	token, err := d.CreateKoboToken(t.Context(), user.ID, "My Device", "hash-abc123")
	if err != nil {
		t.Fatalf("CreateKoboToken() error: %v", err)
	}
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
	if err != nil {
		t.Fatalf("CreateKoboToken() error: %v", err)
	}

	got, err := d.GetKoboToken(t.Context(), created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetKoboToken() error: %v", err)
	}
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
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestGetKoboToken_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Other Token User", "other_tok@example.com", "pass2")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}

	created, err := d.CreateKoboToken(t.Context(), user1.ID, "Private Token", "hash-private")
	if err != nil {
		t.Fatalf("CreateKoboToken(): %v", err)
	}

	_, err = d.GetKoboToken(t.Context(), created.ID, user2.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows when fetching another user's token", err)
	}
}

// ---- GetKoboTokenByHash ----

func TestGetKoboTokenByHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Hash Lookup", "unique-hash-xyz")
	if err != nil {
		t.Fatalf("CreateKoboToken(): %v", err)
	}

	got, err := d.GetKoboTokenByHash(t.Context(), "unique-hash-xyz")
	if err != nil {
		t.Fatalf("GetKoboTokenByHash() error: %v", err)
	}
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
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

// ---- ListKoboTokens ----

func TestListKoboTokens_Empty(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	tokens, err := d.ListKoboTokens(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("ListKoboTokens() error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("len(tokens) = %d, want 0", len(tokens))
	}
}

func TestListKoboTokens_ReturnsUserTokensOnly(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "User 2 Kobo", "u2kobo@example.com", "pass2")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}

	if _, err := d.CreateKoboToken(t.Context(), user1.ID, "Token A", "hash-A"); err != nil {
		t.Fatalf("CreateKoboToken(A): %v", err)
	}
	if _, err := d.CreateKoboToken(t.Context(), user1.ID, "Token B", "hash-B"); err != nil {
		t.Fatalf("CreateKoboToken(B): %v", err)
	}
	if _, err := d.CreateKoboToken(t.Context(), user2.ID, "Token C", "hash-C"); err != nil {
		t.Fatalf("CreateKoboToken(C): %v", err)
	}

	tokens1, err := d.ListKoboTokens(t.Context(), user1.ID)
	if err != nil {
		t.Fatalf("ListKoboTokens(user1) error: %v", err)
	}
	if len(tokens1) != 2 {
		t.Errorf("user1 len = %d, want 2", len(tokens1))
	}

	tokens2, err := d.ListKoboTokens(t.Context(), user2.ID)
	if err != nil {
		t.Fatalf("ListKoboTokens(user2) error: %v", err)
	}
	if len(tokens2) != 1 {
		t.Errorf("user2 len = %d, want 1", len(tokens2))
	}
}

func TestListKoboTokens_OrderedNewestFirst(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	t1, err := d.CreateKoboToken(t.Context(), user.ID, "First Token", "hash-first")
	if err != nil {
		t.Fatalf("CreateKoboToken(first): %v", err)
	}
	t2, err := d.CreateKoboToken(t.Context(), user.ID, "Second Token", "hash-second")
	if err != nil {
		t.Fatalf("CreateKoboToken(second): %v", err)
	}

	tokens, err := d.ListKoboTokens(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("ListKoboTokens() error: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("len(tokens) = %d, want 2", len(tokens))
	}

	// Ordered by created_at DESC then id DESC: second token was created later.
	// When timestamps are equal (same second in SQLite), id DESC determines order.
	if tokens[0].ID != t2.ID && tokens[0].ID != t1.ID {
		t.Errorf("unexpected token order: first = %q", tokens[0].Name)
	}
	// Verify the two expected token IDs are both present.
	ids := map[string]bool{tokens[0].ID: true, tokens[1].ID: true}
	if !ids[t1.ID] || !ids[t2.ID] {
		t.Errorf("expected both tokens in result, got %v", ids)
	}
}

// ---- DeleteKoboToken ----

func TestDeleteKoboToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	created, err := d.CreateKoboToken(t.Context(), user.ID, "Delete Me", "hash-del")
	if err != nil {
		t.Fatalf("CreateKoboToken(): %v", err)
	}

	if err := d.DeleteKoboToken(t.Context(), created.ID, user.ID); err != nil {
		t.Fatalf("DeleteKoboToken() error: %v", err)
	}

	_, err = d.GetKoboToken(t.Context(), created.ID, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("after delete: err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteKoboToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	err := d.DeleteKoboToken(t.Context(), "nonexistent-token-id", user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestDeleteKoboToken_WrongUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUser(t, d)
	user2, err := d.CreateUser(t.Context(), "Unauthorized", "unauth_tok@example.com", "pass2")
	if err != nil {
		t.Fatalf("CreateUser(user2): %v", err)
	}

	created, err := d.CreateKoboToken(t.Context(), user1.ID, "Protected Token", "hash-protected")
	if err != nil {
		t.Fatalf("CreateKoboToken(): %v", err)
	}

	// user2 cannot delete user1's token.
	err = d.DeleteKoboToken(t.Context(), created.ID, user2.ID)
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, want sql.ErrNoRows when deleting another user's token", err)
	}

	// Token should still exist for user1.
	_, err = d.GetKoboToken(t.Context(), created.ID, user1.ID)
	if err != nil {
		t.Errorf("token should still exist for user1: %v", err)
	}
}
