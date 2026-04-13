package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

// secretEncryptPrefix is the version prefix for values encrypted by SecretEncrypter.
const secretEncryptPrefix = "enc:v1:"

// SecretEncrypter encrypts and decrypts sensitive settings stored in the
// database using AES-256-GCM with a key derived from the JWT secret via HKDF.
// Values without the prefix are returned unchanged for backward compatibility
// with settings stored before encryption was introduced.
type SecretEncrypter struct {
	key []byte // 32-byte AES-256 key
}

// newSecretEncrypter derives a 32-byte AES-256 key from secret via HKDF and
// returns a new SecretEncrypter. secret is typically the raw JWT signing secret.
func newSecretEncrypter(secret []byte) (*SecretEncrypter, error) {
	key := make([]byte, 32)
	r := hkdf.New(sha256.New, secret, nil, []byte("settings-secret-v1"))
	if _, err := r.Read(key); err != nil {
		return nil, fmt.Errorf("derive settings encryption key: %w", err)
	}
	return &SecretEncrypter{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a string prefixed
// with "enc:v1:". An empty plaintext is returned unchanged.
func (e *SecretEncrypter) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	// Seal appends the ciphertext (and GCM authentication tag) to nonce.
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return secretEncryptPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value previously encrypted by Encrypt. Values without the
// "enc:v1:" prefix are returned unchanged for backward compatibility with
// settings stored before encryption was introduced.
func (e *SecretEncrypter) Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, secretEncryptPrefix) {
		// Legacy plaintext value — return as-is for backward compatibility.
		return value, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, secretEncryptPrefix))
	if err != nil {
		return "", fmt.Errorf("base64 decode encrypted value: %w", err)
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("encrypted value is too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plaintext), nil
}
