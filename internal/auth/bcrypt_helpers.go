package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// mustGenerateDummyBcryptHash generates a bcrypt hash at package init time for
// use in timing-safe comparisons when a user is not found. It panics on error
// because failure to generate a dummy hash is unrecoverable.
func mustGenerateDummyBcryptHash(secret string, name string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy %s bcrypt hash: %w", name, err))
	}
	return hash
}
