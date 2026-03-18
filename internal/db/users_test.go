package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(context.Background(), "Alice", "alice@example.com", "hashedpw")
	if err != nil {
		failNowf(t, "CreateUser() error: %v", err)
	}
	if user.ID == "" {
		fail(t, "CreateUser() returned empty ID")
	}
	if user.Name != "Alice" {
		failf(t, "Name = %q, want %q", user.Name, "Alice")
	}
	if user.Email != "alice@example.com" {
		failf(t, "Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.PasswordHash != "hashedpw" {
		failf(t, "PasswordHash = %q, want %q", user.PasswordHash, "hashedpw")
	}
	if user.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
	}
	if !user.IsAdmin {
		fail(t, "first user should be admin")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateUser(context.Background(), "Alice", "alice@example.com", "pw1")
	if err != nil {
		failNowf(t, "first CreateUser() error: %v", err)
	}

	_, err = d.CreateUser(context.Background(), "Alice2", "alice@example.com", "pw2")
	if err != ErrEmailExists {
		failf(t, "expected ErrEmailExists, got %v", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateUser(context.Background(), "Bob", "bob@example.com", "pw")

	found, err := d.GetUserByEmail(context.Background(), "bob@example.com")
	if err != nil {
		failNowf(t, "GetUserByEmail() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByEmail(context.Background(), "nobody@example.com")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateUser(context.Background(), "Carol", "carol@example.com", "pw")

	found, err := d.GetUserByID(context.Background(), created.ID)
	if err != nil {
		failNowf(t, "GetUserByID() error: %v", err)
	}
	if found.Email != "carol@example.com" {
		failf(t, "Email = %q, want %q", found.Email, "carol@example.com")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByID(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateOIDCUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(context.Background(), "Dave", "dave@example.com", "oidc-sub-123")
	if err != nil {
		failNowf(t, "CreateOIDCUser() error: %v", err)
	}
	if user.OIDCSubject == nil || *user.OIDCSubject != "oidc-sub-123" {
		failf(t, "OIDCSubject = %v, want %q", user.OIDCSubject, "oidc-sub-123")
	}
	if user.PasswordHash != "" {
		failf(t, "PasswordHash should be empty for OIDC users, got %q", user.PasswordHash)
	}
}

func TestGetUserByOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateOIDCUser(context.Background(), "Eve", "eve@example.com", "oidc-sub-eve")

	found, err := d.GetUserByOIDCSubject(context.Background(), "oidc-sub-eve")
	if err != nil {
		failNowf(t, "GetUserByOIDCSubject() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "ID = %q, want %q", found.ID, created.ID)
	}
}

func TestLinkOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser(context.Background(), "Frank", "frank@example.com", "pw")

	if err := d.LinkOIDCSubject(context.Background(), user.ID, "oidc-sub-frank"); err != nil {
		failNowf(t, "LinkOIDCSubject() error: %v", err)
	}

	found, _ := d.GetUserByOIDCSubject(context.Background(), "oidc-sub-frank")
	if found.ID != user.ID {
		failf(t, "ID after linking = %q, want %q", found.ID, user.ID)
	}
}

func TestLinkOIDCSubject_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.LinkOIDCSubject(context.Background(), "nonexistent-id", "some-subject")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser(context.Background(), "Grace", "grace@example.com", "oldhash")

	if err := d.UpdatePassword(context.Background(), user.ID, "newhash"); err != nil {
		failNowf(t, "UpdatePassword() error: %v", err)
	}

	found, _ := d.GetUserByEmail(context.Background(), "grace@example.com")
	if found.PasswordHash != "newhash" {
		failf(t, "PasswordHash = %q, want %q", found.PasswordHash, "newhash")
	}
}

func TestUpdatePassword_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.UpdatePassword(context.Background(), "nonexistent-id", "hash")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateUser_SecondUserNotAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, _ := d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	if !u1.IsAdmin {
		fail(t, "first user should be admin")
	}
	if u2.IsAdmin {
		fail(t, "second user should not be admin")
	}
}

func TestCreateOIDCUser_FirstUserIsAdmin(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(context.Background(), "First", "first@example.com", "oidc-sub-1")
	if err != nil {
		failNowf(t, "CreateOIDCUser() error: %v", err)
	}
	if !user.IsAdmin {
		fail(t, "first OIDC user should be admin")
	}

	u2, _ := d.CreateOIDCUser(context.Background(), "Second", "second@example.com", "oidc-sub-2")
	if u2.IsAdmin {
		fail(t, "second OIDC user should not be admin")
	}
}

func TestIsAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, _ := d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	isAdmin, err := d.IsAdmin(context.Background(), u1.ID)
	if err != nil {
		failNowf(t, "IsAdmin() error: %v", err)
	}
	if !isAdmin {
		fail(t, "first user should be admin")
	}

	isAdmin2, err := d.IsAdmin(context.Background(), u2.ID)
	if err != nil {
		failNowf(t, "IsAdmin() for second user error: %v", err)
	}
	if isAdmin2 {
		fail(t, "second user should not be admin")
	}
}

func TestSetAdmin(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	// Promote second user
	if err := d.SetAdmin(context.Background(), u2.ID, true); err != nil {
		failNowf(t, "SetAdmin(true) error: %v", err)
	}

	isAdmin, _ := d.IsAdmin(context.Background(), u2.ID)
	if !isAdmin {
		fail(t, "second user should be admin after promotion")
	}

	// Demote second user
	if err := d.SetAdmin(context.Background(), u2.ID, false); err != nil {
		failNowf(t, "SetAdmin(false) error: %v", err)
	}

	isAdmin, _ = d.IsAdmin(context.Background(), u2.ID)
	if isAdmin {
		fail(t, "second user should not be admin after demotion")
	}
}

func TestSetAdmin_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.SetAdmin(context.Background(), "nonexistent-id", true)
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateUser(context.Background(), "Alice", "alice@example.com", "pw")
	_, _ = d.CreateUser(context.Background(), "Bob", "bob@example.com", "pw")
	_, _ = d.CreateUser(context.Background(), "Carol", "carol@example.com", "pw")

	users, err := d.ListUsers(context.Background())
	if err != nil {
		failNowf(t, "ListUsers() error: %v", err)
	}
	if len(users) != 3 {
		failNowf(t, "ListUsers() returned %d users, want 3", len(users))
	}
	if users[0].Name != "Alice" {
		failf(t, "first user Name = %q, want %q", users[0].Name, "Alice")
	}
	if !users[0].IsAdmin {
		fail(t, "first user should be admin")
	}
	if users[1].IsAdmin {
		fail(t, "second user should not be admin")
	}
}
