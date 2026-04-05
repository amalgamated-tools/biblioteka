package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/require"
)

func TestMustGenerateDummyBcryptHash(t *testing.T) {
	hash := mustGenerateDummyBcryptHash("dummy-secret", "test")

	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("dummy-secret")), "generated hash did not match original secret")
	if err := bcrypt.CompareHashAndPassword(hash, []byte("other-secret")); err == nil {
		require.Fail(t, "generated hash unexpectedly matched different secret")
	}
}
