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

		title := p.FileName
		if ext := filepath.Ext(p.FileName); ext != "" && strings.EqualFold(ext[1:], p.FileType) {
			title = strings.TrimSuffix(p.FileName, ext)
		}

		var description, isbn10, isbn13, coverImageURL *string
		var numPages *int

		slog.InfoContext(ctx, "processing file",
			slog.String(otelkeys.Title, title),
			slog.String(otelkeys.FileType, p.FileType),
			slog.String(otelkeys.Path, p.Path),
		)

		// Extract metadata before creating the book record so we can use the
		// extracted fields (now or in the future) to populate or enrich the book.
		// The book ID comes from CreateBook, not from metadata extraction.

		meta, err := extractor.ExtractMetadata(p.Path)
		if err != nil {
			// In environments without ExifTool, metadata extraction is expected to fail
			// for many files. Downgrade those expected errors to DEBUG to avoid log flooding,
			// but keep WARN for unexpected extraction failures.
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "exiftool") &&
				(strings.Contains(errStr, "not found") || strings.Contains(errStr, "not available") || strings.Contains(errStr, "missing")) {
				slog.DebugContext(ctx, "metadata extraction failed due to missing exiftool, continuing with filename-derived metadata",
					slog.String(otelkeys.Path, p.Path),
					slog.Any(otelkeys.Error, err),
					slog.String(otelkeys.Title, title),
				)
			} else {
				slog.WarnContext(ctx, "metadata extraction failed, continuing with filename-derived metadata",
					slog.String(otelkeys.Path, p.Path),
					slog.Any(otelkeys.Error, err),
					slog.String(otelkeys.Title, title),
				)
			}
		} else {
			if meta.Description != "" {
				description = &meta.Description
			}
			if isbn := normalizeISBN(meta.ISBN); isbn != "" {
				// Store the normalized ISBN back into meta so the pointer remains valid.
				meta.ISBN = isbn
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

		book, err := database.CreateBook(
			ctx,
			title,
			description,
			nil,
			isbn10,
			isbn13,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
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

// normalizeISBN strips common prefixes, whitespace, hyphens, and spaces from a
// raw ISBN string, and returns the normalized digits. Returns an empty string
// for empty, unrecognized-length, or sentinel "Not Found" values.
func normalizeISBN(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "urn:isbn:"):
		s = s[len("urn:isbn:"):]
	case strings.HasPrefix(lower, "isbn:"):
		s = s[len("isbn:"):]
	}
	s = strings.TrimSpace(s)
	// Remove hyphens and internal spaces (common ISBN formatting).
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	// Ignore sentinel/non-ISBN values and lengths that don't match ISBN-10 or ISBN-13.
	if strings.EqualFold(s, "not found") || (len(s) != 10 && len(s) != 13) {
		return ""
	}
	return s
}
