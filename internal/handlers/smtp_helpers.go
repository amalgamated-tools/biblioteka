package handlers

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"
)

// makeDecryptingSMTPGetSetting wraps getSetting to transparently decrypt the
// stored SMTP password when secrets is non-nil.
func makeDecryptingSMTPGetSetting(
	getSetting func(context.Context, string) (string, error),
	secrets *auth.SecretEncrypter,
) func(context.Context, string) (string, error) {
	if secrets == nil {
		return getSetting
	}

	return func(ctx context.Context, key string) (string, error) {
		val, err := getSetting(ctx, key)
		if err != nil {
			return "", err
		}

		if key == smtp.SettingKeyPassword {
			decrypted, decErr := secrets.Decrypt(val)
			if decErr != nil {
				slog.WarnContext(ctx, "failed to decrypt stored SMTP password; password will be empty",
					slog.Any(otelkeys.Error, decErr),
				)

				return "", decErr
			}

			return decrypted, nil
		}

		return val, nil
	}
}
