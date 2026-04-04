package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "hashedpw")
	require.NoError(t, err, "CreateUser() error")
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
	require.NoError(t, err, "first CreateUser() error")

	_, err = d.CreateUser(t.Context(), "Alice2", "alice@example.com", "pw2")
	if err != ErrEmailExists {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw")
	require.NoError(t, err, "CreateUser() error")

	found, err := d.GetUserByEmail(t.Context(), "bob@example.com")
	require.NoError(t, err, "GetUserByEmail() error")
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
	require.NoError(t, err, "CreateUser() error")

	found, err := d.GetUserByID(t.Context(), created.ID)
	require.NoError(t, err, "GetUserByID() error")
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
	require.NoError(t, err, "CreateOIDCUser() error")
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
	require.NoError(t, err, "CreateOIDCUser() error")

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-eve")
	require.NoError(t, err, "GetUserByOIDCSubject() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
}

func TestLinkOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Frank", "frank@example.com", "pw")
	require.NoError(t, err, "CreateUser() error")

	if err := d.LinkOIDCSubject(t.Context(), user.ID, "oidc-sub-frank"); err != nil {
		require.NoError(t, err, "LinkOIDCSubject() error")
	}

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-frank")
	require.NoError(t, err, "GetUserByOIDCSubject() error")
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
	require.NoError(t, err, "CreateUser() error")

	if err := d.UpdatePassword(t.Context(), user.ID, "newhash"); err != nil {
		require.NoError(t, err, "UpdatePassword() error")
	}

	found, err := d.GetUserByEmail(t.Context(), "grace@example.com")
	require.NoError(t, err, "GetUserByEmail() error")
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
	require.NoError(t, err, "CreateUser() for First error")
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

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
	require.NoError(t, err, "CreateOIDCUser() error")
	if !user.IsAdmin {
		t.Error("first OIDC user should be admin")
	}

	u2, err := d.CreateOIDCUser(t.Context(), "Second", "second@example.com", "oidc-sub-2")
	require.NoError(t, err, "CreateOIDCUser() for Second error")
	if u2.IsAdmin {
		t.Error("second OIDC user should not be admin")
	}
}

func TestIsAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	require.NoError(t, err, "CreateUser() for First error")
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

	isAdmin, err := d.IsAdmin(t.Context(), u1.ID)
	require.NoError(t, err, "IsAdmin() error")
	if !isAdmin {
		t.Error("first user should be admin")
	}

	isAdmin2, err := d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() for second user error")
	if isAdmin2 {
		t.Error("second user should not be admin")
	}
}

func TestSetAdmin(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw"); err != nil {
		require.NoError(t, err, "CreateUser() for First error")
	}
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

	// Promote second user
	if err := d.SetAdmin(t.Context(), u2.ID, true); err != nil {
		require.NoError(t, err, "SetAdmin(true) error")
	}

	isAdmin, err := d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() after promotion error")
	if !isAdmin {
		t.Error("second user should be admin after promotion")
	}

	// Demote second user
	if err := d.SetAdmin(t.Context(), u2.ID, false); err != nil {
		require.NoError(t, err, "SetAdmin(false) error")
	}

	isAdmin, err = d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() after demotion error")
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
		require.NoError(t, err, "CreateUser() for Alice error")
	}
	if _, err := d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw"); err != nil {
		require.NoError(t, err, "CreateUser() for Bob error")
	}
	if _, err := d.CreateUser(t.Context(), "Carol", "carol@example.com", "pw"); err != nil {
		require.NoError(t, err, "CreateUser() for Carol error")
	}

	users, err := d.ListUsers(t.Context())
	require.NoError(t, err, "ListUsers() error")
	if len(users) != 3 {
		require.Failf(t, "failed", "ListUsers() returned %d users, want 3", len(users))
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
	require.NoError(t, err, "ListUsers() error")
	if len(users) != 0 {
		t.Errorf("len(users) = %d, want 0", len(users))
	}
	if users == nil {
		t.Error("users = nil, want empty slice")
	}
}
