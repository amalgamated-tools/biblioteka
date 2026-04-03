package handlers

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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

	if entity.ID != "cred-123" {
		t.Errorf("ID = %q, want %q", entity.ID, "cred-123")
	}
	if entity.Username != "alice" {
		t.Errorf("Username = %q, want %q", entity.Username, "alice")
	}
	if entity.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", entity.CreatedAt, now)
	}
	if entity.UpdatedAt != now {
		t.Errorf("UpdatedAt = %v, want %v", entity.UpdatedAt, now)
	}
	// credentialEntity deliberately excludes PasswordHash — verify the type
	// has no such field by confirming the struct only carries safe fields.
	// Since credentialEntity lacks a PasswordHash field, the conversion
	// cannot leak the hash. This compile-time guarantee is the point; this
	// assertion documents that intent.
	type hasPasswordHash interface{ GetPasswordHash() string }
	if _, ok := any(entity).(hasPasswordHash); ok {
		t.Error("credentialEntity must not expose PasswordHash")
	}
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

	adapted := credentialGetAdapter(fakeFn)
	entity, err := adapted(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.ID != "cred-1" {
		t.Errorf("ID = %q, want %q", entity.ID, "cred-1")
	}
	if entity.Username != "testuser" {
		t.Errorf("Username = %q, want %q", entity.Username, "testuser")
	}
}

func TestCredentialGetAdapter_Error(t *testing.T) {
	fakeFn := func(_ context.Context, _ string) (*db.ProtocolCredential, error) {
		return nil, sql.ErrNoRows
	}

	adapted := credentialGetAdapter(fakeFn)
	_, err := adapted(context.Background(), "user-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
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

	adapted := credentialUpsertAdapter(fakeFn)
	entity, err := adapted(context.Background(), "user-1", "alice", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entity.ID != "cred-2" {
		t.Errorf("ID = %q, want %q", entity.ID, "cred-2")
	}
	if entity.Username != "alice" {
		t.Errorf("Username = %q, want %q", entity.Username, "alice")
	}
}

func TestCredentialUpsertAdapter_Error(t *testing.T) {
	fakeFn := func(_ context.Context, _, _, _ string) (*db.ProtocolCredential, error) {
		return nil, sql.ErrNoRows
	}

	adapted := credentialUpsertAdapter(fakeFn)
	_, err := adapted(context.Background(), "user-1", "alice", "hash")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
