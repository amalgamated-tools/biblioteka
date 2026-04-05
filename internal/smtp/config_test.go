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
		{"127.0.0.1", false},
		{"::1", false},
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
	params, err := ValidateForSend(Config{
		Host:     "localhost",
		From:     "from@example.com",
		Username: "user",
		Password: "pass",
		TLS:      "none",
	})
	require.NoError(t, err)
	require.NotNil(t, params.Auth, "expected non-nil Auth for username+password")
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
