package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// JobProcessFile is the registered name for the file-processing job.
const JobProcessFile = "process:file"

// ProcessFilePayload is the JSON payload for the process:file job.
type ProcessFilePayload struct {
	Path     string `json:"path"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
}

// NewProcessFileHandler returns a worker.Func that extracts metadata for a file
// and then creates a book and book_file record for it. The extracted metadata
// can be used to populate or enrich the book fields (title, authors, etc.).
func NewProcessFileHandler(database *db.DB, extractor *metadata.Extractor) func(ctx context.Context, payload []byte) error {
	if extractor == nil {
		return func(ctx context.Context, payload []byte) error {
			slog.ErrorContext(ctx, "process:file handler misconfigured: metadata extractor is nil")
			return fmt.Errorf("process file handler misconfigured: metadata extractor is nil")
		}
	}

	return processFile(database, extractor)
}

func processFile(database *db.DB, extractor *metadata.Extractor) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p ProcessFilePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal process:file payload", slog.Any(otelkeys.Error, err), slog.Any(otelkeys.Payload, string(payload)))
			return fmt.Errorf("unmarshal process file payload: %w", err)
		}

		if p.Path == "" {
			slog.ErrorContext(ctx, "process:file payload missing required path", slog.Any(otelkeys.Payload, string(payload)))
			return fmt.Errorf("process file payload: path is required")
		}

		if p.FileName == "" {
			slog.ErrorContext(ctx, "process:file payload missing required file_name", slog.Any(otelkeys.Payload, string(payload)))
			return fmt.Errorf("process file payload: file_name is required")
		}

		if p.FileType == "" {
			slog.ErrorContext(ctx, "process:file payload missing required file_type", slog.Any(otelkeys.Payload, string(payload)))
			return fmt.Errorf("process file payload: file_type is required")
		}

		slog.DebugContext(ctx, "process:file job received",
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.FileName, p.FileName),
			slog.String(otelkeys.FileType, p.FileType),
			slog.Int64(otelkeys.FileSize, p.FileSize),
		)
		return ProcessBookFile(ctx, database, extractor, p)
	}
}
