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

// LibraryLister is the subset of db.DB needed to list libraries.
type LibraryLister interface {
	ListLibraries() ([]db.Library, error)
}

// NewScanLibrariesHandler returns a worker.Func that enqueues a scan:path job
// for every path in each monitored library. It is intended to be run on a
// schedule to keep the library index up to date.
func NewScanLibrariesHandler(lister LibraryLister, enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		slog.Info("starting scheduled libraries scan")

		libraries, err := lister.ListLibraries()
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
				slog.Warn("failed to parse library paths",
					slog.String("library_id", lib.ID),
					slog.String("library_name", lib.Name),
					slog.Any("error", err),
				)
				continue
			}

			for _, path := range paths {
				if _, err := enqueuer.Enqueue(ctx, JobScanPath, ScanPathPayload{Path: path}); err != nil {
					slog.Warn("failed to enqueue scan:path job",
						slog.String("library_id", lib.ID),
						slog.String("path", path),
						slog.Any("error", err),
					)
					continue
				}
				enqueued++
				slog.Info("enqueued path scan",
					slog.String("library_id", lib.ID),
					slog.String("library_name", lib.Name),
					slog.String("path", path),
				)
			}
		}

		slog.Info("scheduled libraries scan complete", slog.Int("paths_enqueued", enqueued))
		return nil
	}
}
