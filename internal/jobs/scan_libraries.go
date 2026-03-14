package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// JobScanLibraries is the registered name for the scan-all-libraries job.
const JobScanLibraries = "scan:libraries"

// JobScanLibrary is the registered name for the single-library scan job.
const JobScanLibrary = "scan:library"

// ScanLibraryPayload is the JSON payload for the scan:library job.
type ScanLibraryPayload struct {
	LibraryID string   `json:"library_id"`
	Paths     []string `json:"paths"`
}

// LibraryLister is the subset of db.DB needed to list libraries.
type LibraryLister interface {
	ListLibraries(ctx context.Context) ([]db.Library, error)
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

		slog.InfoContext(ctx, "starting library scan", slog.String("library_id", p.LibraryID))

		var enqueued int
		for _, path := range p.Paths {
			if _, err := enqueuer.Enqueue(ctx, JobScanPath, ScanPathPayload{Path: path}); err != nil {
				slog.WarnContext(ctx, "failed to enqueue scan:path job",
					slog.String("library_id", p.LibraryID),
					slog.String("path", path),
					slog.Any("error", err),
				)
				continue
			}
			enqueued++
			slog.InfoContext(ctx, "enqueued path scan",
				slog.String("library_id", p.LibraryID),
				slog.String("path", path),
			)
		}

		slog.InfoContext(ctx, "library scan complete",
			slog.String("library_id", p.LibraryID),
			slog.Int("paths_enqueued", enqueued),
		)
		return nil
	}
}

// NewScanLibrariesHandler returns a worker.Func that enqueues a scan:library
// job for each monitored library. It is intended to be run on a schedule to
// keep the library index up to date.
func NewScanLibrariesHandler(lister LibraryLister, enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		slog.InfoContext(ctx, "starting scheduled libraries scan")

		libraries, err := lister.ListLibraries(ctx)
		if err != nil {
			return fmt.Errorf("list libraries: %w", err)
		}

		var enqueued int
		for _, lib := range libraries {
			if !lib.Monitored {
				continue
			}

			var paths []string
			if err := json.Unmarshal([]byte(lib.Paths), &paths); err != nil {
				slog.WarnContext(ctx, "failed to parse library paths",
					slog.String("library_id", lib.ID),
					slog.String("library_name", lib.Name),
					slog.Any("error", err),
				)
				continue
			}
			if len(paths) == 0 {
				slog.WarnContext(ctx, "monitored library has no paths configured; skipping scan",
					slog.String("library_id", lib.ID),
					slog.String("library_name", lib.Name),
				)
				continue
			}

			if _, err := enqueuer.Enqueue(ctx, JobScanLibrary, ScanLibraryPayload{
				LibraryID: lib.ID,
				Paths:     paths,
			}); err != nil {
				slog.WarnContext(ctx, "failed to enqueue scan:library job",
					slog.String("library_id", lib.ID),
					slog.Any("error", err),
				)
				continue
			}
			enqueued++
			slog.InfoContext(ctx, "enqueued library scan",
				slog.String("library_id", lib.ID),
				slog.String("library_name", lib.Name),
			)
		}

		slog.InfoContext(ctx, "scheduled libraries scan complete", slog.Int("libraries_enqueued", enqueued))
		return nil
	}
}
