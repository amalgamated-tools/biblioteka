package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the bcrypt work factor used when hashing new passwords.
// Cost 12 is chosen over bcrypt.DefaultCost (10) because modern hardware
// can brute-force cost-10 hashes significantly faster than when the
// default was standardized.
const BcryptCost = 12

// mustGenerateDummyBcryptHash generates a bcrypt hash at package init time for
// use in timing-safe comparisons when a user is not found. It panics on error
// because failure to generate a dummy hash is unrecoverable.
func mustGenerateDummyBcryptHash(secret string, name string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), BcryptCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy %s bcrypt hash: %w", name, err))
	}
	return hash
}
