package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecretEncrypter_RoundTrip(t *testing.T) {
	enc, err := newSecretEncrypter([]byte("test-secret-key-for-encryption"))
	require.NoError(t, err)

	plaintext := "super-secret-password"
	encrypted, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(encrypted, secretEncryptPrefix), "encrypted value should have enc:v1: prefix")
	require.NotEqual(t, plaintext, encrypted)

	decrypted, err := enc.Decrypt(encrypted)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestSecretEncrypter_EncryptEmptyString(t *testing.T) {
	enc, err := newSecretEncrypter([]byte("test-secret"))
	require.NoError(t, err)

	encrypted, err := enc.Encrypt("")
	require.NoError(t, err)
	require.Equal(t, "", encrypted, "empty string should be returned unchanged")

	decrypted, err := enc.Decrypt("")
	require.NoError(t, err)
	require.Equal(t, "", decrypted)
}

func TestSecretEncrypter_BackwardCompat_PlaintextReturned(t *testing.T) {
	enc, err := newSecretEncrypter([]byte("test-secret"))
	require.NoError(t, err)

	// A value without the prefix (legacy plaintext) is returned as-is.
	legacy := "old-plaintext-password"
	decrypted, err := enc.Decrypt(legacy)
	require.NoError(t, err)
	require.Equal(t, legacy, decrypted)
}

func TestSecretEncrypter_ProducesUniqueCiphertexts(t *testing.T) {
	enc, err := newSecretEncrypter([]byte("test-secret"))
	require.NoError(t, err)

	a, err := enc.Encrypt("same-value")
	require.NoError(t, err)
	b, err := enc.Encrypt("same-value")
	require.NoError(t, err)

	// Each encryption uses a fresh random nonce, so ciphertexts must differ.
	require.NotEqual(t, a, b, "two encryptions of the same value should produce different ciphertexts")
}

func TestSecretEncrypter_WrongKeyFails(t *testing.T) {
	encA, err := newSecretEncrypter([]byte("key-a"))
	require.NoError(t, err)
	encB, err := newSecretEncrypter([]byte("key-b"))
	require.NoError(t, err)

	encrypted, err := encA.Encrypt("secret")
	require.NoError(t, err)

	_, err = encB.Decrypt(encrypted)
	require.Error(t, err, "decrypting with a different key should fail")
}

func TestSecretEncrypter_TamperedCiphertextFails(t *testing.T) {
	enc, err := newSecretEncrypter([]byte("test-secret"))
	require.NoError(t, err)

	encrypted, err := enc.Encrypt("secret")
	require.NoError(t, err)

	// Corrupt a byte in the base64 payload.
	tampered := secretEncryptPrefix + "AAAA" + encrypted[len(secretEncryptPrefix)+4:]
	_, err = enc.Decrypt(tampered)
	require.Error(t, err, "tampered ciphertext should fail authentication")
}

func TestJWTManager_NewSecretEncrypter(t *testing.T) {
	jm, err := NewJWTManager("test-jwt-secret-32-bytes-long-xx", 0)
	require.NoError(t, err)

	enc, err := jm.NewSecretEncrypter()
	require.NoError(t, err)
	require.NotNil(t, enc)

	// Verify it actually works end-to-end.
	ciphertext, err := enc.Encrypt("my-password")
	require.NoError(t, err)
	plaintext, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "my-password", plaintext)
}
