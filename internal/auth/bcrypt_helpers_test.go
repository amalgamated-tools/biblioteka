package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestMustGenerateDummyBcryptHash(t *testing.T) {
	hash := mustGenerateDummyBcryptHash("dummy-secret", "test")

	if err := bcrypt.CompareHashAndPassword(hash, []byte("dummy-secret")); err != nil {
		t.Fatalf("generated hash did not match original secret: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("other-secret")); err == nil {
		t.Fatal("generated hash unexpectedly matched different secret")
	}
}
