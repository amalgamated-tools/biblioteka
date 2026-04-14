package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasskeyCredentials(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	// Create a user to associate passkeys with.
	user, err := d.CreateUser(ctx, "Alice", "alice@example.com", "hash")
	require.NoError(t, err)

	t.Run("create and retrieve", func(t *testing.T) {
		cred, err := d.CreatePasskeyCredential(ctx, user.ID, "My iPhone", "cred-id-1", `{"id":"abc"}`, "aaguid-1")
		require.NoError(t, err)
		require.Equal(t, user.ID, cred.UserID)
		require.Equal(t, "My iPhone", cred.Name)
		require.Equal(t, "cred-id-1", cred.CredentialID)
		require.Equal(t, `{"id":"abc"}`, cred.CredentialData)
		require.Equal(t, "aaguid-1", cred.AAGUID)
		require.NotEmpty(t, cred.ID)

		// Retrieve by ID scoped to user.
		got, err := d.GetPasskeyCredential(ctx, cred.ID, user.ID)
		require.NoError(t, err)
		require.Equal(t, cred.ID, got.ID)
	})

	t.Run("get by credential ID", func(t *testing.T) {
		cred, err := d.CreatePasskeyCredential(ctx, user.ID, "MacBook", "cred-id-2", `{"id":"def"}`, "aaguid-2")
		require.NoError(t, err)

		got, err := d.GetPasskeyCredentialByCredentialID(ctx, "cred-id-2")
		require.NoError(t, err)
		require.Equal(t, cred.ID, got.ID)
		require.Equal(t, "MacBook", got.Name)
	})

	t.Run("list", func(t *testing.T) {
		// Create a second user to ensure isolation.
		other, err := d.CreateUser(ctx, "Bob", "bob@example.com", "hash2")
		require.NoError(t, err)
		_, err = d.CreatePasskeyCredential(ctx, other.ID, "Bob Key", "cred-id-3", `{}`, "")
		require.NoError(t, err)

		creds, err := d.ListPasskeyCredentials(ctx, user.ID)
		require.NoError(t, err)
		// Should contain the two credentials created above in this test, not Bob's.
		for _, c := range creds {
			require.Equal(t, user.ID, c.UserID)
		}
	})

	t.Run("update credential data", func(t *testing.T) {
		cred, err := d.CreatePasskeyCredential(ctx, user.ID, "Updated Key", "cred-id-4", `{"v":1}`, "")
		require.NoError(t, err)

		err = d.UpdatePasskeyCredentialData(ctx, cred.CredentialID, `{"v":2}`)
		require.NoError(t, err)

		got, err := d.GetPasskeyCredentialByCredentialID(ctx, cred.CredentialID)
		require.NoError(t, err)
		require.Equal(t, `{"v":2}`, got.CredentialData)
	})

	t.Run("delete", func(t *testing.T) {
		cred, err := d.CreatePasskeyCredential(ctx, user.ID, "To Delete", "cred-id-5", `{}`, "")
		require.NoError(t, err)

		err = d.DeletePasskeyCredential(ctx, cred.ID, user.ID)
		require.NoError(t, err)

		_, err = d.GetPasskeyCredential(ctx, cred.ID, user.ID)
		require.Error(t, err)
	})

	t.Run("delete wrong user returns not found", func(t *testing.T) {
		cred, err := d.CreatePasskeyCredential(ctx, user.ID, "Wrong User Key", "cred-id-6", `{}`, "")
		require.NoError(t, err)

		other, err := d.CreateUser(ctx, "Charlie", "charlie@example.com", "hash3")
		require.NoError(t, err)

		err = d.DeletePasskeyCredential(ctx, cred.ID, other.ID)
		require.Error(t, err)
	})
}

func TestPasskeyChallenges(t *testing.T) {
	d := newTestDB(t)
	ctx := t.Context()

	uid := "user-abc"

	t.Run("create and get-delete", func(t *testing.T) {
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		challenge, err := d.CreatePasskeyChallenge(ctx, &uid, `{"challenge":"xyz"}`, expiresAt)
		require.NoError(t, err)
		require.NotEmpty(t, challenge.ID)
		require.Equal(t, &uid, challenge.UserID)

		got, err := d.GetAndDeletePasskeyChallenge(ctx, challenge.ID)
		require.NoError(t, err)
		require.Equal(t, challenge.ID, got.ID)
		require.Equal(t, `{"challenge":"xyz"}`, got.SessionData)

		// Second retrieval should return not found (deleted).
		_, err = d.GetAndDeletePasskeyChallenge(ctx, challenge.ID)
		require.Error(t, err)
	})

	t.Run("nil user_id for login challenges", func(t *testing.T) {
		expiresAt := time.Now().UTC().Add(5 * time.Minute)
		challenge, err := d.CreatePasskeyChallenge(ctx, nil, `{"login":"true"}`, expiresAt)
		require.NoError(t, err)
		require.Nil(t, challenge.UserID)

		got, err := d.GetAndDeletePasskeyChallenge(ctx, challenge.ID)
		require.NoError(t, err)
		require.Nil(t, got.UserID)
	})

	t.Run("delete expired", func(t *testing.T) {
		// Create an already-expired challenge.
		past := time.Now().UTC().Add(-time.Minute)
		expired, err := d.CreatePasskeyChallenge(ctx, nil, `{"expired":"true"}`, past)
		require.NoError(t, err)

		// Create a still-valid challenge.
		future := time.Now().UTC().Add(5 * time.Minute)
		valid, err := d.CreatePasskeyChallenge(ctx, nil, `{"valid":"true"}`, future)
		require.NoError(t, err)

		err = d.DeleteExpiredPasskeyChallenges(ctx)
		require.NoError(t, err)

		// Expired challenge should be gone.
		_, err = d.GetAndDeletePasskeyChallenge(ctx, expired.ID)
		require.Error(t, err)

		// Valid challenge should still exist.
		got, err := d.GetAndDeletePasskeyChallenge(ctx, valid.ID)
		require.NoError(t, err)
		require.Equal(t, valid.ID, got.ID)
	})
}
