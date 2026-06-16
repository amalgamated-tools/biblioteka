package handlers

import (
	"context"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/smtp"
	"github.com/stretchr/testify/require"
)

func TestMakeDecryptingSMTPGetSetting_NoSecrets(t *testing.T) {
	t.Parallel()

	wrapped := makeDecryptingSMTPGetSetting(
		func(_ context.Context, key string) (string, error) {
			return "raw-" + key, nil
		},
		nil,
	)

	password, err := wrapped(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err)
	require.Equal(t, "raw-"+smtp.SettingKeyPassword, password)

	host, err := wrapped(t.Context(), smtp.SettingKeyHost)
	require.NoError(t, err)
	require.Equal(t, "raw-"+smtp.SettingKeyHost, host)
}

func TestMakeDecryptingSMTPGetSetting_DecryptsSMTPPassword(t *testing.T) {
	t.Parallel()

	jm := newTestJWT(t)
	enc, err := jm.NewSecretEncrypter()
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt("smtp-password")
	require.NoError(t, err)

	wrapped := makeDecryptingSMTPGetSetting(
		func(_ context.Context, key string) (string, error) {
			if key == smtp.SettingKeyPassword {
				return ciphertext, nil
			}
			return "other", nil
		},
		enc,
	)

	password, err := wrapped(t.Context(), smtp.SettingKeyPassword)
	require.NoError(t, err)
	require.Equal(t, "smtp-password", password)

	// Non-password keys must be returned as-is even when secrets is set.
	host, err := wrapped(t.Context(), smtp.SettingKeyHost)
	require.NoError(t, err)
	require.Equal(t, "other", host)
}

func TestMakeDecryptingSMTPGetSetting_ReturnsDecryptError(t *testing.T) {
	t.Parallel()

	jm := newTestJWT(t)
	enc, err := jm.NewSecretEncrypter()
	require.NoError(t, err)

	wrapped := makeDecryptingSMTPGetSetting(
		func(_ context.Context, _ string) (string, error) {
			return "enc:v1:not-valid-ciphertext", nil
		},
		enc,
	)

	password, err := wrapped(t.Context(), smtp.SettingKeyPassword)
	require.Error(t, err)
	require.Empty(t, password)
}
