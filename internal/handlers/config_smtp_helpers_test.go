package handlers

import (
	"testing"
)

// TestValidateSMTPHost verifies that validateSMTPHost accepts valid hostnames
// and rejects invalid ones.
func TestValidateSMTPHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{name: "valid hostname", host: "smtp.example.com", wantErr: false},
		{name: "valid IPv4", host: "192.168.1.1", wantErr: false},
		{name: "valid IPv6", host: "::1", wantErr: false},
		{name: "localhost", host: "localhost", wantErr: false},
		{name: "empty host", host: "", wantErr: true},
		{name: "host with port", host: "smtp.example.com:587", wantErr: true},
		{name: "host with brackets IPv6", host: "[::1]", wantErr: true},
		{name: "host with control char", host: "smtp\x00.example.com", wantErr: true},
		{name: "host with space", host: "smtp example.com", wantErr: true},
		{name: "host with newline", host: "smtp.example.com\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSMTPHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSMTPHost(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}
}

// TestValidateSMTPForSend_ValidConfig verifies that a fully valid configuration
// produces correct send params.
func TestValidateSMTPForSend_ValidConfig(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{
		Host:     "smtp.example.com",
		Port:     "587",
		Username: "user@example.com",
		Password: "secret",
		From:     "sender@example.com",
		TLS:      "starttls",
	}

	params, err := validateSMTPForSend(cfg)
	if err != nil {
		t.Fatalf("validateSMTPForSend() unexpected error: %v", err)
	}
	if params.Addr != "smtp.example.com:587" {
		t.Errorf("params.Addr = %q, want %q", params.Addr, "smtp.example.com:587")
	}
	if params.From != "sender@example.com" {
		t.Errorf("params.From = %q, want %q", params.From, "sender@example.com")
	}
	if params.TLS != "starttls" {
		t.Errorf("params.TLS = %q, want %q", params.TLS, "starttls")
	}
	if params.Auth == nil {
		t.Error("params.Auth should be set when username and password are provided")
	}
}

// TestValidateSMTPForSend_EmptyHost verifies that an empty host is rejected.
func TestValidateSMTPForSend_EmptyHost(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{From: "sender@example.com", TLS: "starttls"}
	_, err := validateSMTPForSend(cfg)
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

// TestValidateSMTPForSend_EmptyFrom verifies that an empty From address is rejected.
func TestValidateSMTPForSend_EmptyFrom(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{Host: "smtp.example.com", TLS: "starttls"}
	_, err := validateSMTPForSend(cfg)
	if err == nil {
		t.Fatal("expected error for empty from address")
	}
}

// TestValidateSMTPForSend_InvalidFromAddress verifies that malformed From
// addresses are rejected.
func TestValidateSMTPForSend_InvalidFromAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
	}{
		{name: "not an email", from: "not-an-email"},
		{name: "with display name", from: "Display Name <sender@example.com>"},
		{name: "with embedded newline", from: "sender\r\n@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := smtpConfig{Host: "smtp.example.com", From: tt.from, TLS: "starttls"}
			_, err := validateSMTPForSend(cfg)
			if err == nil {
				t.Errorf("expected error for from %q", tt.from)
			}
		})
	}
}

// TestValidateSMTPForSend_InvalidPort verifies that invalid port values are
// rejected.
func TestValidateSMTPForSend_InvalidPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		port string
	}{
		{name: "zero", port: "0"},
		{name: "too large", port: "65536"},
		{name: "not a number", port: "abc"},
		{name: "negative", port: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := smtpConfig{Host: "smtp.example.com", Port: tt.port, From: "sender@example.com", TLS: "starttls"}
			_, err := validateSMTPForSend(cfg)
			if err == nil {
				t.Errorf("expected error for port %q", tt.port)
			}
		})
	}
}

// TestValidateSMTPForSend_DefaultPort verifies that an empty port defaults to
// 587.
func TestValidateSMTPForSend_DefaultPort(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{Host: "smtp.example.com", Port: "", From: "sender@example.com", TLS: "starttls"}
	params, err := validateSMTPForSend(cfg)
	if err != nil {
		t.Fatalf("validateSMTPForSend() unexpected error: %v", err)
	}
	if params.Addr != "smtp.example.com:587" {
		t.Errorf("params.Addr = %q, want smtp.example.com:587", params.Addr)
	}
}

// TestValidateSMTPForSend_DefaultTLS verifies that an empty TLS field defaults
// to "starttls".
func TestValidateSMTPForSend_DefaultTLS(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{Host: "smtp.example.com", From: "sender@example.com", TLS: ""}
	params, err := validateSMTPForSend(cfg)
	if err != nil {
		t.Fatalf("validateSMTPForSend() unexpected error: %v", err)
	}
	if params.TLS != "starttls" {
		t.Errorf("params.TLS = %q, want starttls", params.TLS)
	}
}

// TestValidateSMTPForSend_InvalidTLSMode verifies that an unrecognized TLS mode
// is rejected.
func TestValidateSMTPForSend_InvalidTLSMode(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{Host: "smtp.example.com", From: "sender@example.com", TLS: "insecure"}
	_, err := validateSMTPForSend(cfg)
	if err == nil {
		t.Fatal("expected error for invalid TLS mode")
	}
}

// TestValidateSMTPForSend_TLSModes verifies that each valid TLS mode is accepted.
func TestValidateSMTPForSend_TLSModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"none", "starttls", "tls"} {
		t.Run("tls="+mode, func(t *testing.T) {
			t.Parallel()
			cfg := smtpConfig{Host: "smtp.example.com", From: "sender@example.com", TLS: mode}
			params, err := validateSMTPForSend(cfg)
			if err != nil {
				t.Errorf("validateSMTPForSend() tls=%q unexpected error: %v", mode, err)
			}
			if params.TLS != mode {
				t.Errorf("params.TLS = %q, want %q", params.TLS, mode)
			}
		})
	}
}

// TestValidateSMTPForSend_UsernameWithoutPassword verifies that providing a
// username without a password is rejected.
func TestValidateSMTPForSend_UsernameWithoutPassword(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{
		Host:     "smtp.example.com",
		From:     "sender@example.com",
		TLS:      "starttls",
		Username: "user@example.com",
		Password: "",
	}
	_, err := validateSMTPForSend(cfg)
	if err == nil {
		t.Fatal("expected error for username without password")
	}
}

// TestValidateSMTPForSend_NoAuth verifies that a config without credentials
// produces nil Auth.
func TestValidateSMTPForSend_NoAuth(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{
		Host: "smtp.example.com",
		From: "sender@example.com",
		TLS:  "starttls",
	}
	params, err := validateSMTPForSend(cfg)
	if err != nil {
		t.Fatalf("validateSMTPForSend() unexpected error: %v", err)
	}
	if params.Auth != nil {
		t.Error("params.Auth should be nil when no credentials are provided")
	}
}

// TestValidateSMTPForSend_AuthWithoutTLSRemoteRejected verifies that
// authenticated SMTP without TLS is rejected for non-loopback hosts.
func TestValidateSMTPForSend_AuthWithoutTLSRemoteRejected(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{
		Host:     "smtp.example.com",
		From:     "sender@example.com",
		TLS:      "none",
		Username: "user",
		Password: "pass",
	}
	_, err := validateSMTPForSend(cfg)
	if err == nil {
		t.Fatal("expected error for authenticated SMTP without TLS on remote host")
	}
}

// TestValidateSMTPForSend_AuthWithoutTLSLoopbackAllowed verifies that
// authenticated SMTP without TLS is allowed for loopback hosts.
func TestValidateSMTPForSend_AuthWithoutTLSLoopbackAllowed(t *testing.T) {
	t.Parallel()

	cfg := smtpConfig{
		Host:     "127.0.0.1",
		From:     "sender@example.com",
		TLS:      "none",
		Username: "user",
		Password: "pass",
	}
	_, err := validateSMTPForSend(cfg)
	if err != nil {
		t.Fatalf("validateSMTPForSend() unexpected error for loopback host: %v", err)
	}
}
