package smtp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"smtp.example.com", false},
		{"10.0.0.1", false},
		{"", false},
	}
	for _, tc := range cases {
		got := IsLoopbackHost(tc.host)
		require.Equal(t, tc.want, got, "IsLoopbackHost(%q)", tc.host)
	}
}

func TestValidateHost(t *testing.T) {
	cases := []struct {
		host    string
		wantErr bool
	}{
		{"smtp.example.com", false},
		// Private/loopback/link-local addresses must be rejected (SSRF protection).
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"localhost", true},
		{"LOCALHOST", true},
		{"", true},
		{"host with space", true},
		{"[::1]", true},
		{"host:port", true},
		{"not-an-ipv6::", true},
	}
	for _, tc := range cases {
		err := ValidateHost(tc.host)
		if tc.wantErr {
			require.Error(t, err, "ValidateHost(%q)", tc.host)
		} else {
			require.NoError(t, err, "ValidateHost(%q)", tc.host)
		}
	}
}

func TestValidateForSend_MissingHost(t *testing.T) {
	_, err := ValidateForSend(Config{From: "from@example.com"})
	require.Error(t, err, "expected error for missing host")
}

func TestValidateForSend_MissingFrom(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com"})
	require.Error(t, err, "expected error for missing from")
}

func TestValidateForSend_InvalidPort(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com", From: "from@example.com", Port: "99999"})
	require.Error(t, err, "expected error for invalid port")
}

func TestValidateForSend_InvalidTLS(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com", From: "from@example.com", TLS: "invalid"})
	require.Error(t, err, "expected error for invalid TLS mode")
}

func TestValidateForSend_UsernameWithoutPassword(t *testing.T) {
	_, err := ValidateForSend(Config{
		Host:     "smtp.example.com",
		From:     "from@example.com",
		Username: "user",
		Password: "",
		TLS:      "starttls",
	})
	require.Error(t, err, "expected error for username without password")
}

func TestValidateForSend_PlaintextAuthOnRemote(t *testing.T) {
	_, err := ValidateForSend(Config{
		Host:     "smtp.example.com",
		From:     "from@example.com",
		Username: "user",
		Password: "pass",
		TLS:      "none",
	})
	require.Error(t, err, "expected error for plaintext auth on non-loopback host")
}

func TestValidateForSend_PlaintextAuthOnLoopback(t *testing.T) {
	// localhost is now rejected by SSRF validation, so plaintext auth on
	// loopback is no longer a supported configuration.
	_, err := ValidateForSend(Config{
		Host:     "localhost",
		From:     "from@example.com",
		Username: "user",
		Password: "pass",
		TLS:      "none",
	})
	require.Error(t, err, "expected error: localhost is a private address")
}

func TestValidateForSend_Valid(t *testing.T) {
	params, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		Port: "587",
		From: "noreply@example.com",
		TLS:  "starttls",
	})
	require.NoError(t, err)
	require.Equal(t, "smtp.example.com:587", params.Addr, "Addr")
	require.Equal(t, "noreply@example.com", params.From, "From")
	require.Equal(t, "noreply@example.com", params.FromHeader, "FromHeader")
	require.Equal(t, "starttls", params.TLS, "TLS")
}

func TestValidateForSend_DefaultPort(t *testing.T) {
	params, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		From: "noreply@example.com",
		TLS:  "starttls",
	})
	require.NoError(t, err)
	require.Equal(t, "smtp.example.com:587", params.Addr, "Addr")
}

func TestValidateForSend_FromWithDisplayName(t *testing.T) {
	params, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		From: "Display Name <from@example.com>",
		TLS:  "starttls",
	})
	require.NoError(t, err, "expected no error for from address with display name")
	require.Equal(t, "from@example.com", params.From, "From should be bare address for SMTP envelope")
	require.Equal(t, `"Display Name" <from@example.com>`, params.FromHeader, "FromHeader should include display name")
}

func TestValidateForSend_ErrorsAreValidationError(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"missing host", Config{From: "from@example.com"}},
		{"missing from", Config{Host: "smtp.example.com"}},
		{"invalid port", Config{Host: "smtp.example.com", From: "from@example.com", Port: "99999"}},
		{"invalid TLS", Config{Host: "smtp.example.com", From: "from@example.com", TLS: "bad"}},
		{"username without password", Config{Host: "smtp.example.com", From: "from@example.com", Username: "u", TLS: "starttls"}},
		{"invalid host characters", Config{Host: "host with space", From: "from@example.com"}},
		{"host with brackets", Config{Host: "[::1]", From: "from@example.com"}},
		{"RFC-1918 class A", Config{Host: "10.0.0.1", From: "from@example.com"}},
		{"RFC-1918 class B", Config{Host: "172.16.0.1", From: "from@example.com"}},
		{"RFC-1918 class C", Config{Host: "192.168.1.1", From: "from@example.com"}},
		{"link-local (AWS IMDS)", Config{Host: "169.254.169.254", From: "from@example.com"}},
		{"loopback IPv4", Config{Host: "127.0.0.1", From: "from@example.com"}},
		{"loopback IPv6", Config{Host: "::1", From: "from@example.com"}},
		{"localhost hostname", Config{Host: "localhost", From: "from@example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateForSend(tc.cfg)
			require.Error(t, err, "expected an error")
			var ve *ValidationError
			require.ErrorAs(t, err, &ve, "error should be *ValidationError, got %T: %v", err, err)
		})
	}
}

func TestResolveConfig_EnvOverride(t *testing.T) {
	t.Setenv("SMTP_HOST", "env.smtp.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USERNAME", "envuser")
	t.Setenv("SMTP_PASSWORD", "envpass")
	t.Setenv("SMTP_FROM", "env@example.com")
	t.Setenv("SMTP_TLS", "tls")

	cfg := ResolveConfig(context.Background(), func(_ context.Context, _ string) (string, error) {
		return "", nil
	})

	require.True(t, cfg.EnvOverride, "expected EnvOverride=true")
	require.Equal(t, "env.smtp.example.com", cfg.Host, "Host")
	require.Equal(t, "2525", cfg.Port, "Port")
	require.Equal(t, "tls", cfg.TLS, "TLS")
}

func TestResolveConfig_EnvDefaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "env.smtp.example.com")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_TLS", "")

	cfg := ResolveConfig(context.Background(), func(_ context.Context, _ string) (string, error) {
		return "", nil
	})

	require.Equal(t, "587", cfg.Port, "default Port")
	require.Equal(t, "starttls", cfg.TLS, "default TLS")
}

func TestResolveConfig_DBFallback(t *testing.T) {
	// Ensure SMTP_HOST is not set so we fall through to the getter.
	t.Setenv("SMTP_HOST", "")

	settings := map[string]string{
		SettingKeyHost:     "db.smtp.example.com",
		SettingKeyPort:     "465",
		SettingKeyUsername: "dbuser",
		SettingKeyPassword: "dbpass",
		SettingKeyFrom:     "db@example.com",
		SettingKeyTLS:      "tls",
	}
	cfg := ResolveConfig(context.Background(), func(_ context.Context, key string) (string, error) {
		return settings[key], nil
	})

	require.False(t, cfg.EnvOverride, "expected EnvOverride=false")
	require.Equal(t, "db.smtp.example.com", cfg.Host, "Host")
	require.Equal(t, "tls", cfg.TLS, "TLS")
}
