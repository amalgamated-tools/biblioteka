package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobScanLibraries is the registered name for the scan-all-libraries job.
const JobScanLibraries = "scan:libraries"

// LibraryLister is the subset of db.DB needed to list libraries.
type LibraryLister interface {
	ListLibraries(ctx context.Context) ([]db.Library, error)
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
					slog.String(otelkeys.LibraryID, lib.ID),
					slog.String(otelkeys.LibraryName, lib.Name),
					slog.Any(otelkeys.Error, err),
				)
				continue
			}
			if len(paths) == 0 {
				slog.WarnContext(ctx, "monitored library has no paths configured; skipping scan",
					slog.String(otelkeys.LibraryID, lib.ID),
					slog.String(otelkeys.LibraryName, lib.Name),
				)
				continue
			}

			if _, err := enqueuer.Enqueue(ctx, JobScanLibrary, ScanLibraryPayload{
				LibraryID: lib.ID,
				Paths:     paths,
			}); err != nil {
				slog.WarnContext(ctx, "failed to enqueue scan:library job",
					slog.String(otelkeys.LibraryID, lib.ID),
					slog.Any(otelkeys.Error, err),
				)
				continue
			}
			enqueued++
			slog.InfoContext(ctx, "enqueued library scan",
				slog.String(otelkeys.LibraryID, lib.ID),
				slog.String(otelkeys.LibraryName, lib.Name),
			)
		}

		slog.InfoContext(ctx, "scheduled libraries scan complete", slog.Int(otelkeys.LibrariesEnqueued, enqueued))
		return nil
	}
}
