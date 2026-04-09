package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobScanWatchFolder is the registered name for the watch-folder scan job.
const JobScanWatchFolder = "scan:watch-folder"

const (
	settingWatchFolderPath      = "watch_folder_path"
	settingWatchFolderLibraryID = "watch_folder_library_id"
)

// SettingGetter is the subset of db.DB needed to read settings.
type SettingGetter interface {
	GetSetting(ctx context.Context, key string) (string, error)
}

// NewScanWatchFolderHandler returns a worker.Func that reads the watch folder
// settings and, if configured, scans the watch folder for new book files.
// Files found are enqueued as process:file jobs associated with the configured
// target library.
func NewScanWatchFolderHandler(settings SettingGetter, enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, _ []byte) error {
		slog.DebugContext(ctx, "starting watch folder scan")

		path, err := settings.GetSetting(ctx, settingWatchFolderPath)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				slog.DebugContext(ctx, "watch folder not configured, skipping scan")
				return nil
			}
			return err
		}
		if path == "" {
			slog.DebugContext(ctx, "watch folder path is empty, skipping scan")
			return nil
		}

		libraryID, err := settings.GetSetting(ctx, settingWatchFolderLibraryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				slog.WarnContext(ctx, "watch folder library ID not configured, skipping scan")
				return nil
			}
			return err
		}
		if libraryID == "" {
			slog.WarnContext(ctx, "watch folder library ID is empty, skipping scan")
			return nil
		}

		slog.InfoContext(ctx, "scanning watch folder",
			slog.String(otelkeys.WatchFolderPath, path),
			slog.String(otelkeys.LibraryID, libraryID),
		)

		// Reuse the existing ScanDirectory pipeline. The watch folder path is
		// used as the scan root. The LibraryRoot is left empty so that files
		// stay in the watch folder (unless the library has an organization type,
		// which will cause ProcessBookFile to reorganize files into the
		// library's first configured path).
		return ScanDirectory(ctx, enqueuer, ScanPathPayload{
			Path:      path,
			LibraryID: libraryID,
		})
	}
}
