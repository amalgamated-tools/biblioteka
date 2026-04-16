package authstore

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/goauth/auth"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite" // SQLite driver
)

// newTestDB creates an in-memory SQLite database with all migrations applied.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	t.Setenv("BIBLIOTEKA_ENV", "test")
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	_, err = sqlDB.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA foreign_keys = ON;
	`)
	require.NoError(t, err)

	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.RunMigrations(t.Context(), sqlDB, db.DialectSQLite))

	return &db.DB{DB: sqlDB, Dialect: db.DialectSQLite}
}

func TestDbUserToAuth(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		require.Nil(t, dbUserToAuth(nil))
	})

	t.Run("maps all fields", func(t *testing.T) {
		oidc := "oidc-sub-123"
		now := db.Timestamp{Time: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)}
		u := &db.User{
			ID:           "user-1",
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: "hash123",
			OIDCSubject:  &oidc,
			IsAdmin:      true,
			CreatedAt:    now,
		}

		got := dbUserToAuth(u)

		require.Equal(t, "user-1", got.ID)
		require.Equal(t, "Alice", got.Name)
		require.Equal(t, "alice@example.com", got.Email)
		require.Equal(t, "hash123", got.PasswordHash)
		require.NotNil(t, got.OIDCSubject)
		require.Equal(t, "oidc-sub-123", *got.OIDCSubject)
		require.True(t, got.IsAdmin)
		require.Equal(t, now.Time, got.CreatedAt)
	})

	t.Run("nil OIDC subject", func(t *testing.T) {
		u := &db.User{
			ID:    "user-2",
			Email: "bob@example.com",
		}

		got := dbUserToAuth(u)

		require.Nil(t, got.OIDCSubject)
	})
}

func TestDbAPIKeyToAuth(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		require.Nil(t, dbAPIKeyToAuth(nil))
	})

	t.Run("nil LastUsedAt", func(t *testing.T) {
		k := &db.APIKey{
			ID:        "key-1",
			UserID:    "user-1",
			Name:      "My Key",
			KeyHash:   "hash",
			KeyPrefix: "bib_abc",
			CreatedAt: db.Timestamp{Time: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		}

		got := dbAPIKeyToAuth(k)

		require.Equal(t, "key-1", got.ID)
		require.Equal(t, "user-1", got.UserID)
		require.Equal(t, "My Key", got.Name)
		require.Nil(t, got.LastUsedAt)
	})

	t.Run("non-nil LastUsedAt", func(t *testing.T) {
		lu := &db.Timestamp{Time: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)}
		k := &db.APIKey{
			ID:         "key-2",
			UserID:     "user-2",
			Name:       "Other Key",
			KeyHash:    "hash2",
			KeyPrefix:  "bib_def",
			LastUsedAt: lu,
			CreatedAt:  db.Timestamp{Time: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		}

		got := dbAPIKeyToAuth(k)

		require.NotNil(t, got.LastUsedAt)
		require.Equal(t, lu.Time, *got.LastUsedAt)
	})
}

func TestCreateUser_TranslatesErrEmailExists(t *testing.T) {
	// db.ErrEmailExists and auth.ErrEmailExists are separate errors.New() calls
	// with the same message string. errors.Is checks pointer identity, not string
	// equality, so the adapter must translate between them.
	require.False(t, errors.Is(db.ErrEmailExists, auth.ErrEmailExists),
		"db.ErrEmailExists and auth.ErrEmailExists must be distinct sentinels (different pointers) for the adapter translation to matter")

	d := newTestDB(t)
	adapter := &UserAdapter{DB: d}
	ctx := t.Context()

	// CreateUser: second call with same email must return auth.ErrEmailExists
	_, err := adapter.CreateUser(ctx, "Alice", "dup@example.com", "hash1")
	require.NoError(t, err)

	u, err := adapter.CreateUser(ctx, "Alice2", "dup@example.com", "hash2")
	require.Nil(t, u)
	require.ErrorIs(t, err, auth.ErrEmailExists)

	// CreateOIDCUser: same email must also return auth.ErrEmailExists
	u, err = adapter.CreateOIDCUser(ctx, "Bob", "dup@example.com", "oidc-sub")
	require.Nil(t, u)
	require.ErrorIs(t, err, auth.ErrEmailExists)
}

func TestCreateOIDCUser_TranslatesErrEmailExists(t *testing.T) {
	d := newTestDB(t)
	adapter := &UserAdapter{DB: d}
	ctx := t.Context()

	_, err := adapter.CreateOIDCUser(ctx, "First", "oidc-dup@example.com", "sub-1")
	require.NoError(t, err)

	u, err := adapter.CreateOIDCUser(ctx, "Second", "oidc-dup@example.com", "sub-2")
	require.Nil(t, u)
	require.ErrorIs(t, err, auth.ErrEmailExists)
}

func TestCountUsers_DelegatesToDB(t *testing.T) {
	d := newTestDB(t)
	adapter := &UserAdapter{DB: d}
	ctx := t.Context()

	count, err := adapter.CountUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	_, err = adapter.CreateUser(ctx, "Alice", "alice@example.com", "hash")
	require.NoError(t, err)

	count, err = adapter.CountUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
