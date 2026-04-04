package telemetry

import (
	"context"
	"encoding/json"
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

	t.Setenv(EnvTelemetryEndpoint, srv.URL)

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

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	expectedKeys := []string{"application", "install_id", "version", "os", "arch", "timestamp"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("expected JSON key %q not found in marshaled payload", key)
		}
	}
	if m["application"] != "biblioteka" {
		t.Errorf("application = %v, want biblioteka", m["application"])
	}
	if m["version"] != "v1.0.0" {
		t.Errorf("version = %v, want v1.0.0", m["version"])
	}
}
