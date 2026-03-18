package db

import (
	"context"
	"database/sql"
	"testing"
)

func createTestUserForKOSync(t *testing.T, d *DB, email string) *User {
	t.Helper()
	user, err := d.CreateUser(context.Background(), "Test User", email, "hashedpw")
	if err != nil {
		t.Fatalf("CreateUser(%q): %v", email, err)
	}
	return user
}

// ---- KOSyncCredential tests ----

func TestKOSyncCredential_UpsertAndGet(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "alice@example.com")
	ctx := context.Background()

	cred, err := d.UpsertKOSyncCredential(ctx, user.ID, "alice", "hashval")
	if err != nil {
		t.Fatalf("UpsertKOSyncCredential: %v", err)
	}
	if cred.ID == "" {
		t.Error("cred.ID is empty")
	}
	if cred.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", cred.UserID, user.ID)
	}
	if cred.Username != "alice" {
		t.Errorf("Username = %q, want %q", cred.Username, "alice")
	}
	if cred.PasswordHash != "hashval" {
		t.Errorf("PasswordHash = %q, want %q", cred.PasswordHash, "hashval")
	}

	// Fetch by userID
	fetched, err := d.GetKOSyncCredentialByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetKOSyncCredentialByUserID: %v", err)
	}
	if fetched.Username != "alice" {
		t.Errorf("fetched Username = %q, want %q", fetched.Username, "alice")
	}

	// Fetch by username (lowercase, as the middleware always lowercases before calling)
	fetched2, err := d.GetKOSyncCredentialByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetKOSyncCredentialByUsername: %v", err)
	}
	if fetched2.UserID != user.ID {
		t.Errorf("fetched2 UserID = %q, want %q", fetched2.UserID, user.ID)
	}
}

func TestKOSyncCredential_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "bob@example.com")
	ctx := context.Background()

	_, err := d.UpsertKOSyncCredential(ctx, user.ID, "bob", "hash1")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	updated, err := d.UpsertKOSyncCredential(ctx, user.ID, "bob2", "hash2")
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if updated.Username != "bob2" {
		t.Errorf("updated Username = %q, want %q", updated.Username, "bob2")
	}
	if updated.PasswordHash != "hash2" {
		t.Errorf("updated PasswordHash = %q, want %q", updated.PasswordHash, "hash2")
	}
}

func TestKOSyncCredential_UsernameConflict(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForKOSync(t, d, "user1@example.com")
	user2 := createTestUserForKOSync(t, d, "user2@example.com")
	ctx := context.Background()

	_, err := d.UpsertKOSyncCredential(ctx, user1.ID, "shared", "hash1")
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	_, err = d.UpsertKOSyncCredential(ctx, user2.ID, "shared", "hash2")
	if err != ErrKOSyncUsernameExists {
		t.Errorf("expected ErrKOSyncUsernameExists, got %v", err)
	}
}

func TestKOSyncCredential_GetByUserID_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	_, err := d.GetKOSyncCredentialByUserID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestKOSyncCredential_GetByUsername_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	_, err := d.GetKOSyncCredentialByUsername(ctx, "nobody")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestKOSyncCredential_Delete(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "del@example.com")
	ctx := context.Background()

	_, err := d.UpsertKOSyncCredential(ctx, user.ID, "delme", "hash")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := d.DeleteKOSyncCredential(ctx, user.ID); err != nil {
		t.Fatalf("DeleteKOSyncCredential: %v", err)
	}

	_, err = d.GetKOSyncCredentialByUserID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestKOSyncCredential_Delete_NotFound(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	err := d.DeleteKOSyncCredential(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

// ---- ReadingProgress tests ----

func TestReadingProgress_UpsertAndGet(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "reader@example.com")
	ctx := context.Background()

	device := "MyKindle"
	deviceID := "device-001"
	p, err := d.UpsertReadingProgress(ctx, user.ID, "doc123", "/body/p[1]", 0.25, &device, &deviceID)
	if err != nil {
		t.Fatalf("UpsertReadingProgress: %v", err)
	}
	if p.ID == "" {
		t.Error("p.ID is empty")
	}
	if p.Document != "doc123" {
		t.Errorf("Document = %q, want %q", p.Document, "doc123")
	}
	if p.Progress != "/body/p[1]" {
		t.Errorf("Progress = %q, want %q", p.Progress, "/body/p[1]")
	}
	if p.Percentage != 0.25 {
		t.Errorf("Percentage = %v, want 0.25", p.Percentage)
	}
	if p.Device == nil || *p.Device != "MyKindle" {
		t.Errorf("Device = %v, want %q", p.Device, "MyKindle")
	}
	if p.DeviceID == nil || *p.DeviceID != "device-001" {
		t.Errorf("DeviceID = %v, want %q", p.DeviceID, "device-001")
	}

	// Fetch
	fetched, err := d.GetReadingProgress(ctx, user.ID, "doc123")
	if err != nil {
		t.Fatalf("GetReadingProgress: %v", err)
	}
	if fetched.Progress != "/body/p[1]" {
		t.Errorf("fetched Progress = %q, want %q", fetched.Progress, "/body/p[1]")
	}
}

func TestReadingProgress_Upsert_UpdatesExisting(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "reader2@example.com")
	ctx := context.Background()

	_, err := d.UpsertReadingProgress(ctx, user.ID, "doc456", "/body/p[1]", 0.1, nil, nil)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	updated, err := d.UpsertReadingProgress(ctx, user.ID, "doc456", "/body/p[10]", 0.5, nil, nil)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if updated.Progress != "/body/p[10]" {
		t.Errorf("updated Progress = %q, want %q", updated.Progress, "/body/p[10]")
	}
	if updated.Percentage != 0.5 {
		t.Errorf("updated Percentage = %v, want 0.5", updated.Percentage)
	}
}

func TestReadingProgress_GetNotFound(t *testing.T) {
	d := newTestDB(t)
	user := createTestUserForKOSync(t, d, "nobody@example.com")
	ctx := context.Background()

	_, err := d.GetReadingProgress(ctx, user.ID, "missing-doc")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestReadingProgress_IsolatedByUser(t *testing.T) {
	d := newTestDB(t)
	user1 := createTestUserForKOSync(t, d, "user1@books.com")
	user2 := createTestUserForKOSync(t, d, "user2@books.com")
	ctx := context.Background()

	_, err := d.UpsertReadingProgress(ctx, user1.ID, "shared-doc", "/body/p[5]", 0.3, nil, nil)
	if err != nil {
		t.Fatalf("upsert user1: %v", err)
	}

	// user2 should see no progress for the same document
	_, err = d.GetReadingProgress(ctx, user2.ID, "shared-doc")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows for user2, got %v", err)
	}
}
