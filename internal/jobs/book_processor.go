package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

	// Ensure the source path still exists before proceeding. If a prior attempt
	// reorganized (moved) the file but failed before committing DB rows, the
	// original path will be gone. In that case, try to locate the file at the
	// reorganized target and check whether it was already indexed.
	if _, err := os.Stat(p.Path); err != nil {
		if os.IsNotExist(err) {
			// Check if the file was already indexed at the original path.
			if bf, dbErr := database.GetBookFileByPath(ctx, p.Path); dbErr == nil {
				slog.InfoContext(ctx, "source file missing but already indexed, skipping",
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.BookID, bf.BookID),
				)
				if p.LibraryID != "" {
					_ = database.AddBookToLibrary(ctx, p.LibraryID, bf.BookID)
				}
				return nil
			}

			// Attempt to find the file at the expected reorganized location.
			if p.LibraryRoot != "" {
				pathInfo := pathparser.ParseBookPath(p.Path, p.LibraryRoot)
				if pathInfo.Author != "" && pathInfo.Title != "" {
					candidatePath := filepath.Join(p.LibraryRoot, pathInfo.Author, pathInfo.Title, filepath.Base(p.Path))
					if _, statErr := os.Stat(candidatePath); statErr == nil {
						// Check if the reorganized path is already indexed.
						if bf, dbErr := database.GetBookFileByPath(ctx, candidatePath); dbErr == nil {
							slog.InfoContext(ctx, "reorganized path already indexed, skipping",
								slog.String(otelkeys.Path, candidatePath),
								slog.String(otelkeys.BookID, bf.BookID),
							)
							if p.LibraryID != "" {
								_ = database.AddBookToLibrary(ctx, p.LibraryID, bf.BookID)
							}
							return nil
						}
						// File exists at reorganized location but isn't indexed — update
						// the payload path so processing continues from the new location.
						slog.InfoContext(ctx, "source file moved by prior attempt, continuing from reorganized path",
							slog.String(otelkeys.From, p.Path),
							slog.String(otelkeys.To, candidatePath),
						)
						p.Path = candidatePath
						goto pathResolved
					}
				}
			}

			slog.InfoContext(ctx, "source file no longer exists and could not be located, skipping",
				slog.String(otelkeys.Path, p.Path),
			)
			return nil
		}

		wrappedErr := fmt.Errorf("process book file: stat path %q: %w", p.Path, err)
		slog.ErrorContext(ctx, "book processing failed: error stating source path",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, p.Path),
		)
		return wrappedErr
	}

pathResolved:

	// Check for duplicate: skip full processing if this file path is already indexed,
	// but still ensure the book is linked to the requested library (if any).
	bookFile, err := database.GetBookFileByPath(ctx, p.Path)
	if err == nil {
		slog.InfoContext(ctx, "file already indexed, skipping full processing",
			slog.String(otelkeys.Path, p.Path),
		)

		// Best-effort: if this job was scoped to a specific library, ensure the
		// existing book is associated with that library as well.
		if p.LibraryID != "" {
			if err := database.AddBookToLibrary(ctx, p.LibraryID, bookFile.BookID); err != nil {
				wrappedErr := fmt.Errorf("process book file: add existing book %s to library %s: %w", bookFile.BookID, p.LibraryID, err)
				slog.WarnContext(ctx, "could not associate existing book with library",
					slog.Any(otelkeys.Error, wrappedErr),
					slog.String(otelkeys.Path, p.Path),
				)
			}
		}

		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		wrappedErr := fmt.Errorf("process book file: get existing book file by path %q: %w", p.Path, err)
		slog.ErrorContext(ctx, "book processing failed: error checking for existing file",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, p.Path),
		)
		return wrappedErr
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

	// Reorganize file into Author/Title/ structure if explicitly enabled and we have enough info.
	filePath := p.Path
	if p.LibraryRoot != "" && authorName != "" && title != "" {
		shouldOrganize := false
		setting, settingErr := database.GetSetting(ctx, "organize_files")
		if settingErr != nil {
			slog.WarnContext(ctx, "could not read organize_files setting, skipping reorganization",
				slog.Any(otelkeys.Error, settingErr),
			)
		} else if setting == "true" {
			shouldOrganize = true
		}
		if shouldOrganize {
			newPath, reorgErr := organize.ReorganizeFile(filePath, p.LibraryRoot, authorName, title)
			if reorgErr != nil {
				slog.WarnContext(ctx, "failed to reorganize file, using original path",
					slog.String(otelkeys.Path, filePath),
					slog.Any(otelkeys.Error, reorgErr),
				)
				// If the source file disappeared between the initial stat and
				// this reorganization attempt, abort processing so we do not
				// create a book_files row pointing at a non-existent path.
				if os.IsNotExist(reorgErr) {
					return fmt.Errorf("source file missing during reorganize: %w", reorgErr)
				}
				// If the error came from a failed Remove after a successful copy,
				// an orphaned copy may exist at newPath. Remove it so the next
				// scan does not index it as a separate book.
				if newPath != "" && newPath != filePath {
					if rmErr := os.Remove(newPath); rmErr != nil && !os.IsNotExist(rmErr) {
						slog.WarnContext(ctx, "could not remove orphaned copy after reorganize failure",
							slog.String(otelkeys.Path, newPath),
							slog.Any(otelkeys.Error, rmErr),
						)
					}
				}
			} else if newPath != filePath {
				slog.InfoContext(ctx, "file reorganized",
					slog.String(otelkeys.From, filePath),
					slog.String(otelkeys.To, newPath),
				)
				filePath = newPath

				// Re-check for duplicates at the new path — another worker may
				// have already indexed the reorganized location.
				if existingBF, err := database.GetBookFileByPath(ctx, filePath); err == nil {
					slog.InfoContext(ctx, "reorganized path already indexed, skipping",
						slog.String(otelkeys.Path, filePath),
					)
					if p.LibraryID != "" {
						if linkErr := database.AddBookToLibrary(ctx, p.LibraryID, existingBF.BookID); linkErr != nil {
							slog.WarnContext(ctx, "could not associate existing book with library after reorg",
								slog.String(otelkeys.BookID, existingBF.BookID),
								slog.String(otelkeys.LibraryID, p.LibraryID),
								slog.Any(otelkeys.Error, linkErr),
							)
						}
					}
					return nil
				} else if !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("check duplicate at reorganized path %q: %w", filePath, err)
				}
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
			slog.String(otelkeys.Path, filePath),
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("create book with file for %s: %w", filePath, err)
	}

	// Create an Author record and associate it with the book.
	if authorName != "" {
		author, err := database.FindOrCreateAuthor(ctx, authorName)
		if err != nil {
			slog.WarnContext(ctx, "failed to find or create author",
				slog.String(otelkeys.Path, filePath),
				slog.String(otelkeys.Author, authorName),
				slog.Any(otelkeys.Error, err),
			)
		} else {
			err = database.SetBookAuthors(ctx, book.ID, []string{author.ID})
			if err != nil {
				slog.WarnContext(ctx, "failed to associate author with book for file",
					slog.String(otelkeys.Path, filePath),
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
				slog.String(otelkeys.Path, filePath),
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
		slog.String(otelkeys.Path, filePath),
	)

	return nil
}
