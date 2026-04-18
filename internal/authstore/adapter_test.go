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

// ---- PasskeyAdapter tests ----

func newPasskeyAdapter(t *testing.T) (*PasskeyAdapter, *db.User) {
	t.Helper()
	d := newTestDB(t)
	u, err := d.CreateUser(t.Context(), "Passkey User", "pk@example.com", "hash")
	require.NoError(t, err)
	return &PasskeyAdapter{DB: d}, u
}

func TestPasskeyAdapter_CreateChallenge_NilUserID(t *testing.T) {
	a, _ := newPasskeyAdapter(t)

	expiresAt := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Second)
	c, err := a.CreateChallenge(t.Context(), nil, `{"challenge":"login"}`, expiresAt)

	require.NoError(t, err)
	require.NotEmpty(t, c.ID)
	require.Nil(t, c.UserID)
	require.Equal(t, `{"challenge":"login"}`, c.SessionData)
	require.WithinDuration(t, expiresAt, c.ExpiresAt, time.Second)
	require.False(t, c.CreatedAt.IsZero())
}

func TestPasskeyAdapter_CreateChallenge_WithUserID(t *testing.T) {
	a, u := newPasskeyAdapter(t)

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	c, err := a.CreateChallenge(t.Context(), &u.ID, `{"challenge":"register"}`, expiresAt)

	require.NoError(t, err)
	require.NotNil(t, c.UserID)
	require.Equal(t, u.ID, *c.UserID)
	require.Equal(t, `{"challenge":"register"}`, c.SessionData)
}

func TestPasskeyAdapter_GetAndDeleteChallenge_RemovesOnFirstFetch(t *testing.T) {
	a, _ := newPasskeyAdapter(t)

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	created, err := a.CreateChallenge(t.Context(), nil, `{"challenge":"once"}`, expiresAt)
	require.NoError(t, err)

	// First call: succeeds.
	got, err := a.GetAndDeleteChallenge(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, `{"challenge":"once"}`, got.SessionData)

	// Second call: challenge is gone.
	_, err = a.GetAndDeleteChallenge(t.Context(), created.ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "second fetch must fail — challenge was deleted")
}

func TestPasskeyAdapter_GetAndDeleteChallenge_MissingID(t *testing.T) {
	a, _ := newPasskeyAdapter(t)

	_, err := a.GetAndDeleteChallenge(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPasskeyAdapter_DeleteExpiredChallenges(t *testing.T) {
	a, _ := newPasskeyAdapter(t)
	ctx := t.Context()

	// Insert one already-expired challenge and one valid one.
	expired := time.Now().UTC().Add(-1 * time.Minute)
	future := time.Now().UTC().Add(10 * time.Minute)

	expiredC, err := a.CreateChallenge(ctx, nil, `{"exp":"yes"}`, expired)
	require.NoError(t, err)
	validC, err := a.CreateChallenge(ctx, nil, `{"exp":"no"}`, future)
	require.NoError(t, err)

	require.NoError(t, a.DeleteExpiredChallenges(ctx))

	// Expired one must be gone.
	_, err = a.GetAndDeleteChallenge(ctx, expiredC.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Valid one must still exist.
	got, err := a.GetAndDeleteChallenge(ctx, validC.ID)
	require.NoError(t, err)
	require.Equal(t, validC.ID, got.ID)
}

func TestPasskeyAdapter_CreateCredential_MapsAllFields(t *testing.T) {
	a, u := newPasskeyAdapter(t)

	c, err := a.CreateCredential(t.Context(), u.ID, "My Key", "cred-id-1", `{"data":"v1"}`, "aaguid-abc")

	require.NoError(t, err)
	require.NotEmpty(t, c.ID)
	require.Equal(t, u.ID, c.UserID)
	require.Equal(t, "My Key", c.Name)
	require.Equal(t, "cred-id-1", c.CredentialID)
	require.Equal(t, `{"data":"v1"}`, c.CredentialData)
	require.Equal(t, "aaguid-abc", c.AAGUID)
	require.False(t, c.CreatedAt.IsZero())
}

func TestPasskeyAdapter_ListCredentialsByUser_EmptyAndPopulated(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	// Empty initially.
	creds, err := a.ListCredentialsByUser(ctx, u.ID)
	require.NoError(t, err)
	require.Empty(t, creds)

	// Add two credentials.
	_, err = a.CreateCredential(ctx, u.ID, "Key A", "cred-a", `{"k":"a"}`, "aaguid-a")
	require.NoError(t, err)
	_, err = a.CreateCredential(ctx, u.ID, "Key B", "cred-b", `{"k":"b"}`, "aaguid-b")
	require.NoError(t, err)

	creds, err = a.ListCredentialsByUser(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, creds, 2)
}

func TestPasskeyAdapter_ListCredentialsByUser_Isolation(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	u1, err := d.CreateUser(ctx, "User One", "u1@example.com", "hash")
	require.NoError(t, err)
	u2, err := d.CreateUser(ctx, "User Two", "u2@example.com", "hash")
	require.NoError(t, err)

	a := &PasskeyAdapter{DB: d}
	_, err = a.CreateCredential(ctx, u1.ID, "Key A", "cred-u1", `{}`, "")
	require.NoError(t, err)

	// u2 must not see u1's credentials.
	creds, err := a.ListCredentialsByUser(ctx, u2.ID)
	require.NoError(t, err)
	require.Empty(t, creds)
}

func TestPasskeyAdapter_FindCredentialByCredentialID(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	created, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-find-1", `{"v":1}`, "guid-1")
	require.NoError(t, err)

	found, err := a.FindCredentialByCredentialID(ctx, "cred-find-1")
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "cred-find-1", found.CredentialID)

	// Missing credential ID returns sql.ErrNoRows.
	_, err = a.FindCredentialByCredentialID(ctx, "not-found")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPasskeyAdapter_FindCredentialByIDAndUser(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	created, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-byid-1", `{}`, "guid-1")
	require.NoError(t, err)

	// Correct user: found.
	found, err := a.FindCredentialByIDAndUser(ctx, created.ID, u.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, found.ID)

	// Wrong user: not found.
	_, err = a.FindCredentialByIDAndUser(ctx, created.ID, "wrong-user-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPasskeyAdapter_UpdateCredentialData(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	_, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-upd-1", `{"sign_count":0}`, "guid-1")
	require.NoError(t, err)

	// Update the stored data.
	require.NoError(t, a.UpdateCredentialData(ctx, u.ID, "cred-upd-1", `{"sign_count":1}`))

	// Verify the change is visible.
	found, err := a.FindCredentialByCredentialID(ctx, "cred-upd-1")
	require.NoError(t, err)
	require.Equal(t, `{"sign_count":1}`, found.CredentialData)

	// Wrong user: credential not affected, returns sql.ErrNoRows.
	err = a.UpdateCredentialData(ctx, "wrong-user", "cred-upd-1", `{"sign_count":99}`)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPasskeyAdapter_DeleteCredential(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	created, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-del-1", `{}`, "guid-1")
	require.NoError(t, err)

	// Correct user: deleted.
	require.NoError(t, a.DeleteCredential(ctx, created.ID, u.ID))

	// Credential is gone.
	_, err = a.FindCredentialByCredentialID(ctx, "cred-del-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPasskeyAdapter_DeleteCredential_WrongUser(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	created, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-del-2", `{}`, "guid-2")
	require.NoError(t, err)

	// Wrong user: returns sql.ErrNoRows (credential not affected).
	err = a.DeleteCredential(ctx, created.ID, "wrong-user")
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Credential still exists.
	_, err = a.FindCredentialByCredentialID(ctx, "cred-del-2")
	require.NoError(t, err)
}

// TestPasskeyAdapter_ReturnType_IsAuthPasskeyCredential verifies that CreateCredential
// returns *auth.PasskeyCredential and ListCredentialsByUser returns []auth.PasskeyCredential,
// confirming the field mapping is correct.
func TestPasskeyAdapter_ReturnType_IsAuthPasskeyCredential(t *testing.T) {
	a, u := newPasskeyAdapter(t)
	ctx := t.Context()

	created, err := a.CreateCredential(ctx, u.ID, "My Key", "cred-type-1", `{"type":"check"}`, "guid-x")
	require.NoError(t, err)

	// ListCredentialsByUser must return auth.PasskeyCredential elements.
	creds, err := a.ListCredentialsByUser(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, creds, 1)

	require.Equal(t, created.ID, creds[0].ID)
	require.Equal(t, u.ID, creds[0].UserID)
	require.Equal(t, "My Key", creds[0].Name)
	require.Equal(t, "cred-type-1", creds[0].CredentialID)
	require.Equal(t, `{"type":"check"}`, creds[0].CredentialData)
	require.Equal(t, "guid-x", creds[0].AAGUID)
}
