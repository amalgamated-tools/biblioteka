package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func ProcessBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, p ProcessFilePayload) error {
	var title string
	var description, isbn10, isbn13, coverImageURL *string
	var numPages *int

	title = p.FileName
	if ext := filepath.Ext(p.FileName); ext != "" && strings.EqualFold(ext[1:], p.FileType) {
		slog.DebugContext(ctx, "filename has expected extension, using it to derive title",
			slog.String(otelkeys.FileName, p.FileName),
			slog.String(otelkeys.FileType, p.FileType),
			slog.String(otelkeys.Path, p.Path),
		)
		title = strings.TrimSuffix(p.FileName, ext)
	}

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
		if errors.Is(err, metadata.ErrExifToolUnavailable) {
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
		slog.InfoContext(ctx, "metadata extracted successfully",
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.Title, title),
			slog.Any(otelkeys.BookMetadata, meta),
		)
		if meta.Description != "" {
			description = &meta.Description
		}
		if meta.ISBN != "" {
			// Normalize ISBN: strip whitespace, known prefixes, and hyphens/spaces.
			isbnRaw := strings.TrimSpace(meta.ISBN)
			if isbnRaw != "" {
				lower := strings.ToLower(isbnRaw)
				switch {
				case strings.HasPrefix(lower, "urn:isbn:"):
					isbnRaw = isbnRaw[len("urn:isbn:"):]
				case strings.HasPrefix(lower, "isbn:"):
					isbnRaw = isbnRaw[len("isbn:"):]
				}
				isbnRaw = strings.TrimSpace(isbnRaw)
				// Remove hyphens and internal spaces (common ISBN formatting).
				isbnRaw = strings.ReplaceAll(isbnRaw, "-", "")
				isbnRaw = strings.ReplaceAll(isbnRaw, " ", "")
				isbnRaw = strings.TrimSpace(isbnRaw)
			}
			// Ignore sentinel/non-ISBN values.
			if isbnRaw != "" && !strings.EqualFold(isbnRaw, "not found") {
				// Store the normalized ISBN back into meta so the pointer remains valid.
				meta.ISBN = isbnRaw
				switch len(meta.ISBN) {
				case 10:
					isbn10 = &meta.ISBN
				case 13:
					isbn13 = &meta.ISBN
				}
			}
		}
		if meta.Title != "" {
			title = meta.Title
		}
		slog.InfoContext(ctx, "metadata extracted",
			slog.String(otelkeys.Title, meta.Title),
			slog.String(otelkeys.Format, meta.Format),
			slog.Any(otelkeys.BookMetadata, meta),
		)
	}

	var publicationDate *string
	var publisher *string
	var language *string
	if meta != nil {
		if meta.PublicationDate != "" {
			v := meta.PublicationDate
			publicationDate = &v
		}
		if meta.Publisher != "" {
			v := meta.Publisher
			publisher = &v
		}
		if meta.Language != "" {
			v := meta.Language
			language = &v
		}
	}

	book, err := database.CreateBook(
		ctx,
		title,
		description,
		nil,
		isbn10,
		isbn13,
		publicationDate,
		publisher,
		language,
		nil,
		nil,
		nil,
		numPages,
		coverImageURL,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create book record for file",
			slog.String(otelkeys.Path, p.Path),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create book for %s: %w", p.Path, err)
	}

	_, err = database.CreateBookFile(ctx, book.ID, p.FileType, p.FileName, p.FileSize, nil, p.Path)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create book file for file",
			slog.String(otelkeys.Path, p.Path),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create book file for %s: %w", p.Path, err)
	}

	// create an Author record if metadata extraction found an author and associate it with the book
	if meta != nil && meta.Author != "" {
		author, err := database.GetAuthorByName(ctx, meta.Author)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				slog.ErrorContext(ctx, "failed to get or create author record for file",
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.Author, meta.Author),
					slog.Any(otelkeys.Error, err),
				)
				return fmt.Errorf("get or create author for %s: %w", p.Path, err)
			}
		}
		if author == nil {
			author, err = database.CreateAuthor(ctx, meta.Author, nil, nil, nil, nil)
			if err != nil {
				slog.ErrorContext(ctx, "failed to create author record for file",
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.Author, meta.Author),
					slog.Any(otelkeys.Error, err),
				)
				return fmt.Errorf("create author for %s: %w", p.Path, err)
			}
		}
		err = database.SetBookAuthors(ctx, book.ID, []string{author.ID})
		if err != nil {
			slog.ErrorContext(ctx, "failed to associate author with book for file",
				slog.String(otelkeys.Path, p.Path),
				slog.String(otelkeys.Author, meta.Author),
				slog.Any(otelkeys.Error, err),
			)
			return fmt.Errorf("associate author with book for %s: %w", p.Path, err)
		}
	}

	var format string
	if meta != nil {
		format = meta.Format
	}

	slog.InfoContext(ctx, "file processed",
		slog.String(otelkeys.BookID, book.ID),
		slog.String(otelkeys.Title, title),
		slog.String(otelkeys.FileType, p.FileType),
		slog.Int64(otelkeys.FileSize, p.FileSize),
		slog.Any(otelkeys.BookMetadata, meta),
		slog.String(otelkeys.Format, format),
		slog.String(otelkeys.Path, p.Path),
	)

	return nil
}
