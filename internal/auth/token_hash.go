package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashHighEntropyToken returns the hex-encoded SHA-256 hash of a high-entropy token.
// SHA-256 is appropriate here because these tokens are random values, not passwords.
func hashHighEntropyToken(token string) string {
	h := sha256.Sum256([]byte(token)) // #nosec G401 -- not a password; high-entropy token
	return hex.EncodeToString(h[:])
}
