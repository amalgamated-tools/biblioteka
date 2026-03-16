package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

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

		slog.DebugContext(ctx, "process:file job received",
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.FileName, p.FileName),
			slog.String(otelkeys.FileType, p.FileType),
			slog.Int64(otelkeys.FileSize, p.FileSize),
		)
		var title string
		var description, asin, isbn10, isbn13, goodreadsID, hardcoverID, googleBooksID, publicationDate, publisher, language, coverImageURL *string
		var numPages *int = nil

		title = p.FileName
		if ext := filepath.Ext(p.FileName); ext != "" && strings.EqualFold(ext[1:], p.FileType) {
			title = strings.TrimSuffix(p.FileName, ext)
		}

		slog.InfoContext(ctx, "processing file",
			slog.String(otelkeys.Title, title),
			slog.String(otelkeys.Type, p.FileType),
			slog.String(otelkeys.Path, p.Path),
		)

		// Extract metadata before creating the book record so we can use the
		// extracted fields (now or in the future) to populate or enrich the book.
		// The book ID comes from CreateBook, not from metadata extraction.
		if extractor != nil {
			meta, err := extractor.ExtractMetadata(p.Path)
			if err != nil {
				slog.WarnContext(ctx, "metadata extraction failed, continuing with filename-derived metadata",
					slog.String(otelkeys.Path, p.Path),
					slog.Any(otelkeys.Error, err),
					slog.String(otelkeys.Title, title),
				)
			} else {
				if meta.Description != "" {
					description = &meta.Description
				}
				if meta.ISBN != "" {
					switch len(meta.ISBN) {
					case 10:
						isbn10 = &meta.ISBN
					case 13:
						isbn13 = &meta.ISBN
					}
				}
			if meta.Title != "" {
				title = meta.Title
			}
				slog.DebugContext(ctx, "metadata extracted",
					slog.String(otelkeys.Title, meta.Title),
					slog.String(otelkeys.Format, meta.Format),
				)
			}
		}
		book, err := database.CreateBook(
			ctx,
			title,
			description,
			asin,
			isbn10,
			isbn13,
			goodreadsID,
			hardcoverID,
			googleBooksID,
			publicationDate,
			publisher,
			language,
			numPages,
			coverImageURL,
		)
		if err != nil {
			return fmt.Errorf("create book for %s: %w", p.Path, err)
		}

		_, err = database.CreateBookFile(ctx, book.ID, p.FileType, p.FileName, p.FileSize, nil, p.Path)
		if err != nil {
			return fmt.Errorf("create book file for %s: %w", p.Path, err)
		}

		slog.InfoContext(ctx, "file processed",
			slog.String(otelkeys.BookID, book.ID),
			slog.String(otelkeys.Path, p.Path),
		)

		return nil
	}
}
