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
	"github.com/amalgamated-tools/biblioteka/internal/organize"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
)

func ProcessBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, p ProcessFilePayload) error {
	if database == nil {
		err := fmt.Errorf("process book file: database is nil")
		slog.ErrorContext(ctx, "book processing failed: database is nil",
			slog.Any(otelkeys.Error, err),
		)
		return err
	}

	if extractor == nil {
		err := fmt.Errorf("process book file: extractor is nil")
		slog.ErrorContext(ctx, "book processing failed: extractor is nil",
			slog.Any(otelkeys.Error, err),
		)
		return err
	}

	if strings.TrimSpace(p.Path) == "" {
		err := fmt.Errorf("process book file: payload path is empty")
		slog.ErrorContext(ctx, "book processing failed: empty path in payload",
			slog.Any(otelkeys.Error, err),
		)
		return err
	}

	if strings.TrimSpace(p.FileName) == "" {
		err := fmt.Errorf("process book file: payload file name is empty")
		slog.ErrorContext(ctx, "book processing failed: empty file name in payload",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Path, p.Path),
		)
		return err
	}

	if strings.TrimSpace(p.FileType) == "" {
		err := fmt.Errorf("process book file: payload file type is empty")
		slog.ErrorContext(ctx, "book processing failed: empty file type in payload",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.FileName, p.FileName),
		)
		return err
	}

	// Check for duplicate: skip if this file path is already indexed.
	if _, err := database.GetBookFileByPath(ctx, p.Path); err == nil {
		slog.InfoContext(ctx, "file already indexed, skipping",
			slog.String(otelkeys.Path, p.Path),
		)
		return nil
	}

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
	meta, err := extractor.ExtractMetadata(ctx, p.Path)
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
		slog.DebugContext(ctx, "metadata extracted successfully",
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.Title, title),
		)
		if meta.Description != "" {
			description = &meta.Description
		}
		if meta.ISBN != "" {
			normalized := metadata.NormalizeISBN(meta.ISBN)
			if normalized != "" {
				meta.ISBN = normalized
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
		slog.DebugContext(ctx, "metadata extracted",
			slog.String(otelkeys.Title, meta.Title),
			slog.String(otelkeys.Format, meta.Format),
			slog.Any(otelkeys.BookMetadata, meta),
		)
	}

	// Parse directory structure for author, title, series info.
	var pathInfo pathparser.PathInfo
	if p.LibraryRoot != "" {
		pathInfo = pathparser.ParseBookPath(p.Path, p.LibraryRoot)
	}

	// Resolve author: prefer embedded metadata, fall back to path-derived.
	var metaAuthor string
	if meta != nil {
		metaAuthor = strings.TrimSpace(meta.Author)
	}
	authorName := metaAuthor
	if authorName == "" || authorName == "Unknown" {
		authorName = pathInfo.Author
	}

	// Resolve title: fall back to path-derived title if metadata had none.
	if (meta == nil || meta.Title == "") && pathInfo.Title != "" {
		title = pathInfo.Title
	}

	// Reorganize file into Author/Title/ structure if enabled and we have enough info.
	filePath := p.Path
	if p.LibraryRoot != "" && authorName != "" && title != "" {
		shouldOrganize := true
		setting, settingErr := database.GetSetting(ctx, "organize_files")
		if settingErr == nil && setting == "false" {
			shouldOrganize = false
		}
		if shouldOrganize {
			newPath, reorgErr := organize.ReorganizeFile(filePath, p.LibraryRoot, authorName, title)
			if reorgErr != nil {
				slog.WarnContext(ctx, "failed to reorganize file, using original path",
					slog.String(otelkeys.Path, filePath),
					slog.Any(otelkeys.Error, reorgErr),
				)
			} else if newPath != filePath {
				slog.InfoContext(ctx, "file reorganized",
					slog.String(otelkeys.From, filePath),
					slog.String(otelkeys.To, newPath),
				)
				filePath = newPath
			}
		}
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

	book, _, err := database.CreateBookWithFile(
		ctx,
		title,
		description,
		nil,
		isbn10,
		isbn13,
		nil,
		nil,
		nil,
		publicationDate,
		publisher,
		language,
		numPages,
		coverImageURL,
		p.FileType,
		p.FileName,
		p.FileSize,
		nil,
		filePath,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create book and file records",
			slog.String(otelkeys.Path, p.Path),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create book with file for %s: %w", p.Path, err)
	}

	// Create an Author record and associate it with the book.
	if authorName != "" {
		skipAuthorAssociation := false
		author, err := database.GetAuthorByName(ctx, authorName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Author does not exist yet; we'll create it below.
			} else {
				slog.WarnContext(ctx, "failed to get or create author record for file; skipping author association",
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.Author, authorName),
					slog.Any(otelkeys.Error, err),
				)
				skipAuthorAssociation = true
			}
		}
		if !skipAuthorAssociation {
			if author == nil {
				author, err = database.CreateAuthor(ctx, authorName, nil, nil, nil, nil)
				if err != nil {
					if errors.Is(err, db.ErrAuthorNameExists) {
						author, err = database.GetAuthorByName(ctx, authorName)
						if err != nil {
							slog.ErrorContext(ctx, "failed to load existing author after concurrent create",
								slog.String(otelkeys.Path, p.Path),
								slog.String(otelkeys.Author, authorName),
								slog.Any(otelkeys.Error, err),
							)
							return fmt.Errorf("get existing author after conflict for %s: %w", p.Path, err)
						}
						if author == nil {
							return fmt.Errorf("author %q exists but could not be loaded for %s", authorName, p.Path)
						}
					} else {
						slog.ErrorContext(ctx, "failed to create author record for file",
							slog.String(otelkeys.Path, p.Path),
							slog.String(otelkeys.Author, authorName),
							slog.Any(otelkeys.Error, err),
						)
						return fmt.Errorf("create author for %s: %w", p.Path, err)
					}
				}
			}
			err = database.SetBookAuthors(ctx, book.ID, []string{author.ID})
			if err != nil {
				slog.WarnContext(ctx, "failed to associate author with book for file",
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.Author, authorName),
					slog.Any(otelkeys.Error, err),
				)
			}
		}
	}

	// Link series to book.
	if pathInfo.SeriesName != "" {
		series, seriesErr := database.FindOrCreateSeries(ctx, pathInfo.SeriesName)
		if seriesErr != nil {
			slog.WarnContext(ctx, "failed to find or create series",
				slog.String(otelkeys.Path, p.Path),
				slog.Any(otelkeys.Error, seriesErr),
			)
		} else {
			if err := database.SetBookSeries(ctx, book.ID, []db.BookSeriesInput{
				{SeriesID: series.ID, Position: pathInfo.SeriesPosition},
			}); err != nil {
				slog.WarnContext(ctx, "failed to set book series",
					slog.String(otelkeys.BookID, book.ID),
					slog.Any(otelkeys.Error, err),
				)
			}
		}
	}

	// Link book to library.
	if p.LibraryID != "" {
		if err := database.AddBookToLibrary(ctx, p.LibraryID, book.ID); err != nil {
			slog.WarnContext(ctx, "failed to add book to library",
				slog.String(otelkeys.BookID, book.ID),
				slog.String(otelkeys.LibraryID, p.LibraryID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	var format string
	if meta != nil {
		format = meta.Format
	}

	// Log full metadata only at DEBUG level to avoid bloating INFO logs.
	if meta != nil {
		slog.DebugContext(ctx, "book metadata extracted",
			slog.String(otelkeys.BookID, book.ID),
			slog.Any(otelkeys.BookMetadata, meta),
		)
	}

	slog.InfoContext(ctx, "file processed",
		slog.String(otelkeys.BookID, book.ID),
		slog.String(otelkeys.Title, title),
		slog.String(otelkeys.FileType, p.FileType),
		slog.Int64(otelkeys.FileSize, p.FileSize),
		slog.String(otelkeys.Format, format),
		slog.String(otelkeys.Path, p.Path),
	)

	return nil
}
