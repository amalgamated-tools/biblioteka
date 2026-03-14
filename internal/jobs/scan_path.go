package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobScanPath is the registered name for the path-scanning job.
const JobScanPath = "scan:path"

// supportedExtensions maps lowercase file extensions to their file type label.
var supportedExtensions = map[string]string{
	".epub": "epub",
	".mobi": "mobi",
	".pdf":  "pdf",
	".azw3": "azw3",
}

// Enqueuer is the subset of worker.Worker needed to enqueue jobs.
type Enqueuer interface {
	Enqueue(ctx context.Context, name string, payload any) (string, error)
}

// ScanPathPayload is the JSON payload for the scan:path job.
type ScanPathPayload struct {
	Path string `json:"path"`
}

// NewScanPathHandler returns a worker.Func that walks the given path and
// enqueues a process:file job for every EPUB, MOBI, PDF, or AZW3 it finds.
func NewScanPathHandler(enqueuer Enqueuer) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p ScanPathPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal scan path payload: %w", err)
		}

		if p.Path == "" {
			return fmt.Errorf("scan path payload: path is required")
		}

		slog.DebugContext(ctx, "scan:path job received", slog.String("path", p.Path))

		if _, err := os.Stat(p.Path); err != nil {
			return fmt.Errorf("scan path %s: %w", p.Path, err)
		}

		slog.InfoContext(ctx, "starting path scan", slog.String("path", p.Path))

		var found int
		err := filepath.WalkDir(p.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.WarnContext(ctx, "error accessing path", slog.String("path", path), slog.Any(otelkeys.Error, err))
				return nil
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			if d.IsDir() {
				slog.DebugContext(ctx, "scan:path skipping directory", slog.String("path", path))
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			fileType, ok := supportedExtensions[ext]
			if !ok {
				slog.DebugContext(ctx, "scan:path skipping unsupported file", slog.String("path", path), slog.String("ext", ext))
				return nil
			}

			info, err := d.Info()
			if err != nil {
				slog.WarnContext(ctx, "error reading file info", slog.String("path", path), slog.Any(otelkeys.Error, err))
				return nil
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			_, err = enqueuer.Enqueue(ctx, JobProcessFile, ProcessFilePayload{
				Path:     absPath,
				FileName: filepath.Base(path),
				FileType: fileType,
				FileSize: info.Size(),
			})
			if err != nil {
				slog.WarnContext(ctx, "error enqueuing process:file job", slog.String("path", absPath), slog.Any(otelkeys.Error, err))
				return nil
			}

			found++
			slog.InfoContext(ctx, "enqueued file for processing",
				slog.String("type", fileType),
				slog.String("path", absPath),
			)

			return nil
		})

		if err != nil {
			return fmt.Errorf("walk path %s: %w", p.Path, err)
		}

		slog.InfoContext(ctx, "path scan complete", slog.String("path", p.Path), slog.Int("files_found", found))
		return nil
	}
}
