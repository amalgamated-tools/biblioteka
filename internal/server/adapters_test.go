package server

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/stretchr/testify/require"
)

// ── protocolCredDBAdapter ─────────────────────────────────────────────────────

func TestGetOPDSCredential_Found(t *testing.T) {
	d := newTestDB(t)
	a := &protocolCredDBAdapter{db: d}

	user, err := d.CreateUser(t.Context(), "OPDS User", "opds@example.com", "secret")
	require.NoError(t, err)
	_, err = d.UpsertOPDSCredential(t.Context(), user.ID, "opdsuser", "hashed_pw")
	require.NoError(t, err)

	result, err := a.GetOPDSCredential(t.Context(), "opdsuser")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, user.ID, result.UserID)
	require.Equal(t, "hashed_pw", result.PasswordHash)
}

func TestGetOPDSCredential_NotFound(t *testing.T) {
	d := newTestDB(t)
	a := &protocolCredDBAdapter{db: d}

	_, err := a.GetOPDSCredential(t.Context(), "nobody")
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows),
		"missing credential should wrap sql.ErrNoRows; got: %v", err)
}

func TestGetKOSyncCredential_Found(t *testing.T) {
	d := newTestDB(t)
	a := &protocolCredDBAdapter{db: d}

	user, err := d.CreateUser(t.Context(), "KOSync User", "kosync@example.com", "secret")
	require.NoError(t, err)
	_, err = d.UpsertKOSyncCredential(t.Context(), user.ID, "kosyncuser", "hashed_pw")
	require.NoError(t, err)

	result, err := a.GetKOSyncCredential(t.Context(), "kosyncuser")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, user.ID, result.UserID)
	require.Equal(t, "hashed_pw", result.PasswordHash)
}

func TestGetKOSyncCredential_NotFound(t *testing.T) {
	d := newTestDB(t)
	a := &protocolCredDBAdapter{db: d}

	_, err := a.GetKOSyncCredential(t.Context(), "nobody")
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows),
		"missing KOSync credential should wrap sql.ErrNoRows; got: %v", err)
}

// ── koboDBAdapter ─────────────────────────────────────────────────────────────

func TestGetKoboTokenByToken_Found(t *testing.T) {
	d := newTestDB(t)
	a := &koboDBAdapter{db: d}

	user, err := d.CreateUser(t.Context(), "Kobo User", "kobo@example.com", "secret")
	require.NoError(t, err)

	rawToken := "raw-kobo-token-12345"
	tokenHash := auth.HashKoboToken(rawToken)
	_, err = d.CreateKoboToken(t.Context(), user.ID, "My Kobo", tokenHash)
	require.NoError(t, err)

	result, err := a.GetKoboTokenByToken(t.Context(), rawToken)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, user.ID, result.UserID)
}

func TestGetKoboTokenByToken_NotFound(t *testing.T) {
	d := newTestDB(t)
	a := &koboDBAdapter{db: d}

	_, err := a.GetKoboTokenByToken(t.Context(), "nonexistent-token")
	require.Error(t, err)
	require.True(t, errors.Is(err, sql.ErrNoRows),
		"missing Kobo token should wrap sql.ErrNoRows; got: %v", err)
}

func TestGetKoboTokenByToken_HashesBeforeLookup(t *testing.T) {
	// Verify that the adapter hashes the raw token before querying the DB.
	// If the raw token itself were stored (not its hash) the lookup would fail.
	d := newTestDB(t)
	a := &koboDBAdapter{db: d}

	user, err := d.CreateUser(t.Context(), "Kobo Hash User", "kobohash@example.com", "secret")
	require.NoError(t, err)

	rawToken := "another-raw-token"
	tokenHash := auth.HashKoboToken(rawToken)
	_, err = d.CreateKoboToken(t.Context(), user.ID, "Device", tokenHash)
	require.NoError(t, err)

	// Looking up with the raw token must succeed (adapter hashes internally).
	result, err := a.GetKoboTokenByToken(t.Context(), rawToken)
	require.NoError(t, err)
	require.Equal(t, user.ID, result.UserID)

	// Looking up with the hash directly must fail (the DB stores the hash, not the
	// raw token, so passing the hash again would double-hash and not match).
	_, err = a.GetKoboTokenByToken(t.Context(), tokenHash)
	require.Error(t, err, "passing the hash directly should fail (adapter hashes again)")
}
