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
		if got != tc.want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
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
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateHost(%q) error = %v, wantErr %v", tc.host, err, tc.wantErr)
		}
	}
}

func TestValidateForSend_MissingHost(t *testing.T) {
	_, err := ValidateForSend(Config{From: "from@example.com"})
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestValidateForSend_MissingFrom(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com"})
	if err == nil {
		t.Error("expected error for missing from")
	}
}

func TestValidateForSend_InvalidPort(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com", From: "from@example.com", Port: "99999"})
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestValidateForSend_InvalidTLS(t *testing.T) {
	_, err := ValidateForSend(Config{Host: "smtp.example.com", From: "from@example.com", TLS: "invalid"})
	if err == nil {
		t.Error("expected error for invalid TLS mode")
	}
}

func TestValidateForSend_UsernameWithoutPassword(t *testing.T) {
	_, err := ValidateForSend(Config{
		Host:     "smtp.example.com",
		From:     "from@example.com",
		Username: "user",
		Password: "",
		TLS:      "starttls",
	})
	if err == nil {
		t.Error("expected error for username without password")
	}
}

func TestValidateForSend_PlaintextAuthOnRemote(t *testing.T) {
	_, err := ValidateForSend(Config{
		Host:     "smtp.example.com",
		From:     "from@example.com",
		Username: "user",
		Password: "pass",
		TLS:      "none",
	})
	if err == nil {
		t.Error("expected error for plaintext auth on non-loopback host")
	}
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
	if params.Auth == nil {
		t.Error("expected non-nil Auth for username+password")
	}
}

func TestValidateForSend_Valid(t *testing.T) {
	params, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		Port: "587",
		From: "noreply@example.com",
		TLS:  "starttls",
	})
	require.NoError(t, err)
	if params.Addr != "smtp.example.com:587" {
		t.Errorf("Addr = %q, want %q", params.Addr, "smtp.example.com:587")
	}
	if params.From != "noreply@example.com" {
		t.Errorf("From = %q, want %q", params.From, "noreply@example.com")
	}
	if params.TLS != "starttls" {
		t.Errorf("TLS = %q, want %q", params.TLS, "starttls")
	}
}

func TestValidateForSend_DefaultPort(t *testing.T) {
	params, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		From: "noreply@example.com",
		TLS:  "starttls",
	})
	require.NoError(t, err)
	if params.Addr != "smtp.example.com:587" {
		t.Errorf("Addr = %q, want %q", params.Addr, "smtp.example.com:587")
	}
}

func TestValidateForSend_FromWithDisplayName(t *testing.T) {
	_, err := ValidateForSend(Config{
		Host: "smtp.example.com",
		From: "Display Name <from@example.com>",
		TLS:  "starttls",
	})
	if err == nil {
		t.Error("expected error for from address with display name")
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

	if !cfg.EnvOverride {
		t.Error("expected EnvOverride=true")
	}
	if cfg.Host != "env.smtp.example.com" {
		t.Errorf("Host = %q", cfg.Host)
	}
	if cfg.Port != "2525" {
		t.Errorf("Port = %q", cfg.Port)
	}
	if cfg.TLS != "tls" {
		t.Errorf("TLS = %q", cfg.TLS)
	}
}

func TestResolveConfig_EnvDefaults(t *testing.T) {
	t.Setenv("SMTP_HOST", "env.smtp.example.com")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_TLS", "")

	cfg := ResolveConfig(context.Background(), func(_ context.Context, _ string) (string, error) {
		return "", nil
	})

	if cfg.Port != "587" {
		t.Errorf("default Port = %q, want 587", cfg.Port)
	}
	if cfg.TLS != "starttls" {
		t.Errorf("default TLS = %q, want starttls", cfg.TLS)
	}
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

	if cfg.EnvOverride {
		t.Error("expected EnvOverride=false")
	}
	if cfg.Host != "db.smtp.example.com" {
		t.Errorf("Host = %q", cfg.Host)
	}
	if cfg.TLS != "tls" {
		t.Errorf("TLS = %q", cfg.TLS)
	}
}
