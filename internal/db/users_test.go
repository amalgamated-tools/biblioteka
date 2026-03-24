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
		t.Fatalf("CreateUser() error: %v", err)
	}
	if user.ID == "" {
		t.Error("CreateUser() returned empty ID")
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want %q", user.Name, "Alice")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.PasswordHash != "hashedpw" {
		t.Errorf("PasswordHash = %q, want %q", user.PasswordHash, "hashedpw")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if !user.IsAdmin {
		t.Error("first user should be admin")
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateUser(context.Background(), "Alice", "alice@example.com", "pw1")
	if err != nil {
		t.Fatalf("first CreateUser() error: %v", err)
	}

	_, err = d.CreateUser(context.Background(), "Alice2", "alice@example.com", "pw2")
	if err != ErrEmailExists {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateUser(context.Background(), "Bob", "bob@example.com", "pw")

	found, err := d.GetUserByEmail(context.Background(), "bob@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByEmail(context.Background(), "nobody@example.com")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateUser(context.Background(), "Carol", "carol@example.com", "pw")

	found, err := d.GetUserByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error: %v", err)
	}
	if found.Email != "carol@example.com" {
		t.Errorf("Email = %q, want %q", found.Email, "carol@example.com")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByID(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateOIDCUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(context.Background(), "Dave", "dave@example.com", "oidc-sub-123")
	if err != nil {
		t.Fatalf("CreateOIDCUser() error: %v", err)
	}
	if user.OIDCSubject == nil || *user.OIDCSubject != "oidc-sub-123" {
		t.Errorf("OIDCSubject = %v, want %q", user.OIDCSubject, "oidc-sub-123")
	}
	if user.PasswordHash != "" {
		t.Errorf("PasswordHash should be empty for OIDC users, got %q", user.PasswordHash)
	}
}

func TestGetUserByOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateOIDCUser(context.Background(), "Eve", "eve@example.com", "oidc-sub-eve")

	found, err := d.GetUserByOIDCSubject(context.Background(), "oidc-sub-eve")
	if err != nil {
		t.Fatalf("GetUserByOIDCSubject() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestLinkOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser(context.Background(), "Frank", "frank@example.com", "pw")

	if err := d.LinkOIDCSubject(context.Background(), user.ID, "oidc-sub-frank"); err != nil {
		t.Fatalf("LinkOIDCSubject() error: %v", err)
	}

	found, _ := d.GetUserByOIDCSubject(context.Background(), "oidc-sub-frank")
	if found.ID != user.ID {
		t.Errorf("ID after linking = %q, want %q", found.ID, user.ID)
	}
}

func TestLinkOIDCSubject_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.LinkOIDCSubject(context.Background(), "nonexistent-id", "some-subject")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser(context.Background(), "Grace", "grace@example.com", "oldhash")

	if err := d.UpdatePassword(context.Background(), user.ID, "newhash"); err != nil {
		t.Fatalf("UpdatePassword() error: %v", err)
	}

	found, _ := d.GetUserByEmail(context.Background(), "grace@example.com")
	if found.PasswordHash != "newhash" {
		t.Errorf("PasswordHash = %q, want %q", found.PasswordHash, "newhash")
	}
}

func TestUpdatePassword_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.UpdatePassword(context.Background(), "nonexistent-id", "hash")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateUser_SecondUserNotAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, _ := d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	if !u1.IsAdmin {
		t.Error("first user should be admin")
	}
	if u2.IsAdmin {
		t.Error("second user should not be admin")
	}
}

func TestCreateOIDCUser_FirstUserIsAdmin(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(context.Background(), "First", "first@example.com", "oidc-sub-1")
	if err != nil {
		t.Fatalf("CreateOIDCUser() error: %v", err)
	}
	if !user.IsAdmin {
		t.Error("first OIDC user should be admin")
	}

	u2, _ := d.CreateOIDCUser(context.Background(), "Second", "second@example.com", "oidc-sub-2")
	if u2.IsAdmin {
		t.Error("second OIDC user should not be admin")
	}
}

func TestIsAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, _ := d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	isAdmin, err := d.IsAdmin(context.Background(), u1.ID)
	if err != nil {
		t.Fatalf("IsAdmin() error: %v", err)
	}
	if !isAdmin {
		t.Error("first user should be admin")
	}

	isAdmin2, err := d.IsAdmin(context.Background(), u2.ID)
	if err != nil {
		t.Fatalf("IsAdmin() for second user error: %v", err)
	}
	if isAdmin2 {
		t.Error("second user should not be admin")
	}
}

func TestSetAdmin(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateUser(context.Background(), "First", "first@example.com", "pw")
	u2, _ := d.CreateUser(context.Background(), "Second", "second@example.com", "pw")

	// Promote second user
	if err := d.SetAdmin(context.Background(), u2.ID, true); err != nil {
		t.Fatalf("SetAdmin(true) error: %v", err)
	}

	isAdmin, _ := d.IsAdmin(context.Background(), u2.ID)
	if !isAdmin {
		t.Error("second user should be admin after promotion")
	}

	// Demote second user
	if err := d.SetAdmin(context.Background(), u2.ID, false); err != nil {
		t.Fatalf("SetAdmin(false) error: %v", err)
	}

	isAdmin, _ = d.IsAdmin(context.Background(), u2.ID)
	if isAdmin {
		t.Error("second user should not be admin after demotion")
	}
}

func TestSetAdmin_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.SetAdmin(context.Background(), "nonexistent-id", true)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateUser(context.Background(), "Alice", "alice@example.com", "pw")
	_, _ = d.CreateUser(context.Background(), "Bob", "bob@example.com", "pw")
	_, _ = d.CreateUser(context.Background(), "Carol", "carol@example.com", "pw")

	users, err := d.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("ListUsers() returned %d users, want 3", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("first user Name = %q, want %q", users[0].Name, "Alice")
	}
	if !users[0].IsAdmin {
		t.Error("first user should be admin")
	}
	if users[1].IsAdmin {
		t.Error("second user should not be admin")
	}
}

func TestListUsersEmptyTable(t *testing.T) {
	d := newTestDB(t)

	users, err := d.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers() error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("len(users) = %d, want 0", len(users))
	}
	if users == nil {
		t.Error("users = nil, want empty slice")
	}
}
