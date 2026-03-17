package jobs

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

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

// ScanDirectory walks p.Path recursively and enqueues a process:file job via
// enqueuer for every supported ebook file (.epub, .mobi, .pdf, .azw3) found.
func ScanDirectory(ctx context.Context, enqueuer Enqueuer, p ScanPathPayload) error {
	if p.Path == "" {
		return fmt.Errorf("scan path payload: path is required")
	}

	slog.DebugContext(ctx, "scan:path job received", slog.String(otelkeys.Path, p.Path))

	info, err := os.Stat(p.Path)
	if err != nil {
		return fmt.Errorf("scan path %s: %w", p.Path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scan path %s: not a directory", p.Path)
	}

	slog.InfoContext(ctx, "starting path scan", slog.String(otelkeys.Path, p.Path))

	libraryRootAbs := p.LibraryRoot
	if p.LibraryRoot != "" {
		if lrAbs, err := filepath.Abs(p.LibraryRoot); err == nil {
			libraryRootAbs = lrAbs
		} else {
			libraryRootAbs = filepath.Clean(p.LibraryRoot)
		}
	}

	var found int
	err = filepath.WalkDir(p.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.WarnContext(ctx, "error accessing path",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, err),
			)
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if d.IsDir() {
			slog.DebugContext(ctx, "scan:path visiting directory", slog.String(otelkeys.Path, path))
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		fileType, ok := supportedExtensions[ext]
		if !ok {
			slog.DebugContext(ctx, "scan:path skipping unsupported file",
				slog.String(otelkeys.Path, path),
				slog.String(otelkeys.Ext, ext),
			)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			slog.WarnContext(ctx, "error reading file info",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, err),
			)
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		_, err = enqueuer.Enqueue(ctx, JobProcessFile, ProcessFilePayload{
			Path:        absPath,
			FileName:    filepath.Base(path),
			FileType:    fileType,
			FileSize:    info.Size(),
			LibraryID:   p.LibraryID,
			LibraryRoot: libraryRootAbs,
		})
		if err != nil {
			slog.WarnContext(ctx, "error enqueuing process:file job",
				slog.String(otelkeys.Path, absPath),
				slog.Any(otelkeys.Error, err),
			)
			return nil
		}

		found++
		slog.InfoContext(ctx, "enqueued file for processing",
			slog.String(otelkeys.FileType, fileType),
			slog.String(otelkeys.Path, absPath),
		)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walk path %s: %w", p.Path, err)
	}

	slog.InfoContext(ctx, "path scan complete",
		slog.String(otelkeys.Path, p.Path),
		slog.Int(otelkeys.FilesFound, found),
	)
	return nil
}
