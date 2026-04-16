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
	require.NotEqual(t, "", user.ID)
	require.Equal(t, "Alice", user.Name)
	require.Equal(t, "alice@example.com", user.Email)
	require.Equal(t, "hashedpw", user.PasswordHash)
	require.False(t, user.CreatedAt.IsZero())
	require.True(t, user.IsAdmin)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "pw1")
	require.NoError(t, err, "first CreateUser() error")

	_, err = d.CreateUser(t.Context(), "Alice2", "alice@example.com", "pw2")
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestGetUserByEmail(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw")
	require.NoError(t, err, "CreateUser() error")

	found, err := d.GetUserByEmail(t.Context(), "bob@example.com")
	require.NoError(t, err, "GetUserByEmail() error")
	require.Equal(t, created.ID, found.ID)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByEmail(t.Context(), "nobody@example.com")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetUserByID(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateUser(t.Context(), "Carol", "carol@example.com", "pw")
	require.NoError(t, err, "CreateUser() error")

	found, err := d.GetUserByID(t.Context(), created.ID)
	require.NoError(t, err, "GetUserByID() error")
	require.Equal(t, "carol@example.com", found.Email)
}

func TestGetUserByID_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetUserByID(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateOIDCUser(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(t.Context(), "Dave", "dave@example.com", "oidc-sub-123")
	require.NoError(t, err, "CreateOIDCUser() error")
	require.NotNil(t, user.OIDCSubject)
	require.Equal(t, "oidc-sub-123", *user.OIDCSubject)
	require.Equal(t, "", user.PasswordHash)
}

func TestGetUserByOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateOIDCUser(t.Context(), "Eve", "eve@example.com", "oidc-sub-eve")
	require.NoError(t, err, "CreateOIDCUser() error")

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-eve")
	require.NoError(t, err, "GetUserByOIDCSubject() error")
	require.Equal(t, created.ID, found.ID)
}

func TestLinkOIDCSubject(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Frank", "frank@example.com", "pw")
	require.NoError(t, err, "CreateUser() error")

	require.NoError(t, d.LinkOIDCSubject(t.Context(), user.ID, "oidc-sub-frank"), "LinkOIDCSubject() error")

	found, err := d.GetUserByOIDCSubject(t.Context(), "oidc-sub-frank")
	require.NoError(t, err, "GetUserByOIDCSubject() error")
	require.Equal(t, user.ID, found.ID)
}

func TestLinkOIDCSubject_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.LinkOIDCSubject(t.Context(), "nonexistent-id", "some-subject")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpdatePassword(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateUser(t.Context(), "Grace", "grace@example.com", "oldhash")
	require.NoError(t, err, "CreateUser() error")

	require.NoError(t, d.UpdatePassword(t.Context(), user.ID, "newhash"), "UpdatePassword() error")

	found, err := d.GetUserByEmail(t.Context(), "grace@example.com")
	require.NoError(t, err, "GetUserByEmail() error")
	require.Equal(t, "newhash", found.PasswordHash)
}

func TestUpdatePassword_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.UpdatePassword(t.Context(), "nonexistent-id", "hash")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateUser_SecondUserNotAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	require.NoError(t, err, "CreateUser() for First error")
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

	require.True(t, u1.IsAdmin)
	require.False(t, u2.IsAdmin)
}

func TestCreateOIDCUser_FirstUserIsAdmin(t *testing.T) {
	d := newTestDB(t)

	user, err := d.CreateOIDCUser(t.Context(), "First", "first@example.com", "oidc-sub-1")
	require.NoError(t, err, "CreateOIDCUser() error")
	require.True(t, user.IsAdmin)

	u2, err := d.CreateOIDCUser(t.Context(), "Second", "second@example.com", "oidc-sub-2")
	require.NoError(t, err, "CreateOIDCUser() for Second error")
	require.False(t, u2.IsAdmin)
}

func TestIsAdmin(t *testing.T) {
	d := newTestDB(t)

	u1, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	require.NoError(t, err, "CreateUser() for First error")
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

	isAdmin, err := d.IsAdmin(t.Context(), u1.ID)
	require.NoError(t, err, "IsAdmin() error")
	require.True(t, isAdmin)

	isAdmin2, err := d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() for second user error")
	require.False(t, isAdmin2)
}

func TestSetAdmin(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateUser(t.Context(), "First", "first@example.com", "pw")
	require.NoError(t, err, "CreateUser() for First error")
	u2, err := d.CreateUser(t.Context(), "Second", "second@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Second error")

	// Promote second user
	require.NoError(t, d.SetAdmin(t.Context(), u2.ID, true), "SetAdmin(true) error")

	isAdmin, err := d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() after promotion error")
	require.True(t, isAdmin)

	// Demote second user
	require.NoError(t, d.SetAdmin(t.Context(), u2.ID, false), "SetAdmin(false) error")

	isAdmin, err = d.IsAdmin(t.Context(), u2.ID)
	require.NoError(t, err, "IsAdmin() after demotion error")
	require.False(t, isAdmin)
}

func TestSetAdmin_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.SetAdmin(t.Context(), "nonexistent-id", true)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListUsers(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateUser(t.Context(), "Alice", "alice@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Alice error")
	_, err = d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Bob error")
	_, err = d.CreateUser(t.Context(), "Carol", "carol@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Carol error")

	users, err := d.ListUsers(t.Context())
	require.NoError(t, err, "ListUsers() error")
	require.Len(t, users, 3)
	require.Equal(t, "Alice", users[0].Name)
	require.True(t, users[0].IsAdmin)
	require.False(t, users[1].IsAdmin, "second user should not be admin")
}

func TestListUsersEmptyTable(t *testing.T) {
	d := newTestDB(t)

	users, err := d.ListUsers(t.Context())
	require.NoError(t, err, "ListUsers() error")
	require.Len(t, users, 0)
	require.NotNil(t, users)
}

func TestCountUsers(t *testing.T) {
	d := newTestDB(t)

	count, err := d.CountUsers(t.Context())
	require.NoError(t, err, "CountUsers() on empty table error")
	require.Equal(t, 0, count)

	_, err = d.CreateUser(t.Context(), "Alice", "alice@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Alice error")
	_, err = d.CreateUser(t.Context(), "Bob", "bob@example.com", "pw")
	require.NoError(t, err, "CreateUser() for Bob error")

	count, err = d.CountUsers(t.Context())
	require.NoError(t, err, "CountUsers() with users error")
	require.Equal(t, 2, count)
}
