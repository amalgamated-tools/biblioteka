package db

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- hasColumn ----

func TestHasColumn_ExistingColumn(t *testing.T) {
	d := newTestDB(t)

	// The books table is guaranteed to exist after migrations.
	exists, err := hasColumn(t.Context(), d, "books", "title")
	require.NoError(t, err, "hasColumn(books, title) error")
	if !exists {
		t.Error("hasColumn(books, title) = false, want true")
	}
}

func TestHasColumn_MissingColumn(t *testing.T) {
	d := newTestDB(t)

	exists, err := hasColumn(t.Context(), d, "books", "nonexistent_column_xyz")
	require.NoError(t, err, "hasColumn(books, nonexistent_column_xyz) error")
	if exists {
		t.Error("hasColumn(books, nonexistent_column_xyz) = true, want false")
	}
}

func TestHasColumn_KoboTokensColumns(t *testing.T) {
	d := newTestDB(t)

	// Verify the columns that backfillKoboTokenHashes depends on are present.
	for _, col := range []string{"token", "token_hash"} {
		exists, err := hasColumn(t.Context(), d, "kobo_tokens", col)
		require.NoError(t, err, "hasColumn(kobo_tokens, %q) error", col)
		if !exists {
			t.Errorf("hasColumn(kobo_tokens, %q) = false, want true", col)
		}
	}
}

// ---- backfillKoboTokenHashes ----

// expectedHash returns the SHA-256 hex digest that backfillKoboTokenHashes
// would produce for a given token value.
func expectedHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func TestBackfillKoboTokenHashes_NothingToBackfill(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	// CreateKoboToken always sets token_hash, so nothing needs backfilling.
	if _, err := d.CreateKoboToken(t.Context(), user.ID, "Already Hashed", "pre-hashed-value"); err != nil {
		require.NoError(t, err, "CreateKoboToken()")
	}

	// backfillKoboTokenHashes should be a no-op.
	require.NoError(t, backfillKoboTokenHashes(t.Context(), d), "backfillKoboTokenHashes() error")

	got, err := d.GetKoboTokenByHash(t.Context(), "pre-hashed-value")
	require.NoError(t, err, "GetKoboTokenByHash() after backfill error")
	if got.TokenHash != "pre-hashed-value" {
		t.Errorf("TokenHash = %q, want pre-hashed-value (unchanged)", got.TokenHash)
	}
}

func TestBackfillKoboTokenHashes_BackfillsNullHash(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	// Manually insert a row that has a token but an empty token_hash to
	// simulate the pre-migration state that backfill is designed to handle.
	rawToken := "raw-unhashed-token-value"
	var tokenID string
	err := d.QueryRowContext(t.Context(),
		`INSERT INTO kobo_tokens (user_id, name, token, token_hash) VALUES ($1, $2, $3, '') RETURNING id`,
		user.ID, "Legacy Token", rawToken,
	).Scan(&tokenID)
	require.NoError(t, err, "manual insert legacy token")

	require.NoError(t, backfillKoboTokenHashes(t.Context(), d), "backfillKoboTokenHashes() error")

	// Verify the token can now be found by its computed hash.
	hash := expectedHash(rawToken)
	got, err := d.GetKoboTokenByHash(t.Context(), hash)
	require.NoError(t, err, "GetKoboTokenByHash() after backfill error")
	if got.ID != tokenID {
		t.Errorf("ID = %q, want %q", got.ID, tokenID)
	}
}

func TestBackfillKoboTokenHashes_SkipsEmptyToken(t *testing.T) {
	d := newTestDB(t)
	user := createTestUser(t, d)

	// Insert a row with both token and token_hash empty; backfill should skip it.
	var tokenID string
	err := d.QueryRowContext(t.Context(),
		`INSERT INTO kobo_tokens (user_id, name, token, token_hash) VALUES ($1, $2, '', '') RETURNING id`,
		user.ID, "Empty Token",
	).Scan(&tokenID)
	require.NoError(t, err, "manual insert empty-token row")

	require.NoError(t, backfillKoboTokenHashes(t.Context(), d), "backfillKoboTokenHashes() error")

	// The row should still have an empty token_hash since the token was empty.
	var gotHash string
	if err := d.QueryRowContext(t.Context(),
		`SELECT token_hash FROM kobo_tokens WHERE id = $1`, tokenID,
	).Scan(&gotHash); err != nil {
		require.NoError(t, err, "fetch token_hash")
	}
	if gotHash != "" {
		t.Errorf("token_hash = %q, want empty (skip empty-token rows)", gotHash)
	}
}

// ---- hashKoboToken ----

func TestHashKoboToken(t *testing.T) {
	token := "test-token-string"
	got := hashKoboToken(token)
	want := expectedHash(token)
	if got != want {
		t.Errorf("hashKoboToken(%q) = %q, want %q", token, got, want)
	}
}

func TestHashKoboToken_DifferentInputs(t *testing.T) {
	h1 := hashKoboToken("token-one")
	h2 := hashKoboToken("token-two")
	if h1 == h2 {
		t.Error("different tokens produced the same hash")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 (SHA-256 hex)", len(h1))
	}
}
