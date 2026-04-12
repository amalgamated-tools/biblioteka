// Package telemetry sends anonymous, opt-in usage pings to the Biblioteka
// telemetry endpoint. The payload contains only non-identifying system
// information (OS, architecture, version) and a randomly generated install ID.
// Telemetry is disabled by default and can be enabled via the
// TELEMETRY_ENABLED=true environment variable.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/google/uuid"
)

// Payload is the JSON body sent to the telemetry endpoint on each boot.
// All fields are non-identifying system information.
type Payload struct {
	Application string `json:"application"`
	InstallID   string `json:"install_id"`
	Version     string `json:"version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Timestamp   string `json:"timestamp"`
}

const (
	// EnvTelemetryEnabled is the environment variable that must be set to
	// "true" to opt in to telemetry. Telemetry is disabled by default.
	EnvTelemetryEnabled = "TELEMETRY_ENABLED"
	// EnvTelemetryEndpoint overrides the default telemetry endpoint URL.
	EnvTelemetryEndpoint = "TELEMETRY_ENDPOINT"
	// DefaultTelemetryEndpoint is the URL used when TELEMETRY_ENDPOINT is not
	// set. It points to the Biblioteka telemetry ingestion endpoint.
	DefaultTelemetryEndpoint = "https://telemetry-worker.amalgamated-tools.workers.dev"
)

// nowRFC3339 returns the current UTC time formatted as RFC 3339.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// SendBoot fires a single telemetry ping if TELEMETRY_ENABLED=true. It is
// called once during server startup and may block the application boot path
// while the request is performed. The request times out after 3 seconds.
func SendBoot(ctx context.Context, version string) {
	// Telemetry is opt-in meaning it is disabled by default unless explicitly enabled
	envTelemetryEnabled, ok := os.LookupEnv(EnvTelemetryEnabled)
	if ok {
		slog.DebugContext(ctx, "telemetry environment variable found", slog.String(otelkeys.TelemetryEnabled, envTelemetryEnabled))

		if !strings.EqualFold(envTelemetryEnabled, "true") {
			slog.InfoContext(ctx, "telemetry is disabled via TELEMETRY_ENABLED environment variable")
			return
		}
	} else {
		slog.WarnContext(ctx, "TELEMETRY_ENABLED environment variable not set, telemetry is disabled by default")
		return
	}

	endpoint := os.Getenv(EnvTelemetryEndpoint)
	if endpoint == "" {
		slog.DebugContext(ctx, "telemetry endpoint not set, using default")
		endpoint = DefaultTelemetryEndpoint
	}

	slog.WarnContext(ctx, "anonymous telemetry is enabled",
		slog.Bool(otelkeys.TelemetryEnabled, true),
		slog.String(otelkeys.TelemetryEndpoint, endpoint),
		slog.String(otelkeys.TelemetryDisableHint, "set TELEMETRY_ENABLED=false to disable"),
	)

	var installIDPath string
	// Determine install ID path: prefer mounted /data folder, fall back to ./data
	if _, err := os.Stat("/data"); err == nil {
		installIDPath = "/data/install_id"
		slog.DebugContext(ctx, "using mounted /data folder for install ID", slog.String(otelkeys.Path, installIDPath))
	} else {
		installIDPath = "./data/install_id"
		slog.DebugContext(ctx, "using local data folder for install ID", slog.String(otelkeys.Path, installIDPath))
	}

	// Only send once per install
	if _, err := os.Stat(installIDPath); err == nil {
		slog.DebugContext(ctx, "telemetry already sent for this install, skipping")
		return
	}

	slog.DebugContext(ctx, "install ID not found, sending telemetry data")
	// Create install ID
	id := uuid.New().String()

	payload := Payload{
		Application: "biblioteka",
		InstallID:   id,
		Version:     version,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Timestamp:   nowRFC3339(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal telemetry payload", slog.Any(otelkeys.Error, err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create telemetry request", slog.Any(otelkeys.Error, err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to send telemetry request", slog.Any(otelkeys.Error, err))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close telemetry response body", slog.Any(otelkeys.Error, err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "telemetry request failed", slog.Int(otelkeys.StatusCode, resp.StatusCode))
		return
	}

	// write out response to log
	slog.DebugContext(ctx, "telemetry sent successfully")
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read telemetry response", slog.Any(otelkeys.Error, err))
		return
	}
	slog.DebugContext(ctx, "telemetry response", slog.String(otelkeys.Body, string(body)))

	err = os.WriteFile(installIDPath, []byte(id), 0o644)
	if err != nil {
		slog.ErrorContext(ctx, "failed to write install ID", slog.Any(otelkeys.Error, err))
		return
	}
}
