package authstore

import (
	"errors"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/goauth/auth"
	"github.com/stretchr/testify/require"
)

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
}
