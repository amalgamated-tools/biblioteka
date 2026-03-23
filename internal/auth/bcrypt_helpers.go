package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func mustGenerateDummyBcryptHash(secret string, name string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("generate dummy %s bcrypt hash: %w", name, err))
	}
	return hash
}
