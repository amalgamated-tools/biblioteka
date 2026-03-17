package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobScanLibrary is the registered name for the single-library scan job.
const JobScanLibrary = "scan:library"

// ScanLibraryPayload is the JSON payload for the scan:library job.
type ScanLibraryPayload struct {
	LibraryID string   `json:"library_id"`
	Paths     []string `json:"paths"`
}

// NewScanLibraryHandler returns a worker.Func that enqueues a scan:path job
// for every path in the given library. It is used when a single library needs
// scanning, e.g. after creation.
func NewScanLibraryHandler(enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p ScanLibraryPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal scan library payload: %w", err)
		}

		if p.LibraryID == "" {
			return fmt.Errorf("scan library payload: library_id is required")
		}

		slog.InfoContext(ctx, "starting library scan", slog.String(otelkeys.LibraryID, p.LibraryID))

		var enqueued int
		for _, path := range p.Paths {
			if _, err := enqueuer.Enqueue(ctx, JobScanPath, ScanPathPayload{Path: path, LibraryID: p.LibraryID, LibraryRoot: path}); err != nil {
				slog.WarnContext(ctx, "failed to enqueue scan:path job",
					slog.String(otelkeys.LibraryID, p.LibraryID),
					slog.String(otelkeys.Path, path),
					slog.Any(otelkeys.Error, err),
				)
				continue
			}
			enqueued++
			slog.InfoContext(ctx, "enqueued path scan",
				slog.String(otelkeys.LibraryID, p.LibraryID),
				slog.String(otelkeys.Path, path),
			)
		}

		slog.InfoContext(ctx, "library scan complete",
			slog.String(otelkeys.LibraryID, p.LibraryID),
			slog.Int(otelkeys.PathsEnqueued, enqueued),
		)
		return nil
	}
}
