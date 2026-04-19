package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestSeedInitialAdmin_SeedsWhenEmpty(t *testing.T) {
	d := newTestDB(t)
	s := &Server{DB: d}

	t.Setenv("INITIAL_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "supersecret")

	err := s.seedInitialAdmin(t.Context())
	require.NoError(t, err)

	user, err := d.GetUserByEmail(t.Context(), "admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "Admin", user.Name)
	require.True(t, user.IsAdmin, "seeded user should be admin")

	// Password should be stored as a valid bcrypt hash.
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("supersecret")))
}

func TestSeedInitialAdmin_NoOpWhenUsersExist(t *testing.T) {
	d := newTestDB(t)
	s := &Server{DB: d}

	// Create an existing user first.
	_, err := d.CreateUser(t.Context(), "Existing", "existing@example.com", "hash")
	require.NoError(t, err)

	t.Setenv("INITIAL_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "supersecret")

	err = s.seedInitialAdmin(t.Context())
	require.NoError(t, err)

	// The admin seeding email should NOT have been created.
	count, err := d.CountUsers(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, count, "only the pre-existing user should exist")
}

func TestSeedInitialAdmin_NoOpWhenEnvVarsNotSet(t *testing.T) {
	d := newTestDB(t)
	s := &Server{DB: d}

	t.Setenv("INITIAL_ADMIN_EMAIL", "")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")

	err := s.seedInitialAdmin(t.Context())
	require.NoError(t, err)

	count, err := d.CountUsers(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, count, "no users should be created when env vars are not set")
}

func TestSeedInitialAdmin_NoOpWhenEmailNotSet(t *testing.T) {
	d := newTestDB(t)
	s := &Server{DB: d}

	t.Setenv("INITIAL_ADMIN_EMAIL", "")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "supersecret")

	err := s.seedInitialAdmin(t.Context())
	require.NoError(t, err)

	count, err := d.CountUsers(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestSeedInitialAdmin_NoOpWhenPasswordNotSet(t *testing.T) {
	d := newTestDB(t)
	s := &Server{DB: d}

	t.Setenv("INITIAL_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")

	err := s.seedInitialAdmin(t.Context())
	require.NoError(t, err)

	count, err := d.CountUsers(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
