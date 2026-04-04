package db

import (
	"database/sql"
	"testing"
)

func TestCreateUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "hashedpw")
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

	_, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "pw1")
	if err != nil {
		t.Fatalf("first CreateUser() error: %v", err)
	}

	_, err = d.CreateUser(t.Context(), "Alice2", "alice@example.com", "pw2")
	if err != ErrEmailExists {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}

	found, err := d.GetUserByEmail(t.Context(), "bob@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByEmail(t.Context(), "nobody@example.com")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateUser(t.Context(), "Carol", "carol@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}

	found, err := d.GetUserByID(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error: %v", err)
	}
	if found.Email != "carol@example.com" {
		t.Errorf("Email = %q, want %q", found.Email, "carol@example.com")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByID(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateOIDCUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(t.Context(), "Dave", "dave@example.com", "oidc-sub-123")
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

	created, err := d.CreateOIDCUser(t.Context(), "Eve", "eve@example.com", "oidc-sub-eve")
	if err != nil {
		t.Fatalf("CreateOIDCUser() error: %v", err)
	}

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-eve")
	if err != nil {
		t.Fatalf("GetUserByOIDCSubject() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestLinkOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Frank", "frank@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}

	if err := d.LinkOIDCSubject(t.Context(), user.ID, "oidc-sub-frank"); err != nil {
		t.Fatalf("LinkOIDCSubject() error: %v", err)
	}

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-frank")
	if err != nil {
		t.Fatalf("GetUserByOIDCSubject() error: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("ID after linking = %q, want %q", found.ID, user.ID)
	}
}

func TestLinkOIDCSubject_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.LinkOIDCSubject(t.Context(), "nonexistent-id", "some-subject")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Grace", "grace@example.com", "oldhash")
	if err != nil {
		t.Fatalf("CreateUser() error: %v", err)
	}

	if err := d.UpdatePassword(t.Context(), user.ID, "newhash"); err != nil {
		t.Fatalf("UpdatePassword() error: %v", err)
	}

	found, err := d.GetUserByEmail(t.Context(), "grace@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error: %v", err)
	}
	if found.PasswordHash != "newhash" {
		t.Errorf("PasswordHash = %q, want %q", found.PasswordHash, "newhash")
	}
}

func TestUpdatePassword_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.UpdatePassword(t.Context(), "nonexistent-id", "hash")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateUser_SecondUserNotAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() for First error: %v", err)
	}
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() for Second error: %v", err)
	}

	if !u1.IsAdmin {
		t.Error("first user should be admin")
	}
	if u2.IsAdmin {
		t.Error("second user should not be admin")
	}
}

func TestCreateOIDCUser_FirstUserIsAdmin(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(t.Context(), "First", "first@example.com", "oidc-sub-1")
	if err != nil {
		t.Fatalf("CreateOIDCUser() error: %v", err)
	}
	if !user.IsAdmin {
		t.Error("first OIDC user should be admin")
	}

	u2, err := d.CreateOIDCUser(t.Context(), "Second", "second@example.com", "oidc-sub-2")
	if err != nil {
		t.Fatalf("CreateOIDCUser() for Second error: %v", err)
	}
	if u2.IsAdmin {
		t.Error("second OIDC user should not be admin")
	}
}

func TestIsAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() for First error: %v", err)
	}
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() for Second error: %v", err)
	}

	isAdmin, err := d.IsAdmin(t.Context(), u1.ID)
	if err != nil {
		t.Fatalf("IsAdmin() error: %v", err)
	}
	if !isAdmin {
		t.Error("first user should be admin")
	}

	isAdmin2, err := d.IsAdmin(t.Context(), u2.ID)
	if err != nil {
		t.Fatalf("IsAdmin() for second user error: %v", err)
	}
	if isAdmin2 {
		t.Error("second user should not be admin")
	}
}

func TestSetAdmin(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw"); err != nil {
		t.Fatalf("CreateUser() for First error: %v", err)
	}
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	if err != nil {
		t.Fatalf("CreateUser() for Second error: %v", err)
	}

	// Promote second user
	if err := d.SetAdmin(t.Context(), u2.ID, true); err != nil {
		t.Fatalf("SetAdmin(true) error: %v", err)
	}

	isAdmin, err := d.IsAdmin(t.Context(), u2.ID)
	if err != nil {
		t.Fatalf("IsAdmin() after promotion error: %v", err)
	}
	if !isAdmin {
		t.Error("second user should be admin after promotion")
	}

	// Demote second user
	if err := d.SetAdmin(t.Context(), u2.ID, false); err != nil {
		t.Fatalf("SetAdmin(false) error: %v", err)
	}

	isAdmin, err = d.IsAdmin(t.Context(), u2.ID)
	if err != nil {
		t.Fatalf("IsAdmin() after demotion error: %v", err)
	}
	if isAdmin {
		t.Error("second user should not be admin after demotion")
	}
}

func TestSetAdmin_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.SetAdmin(t.Context(), "nonexistent-id", true)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "pw"); err != nil {
		t.Fatalf("CreateUser() for Alice error: %v", err)
	}
	if _, err := d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw"); err != nil {
		t.Fatalf("CreateUser() for Bob error: %v", err)
	}
	if _, err := d.CreateUser(t.Context(), "Carol", "carol@example.com", "pw"); err != nil {
		t.Fatalf("CreateUser() for Carol error: %v", err)
	}

	users, err := d.ListUsers(t.Context())
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

	users, err := d.ListUsers(t.Context())
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
