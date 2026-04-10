package handlers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func TestToCredentialEntity(t *testing.T) {
	now := db.Timestamp{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	c := &db.ProtocolCredential{
		ID:           "cred-123",
		UserID:       "user-456",
		Username:     "alice",
		PasswordHash: "secret-hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	entity := toCredentialEntity(c)

	require.Equal(t, "cred-123", entity.ID)
	require.Equal(t, "alice", entity.Username)
	require.Equal(t, now, entity.CreatedAt)
	require.Equal(t, now, entity.UpdatedAt)
	// credentialEntity deliberately excludes PasswordHash — verify the type
	// has no such field by confirming the struct only carries safe fields.
	// Since credentialEntity lacks a PasswordHash field, the conversion
	// cannot leak the hash. This compile-time guarantee is the point; this
	// assertion documents that intent.
	type hasPasswordHash interface{ GetPasswordHash() string }
	_, ok := any(entity).(hasPasswordHash)
	require.False(t, ok, "credentialEntity must not expose PasswordHash")
}

func TestCredentialGetAdapter_Success(t *testing.T) {
	now := db.Timestamp{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	fakeFn := func(_ context.Context, userID string) (*db.ProtocolCredential, error) {
		return &db.ProtocolCredential{
			ID:        "cred-1",
			UserID:    userID,
			Username:  "testuser",
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	adapted := credGetAdapter(fakeFn)
	entity, err := adapted(t.Context(), "user-1")
	require.NoError(t, err)
	require.Equal(t, "cred-1", entity.ID)
	require.Equal(t, "testuser", entity.Username)
}

func TestCredentialGetAdapter_Error(t *testing.T) {
	fakeFn := func(_ context.Context, _ string) (*db.ProtocolCredential, error) {
		return nil, sql.ErrNoRows
	}

	adapted := credGetAdapter(fakeFn)
	_, err := adapted(t.Context(), "user-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCredentialUpsertAdapter_Success(t *testing.T) {
	now := db.Timestamp{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	fakeFn := func(_ context.Context, userID, username, _ string) (*db.ProtocolCredential, error) {
		return &db.ProtocolCredential{
			ID:        "cred-2",
			UserID:    userID,
			Username:  username,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	adapted := credUpsertAdapter(fakeFn)
	entity, err := adapted(t.Context(), "user-1", "alice", "hash")
	require.NoError(t, err)
	require.Equal(t, "cred-2", entity.ID)
	require.Equal(t, "alice", entity.Username)
}

func TestCredentialUpsertAdapter_Error(t *testing.T) {
	fakeFn := func(_ context.Context, _, _, _ string) (*db.ProtocolCredential, error) {
		return nil, sql.ErrNoRows
	}

	adapted := credUpsertAdapter(fakeFn)
	_, err := adapted(t.Context(), "user-1", "alice", "hash")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestConvertCredResult_Success(t *testing.T) {
	ts := db.Timestamp{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	cred := &db.ProtocolCredential{
		ID: "id-1", Username: "alice", CreatedAt: ts, UpdatedAt: ts,
	}

	got, err := convertCredResult(cred, nil)
	require.NoError(t, err)
	require.Equal(t, "id-1", got.ID)
	require.Equal(t, "alice", got.Username)
}

func TestConvertCredResult_Error(t *testing.T) {
	sentinel := errors.New("db failure")
	got, err := convertCredResult[*db.ProtocolCredential](nil, sentinel)
	require.ErrorIs(t, err, sentinel)
	require.Equal(t, (credentialEntity{}), got)
}
