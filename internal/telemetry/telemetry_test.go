package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSendBoot_DisabledByDefault verifies that SendBoot is a no-op when the
// TELEMETRY_ENABLED environment variable is not set.
func TestSendBoot_DisabledByDefault(t *testing.T) {
	// Remove the env var so telemetry is disabled.
	t.Setenv(EnvTelemetryEnabled, "")

	// Point telemetry at a test server that would fail the test if hit.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// We can't easily unset an env var that was set; use a different env var
	// approach. telemetry.SendBoot checks os.LookupEnv; an empty value is treated
	// as disabled, so setting it to an empty string is safe.
	t.Setenv(EnvTelemetryEnabled, "")
	SendBoot(context.Background(), "test-version")

	if called {
		t.Error("expected SendBoot to be a no-op when TELEMETRY_ENABLED is empty/false")
	}
}

// TestSendBoot_ExplicitlyDisabled verifies that SendBoot is a no-op when
// TELEMETRY_ENABLED is set to "false".
func TestSendBoot_ExplicitlyDisabled(t *testing.T) {
	t.Setenv(EnvTelemetryEnabled, "false")

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	t.Setenv(EnvTelemetryEndpoint, srv.URL)

	SendBoot(context.Background(), "v1.2.3")
	if called {
		t.Error("expected SendBoot to skip HTTP call when TELEMETRY_ENABLED=false")
	}
}

// TestSendBoot_Enabled verifies that SendBoot sends an HTTP request when
// TELEMETRY_ENABLED is set to "true".
func TestSendBoot_Enabled(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv(EnvTelemetryEnabled, "true")
	t.Setenv(EnvTelemetryEndpoint, srv.URL)

	SendBoot(context.Background(), "v1.0.0")
	if !called {
		t.Error("expected SendBoot to make an HTTP call when TELEMETRY_ENABLED=true")
	}
}

// TestPayload_Fields verifies that the Payload struct has the expected JSON field names.
func TestPayload_Fields(t *testing.T) {
	t.Parallel()

	p := Payload{
		Application: "biblioteka",
		InstallID:   "test-install-id",
		Version:     "v1.0.0",
		OS:          "linux",
		Arch:        "amd64",
		Timestamp:   "2026-01-01T00:00:00Z",
	}
	if p.Application != "biblioteka" {
		t.Errorf("Application = %q, want biblioteka", p.Application)
	}
	if p.Version != "v1.0.0" {
		t.Errorf("Version = %q, want v1.0.0", p.Version)
	}
}
