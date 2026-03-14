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

	"github.com/google/uuid"
)

type Payload struct {
	Application string `json:"application"`
	InstallID   string `json:"install_id"`
	Version     string `json:"version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Timestamp   string `json:"timestamp"`
}

func SendBoot(ctx context.Context, version string) {
	// Telemetry is opt-in meaning it is disabled by default unless explicitly enabled
	envTelemetryEnabled, ok := os.LookupEnv("TELEMETRY_ENABLED")
	if ok {
		slog.DebugContext(ctx, "Telemetry environment variable found", slog.String("telemetry_enabled", envTelemetryEnabled))

		if !strings.EqualFold(envTelemetryEnabled, "true") {
			slog.InfoContext(ctx, "Telemetry is disabled via TELEMETRY_ENABLED environment variable")
			return
		}
	} else {
		slog.WarnContext(ctx, "TELEMETRY_ENABLED environment variable not set, telemetry is disabled by default")
		return
	}

	endpoint := os.Getenv("TELEMETRY_ENDPOINT")
	if endpoint == "" {
		slog.DebugContext(ctx, "Telemetry endpoint not set, using default")
		endpoint = "https://telemetry-worker.amalgamated-tools.workers.dev"
	}

	slog.WarnContext(ctx, "NOTICE: This application collects anonymous telemetry data to help improve the product. To disable telemetry, set the environment variable TELEMETRY_ENABLED=false")

	var installIDPath string
	// Determine install ID path: prefer mounted /data folder, fall back to ./data
	if _, err := os.Stat("/data"); err == nil {
		installIDPath = "/data/install_id"
		slog.DebugContext(ctx, "Using mounted /data folder for install ID", slog.String("path", installIDPath))
	} else {
		installIDPath = "./data/install_id"
		slog.DebugContext(ctx, "Using local data folder for install ID", slog.String("path", installIDPath))
	}

	// Only send once per install
	if _, err := os.Stat(installIDPath); err == nil {
		slog.DebugContext(ctx, "Telemetry already sent for this install, skipping")
		return
	}

	slog.DebugContext(ctx, "Install ID not found, sending telemetry data")
	// Create install ID
	id := uuid.New().String()

	payload := Payload{
		Application: "biblioteka",
		InstallID:   id,
		Version:     version,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create telemetry request", slog.Any("error", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send telemetry request", slog.Any("error", err))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.ErrorContext(ctx, "Failed to close telemetry response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Telemetry request failed", slog.Int("status", resp.StatusCode))
		return
	}

	// write out response to log
	slog.DebugContext(ctx, "Telemetry sent successfully")
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read telemetry response", slog.Any("error", err))
		return
	}
	slog.DebugContext(ctx, "Telemetry response", slog.String("body", string(body)))

	err = os.WriteFile(installIDPath, []byte(id), 0644)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to write install ID", slog.Any("error", err))
		return
	}
}
