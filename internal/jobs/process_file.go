package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
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

// NewProcessFileHandler returns a worker.Func that creates a book and book_file
// record for the given file. This is where future metadata extraction (parsing
// EPUB/MOBI/PDF internals) would be added.
func NewProcessFileHandler(database *db.DB) func(ctx context.Context, payload []byte) error {
	return func(ctx context.Context, payload []byte) error {
		var p ProcessFilePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("unmarshal process file payload: %w", err)
		}

		if p.Path == "" {
			return fmt.Errorf("process file payload: path is required")
		}

		if p.FileName == "" {
			return fmt.Errorf("process file payload: file_name is required")
		}

		if p.FileType == "" {
			return fmt.Errorf("process file payload: file_type is required")
		}

		title := p.FileName
		if ext := filepath.Ext(p.FileName); ext != "" && strings.EqualFold(ext[1:], p.FileType) {
			title = strings.TrimSuffix(p.FileName, ext)
		}

		slog.Info("processing file",
			slog.String("title", title),
			slog.String("type", p.FileType),
			slog.String("path", p.Path),
		)

		book, err := database.CreateBook(title, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if err != nil {
			return fmt.Errorf("create book for %s: %w", p.Path, err)
		}

		_, err = database.CreateBookFile(book.ID, p.FileType, p.FileName, p.FileSize, nil, p.Path)
		if err != nil {
			return fmt.Errorf("create book file for %s: %w", p.Path, err)
		}

		slog.Info("file processed",
			slog.String("title", title),
			slog.String("book_id", book.ID),
			slog.String("path", p.Path),
		)

		return nil
	}
}
