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

// ProcessBookFile orchestrates the complete processing of a single book file:
// validation, deduplication, metadata extraction, file reorganization, and
// database record creation.
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

	if err := validatePayload(ctx, p); err != nil {
		return err
	}

	resolvedPath, skip, err := resolveSourcePath(ctx, database, p)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}
	p.Path = resolvedPath

	alreadyIndexed, err := checkDuplicate(ctx, database, p.Path, p.LibraryID)
	if err != nil {
		return err
	}
	if alreadyIndexed {
		return nil
	}

	title := deriveTitle(ctx, p.FileName, p.FileType, p.Path)

	slog.InfoContext(ctx, "processing file",
		slog.String(otelkeys.Title, title),
		slog.String(otelkeys.FileType, p.FileType),
		slog.String(otelkeys.Path, p.Path),
	)

	meta := extractBookMetadata(ctx, extractor, p.Path, title)

	var pathInfo pathparser.PathInfo
	if p.LibraryRoot != "" {
		pathInfo = pathparser.ParseBookPath(p.Path, p.LibraryRoot)
	}

	authorName, title := resolveAuthorAndTitle(meta, pathInfo, title)

	filePath, skip, err := maybeReorganizeFile(ctx, database, p.Path, p.LibraryRoot, authorName, title, p.LibraryID)
	if err != nil {
		return err
	}
	if skip {
		return nil
	}

	book, err := createBookRecord(ctx, database, title, meta, p, filePath)
	if err != nil {
		return err
	}

	linkBookAssociations(ctx, database, book.ID, authorName, p.LibraryID, pathInfo, filePath)

	var format string
	if meta != nil {
		format = meta.Format

		// Log full metadata only at DEBUG level to avoid bloating INFO logs.
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

// validatePayload checks that required payload string fields are non-empty.
func validatePayload(ctx context.Context, p ProcessFilePayload) error {
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

	return nil
}

// resolveSourcePath ensures the source file exists. If it is missing, attempts
// to recover from a prior processing attempt that moved the file but failed to
// commit DB records. Returns the resolved path, whether processing should be
// skipped, and any hard error.
func resolveSourcePath(ctx context.Context, database *db.DB, p ProcessFilePayload) (string, bool, error) {
	_, statErr := os.Stat(p.Path)
	if statErr == nil {
		return p.Path, false, nil
	}
	if !os.IsNotExist(statErr) {
		wrappedErr := fmt.Errorf("process book file: stat path %q: %w", p.Path, statErr)
		slog.ErrorContext(ctx, "book processing failed: error stating source path",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, p.Path),
		)
		return "", false, wrappedErr
	}

	// File does not exist — check if it was already indexed at the original path.
	bf, dbErr := database.GetBookFileByPath(ctx, p.Path)
	if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) {
		wrappedErr := fmt.Errorf("process book file: get book file by path %q: %w", p.Path, dbErr)
		slog.ErrorContext(ctx, "book processing failed: error looking up book file by path",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, p.Path),
		)
		return "", false, wrappedErr
	}
	if dbErr == nil {
		slog.InfoContext(ctx, "source file missing but already indexed, skipping",
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.BookID, bf.BookID),
		)
		if p.LibraryID != "" {
			if err := database.AddBookToLibrary(ctx, p.LibraryID, bf.BookID); err != nil {
				slog.WarnContext(ctx, "failed to add already-indexed book to library",
					slog.Any(otelkeys.Error, err),
					slog.String(otelkeys.Path, p.Path),
					slog.String(otelkeys.BookID, bf.BookID),
				)
			}
		}
		return "", true, nil
	}

	// Attempt to find the file at the expected reorganized location.
	if p.LibraryRoot != "" {
		pathInfo := pathparser.ParseBookPath(p.Path, p.LibraryRoot)
		if pathInfo.Author != "" && pathInfo.Title != "" {
			candidatePath := organize.TargetPath(p.Path, p.LibraryRoot, pathInfo.Author, pathInfo.Title)
			if candidatePath != "" {
				if _, candidateStatErr := os.Stat(candidatePath); candidateStatErr == nil {
					// Check if the reorganized path is already indexed.
					bf, dbErr := database.GetBookFileByPath(ctx, candidatePath)
					if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) {
						wrappedErr := fmt.Errorf("process book file: get book file by path %q: %w", candidatePath, dbErr)
						slog.ErrorContext(ctx, "book processing failed: error looking up reorganized book file by path",
							slog.Any(otelkeys.Error, wrappedErr),
							slog.String(otelkeys.Path, candidatePath),
						)
						return "", false, wrappedErr
					}
					if dbErr == nil {
						slog.InfoContext(ctx, "reorganized path already indexed, skipping",
							slog.String(otelkeys.Path, candidatePath),
							slog.String(otelkeys.BookID, bf.BookID),
						)
						if p.LibraryID != "" {
							if err := database.AddBookToLibrary(ctx, p.LibraryID, bf.BookID); err != nil {
								slog.WarnContext(ctx, "failed to add reorganized already-indexed book to library",
									slog.Any(otelkeys.Error, err),
									slog.String(otelkeys.Path, candidatePath),
									slog.String(otelkeys.BookID, bf.BookID),
								)
							}
						}
						return "", true, nil
					}
					// File exists at reorganized location but isn't indexed.
					slog.InfoContext(ctx, "source file moved by prior attempt, continuing from reorganized path",
						slog.String(otelkeys.From, p.Path),
						slog.String(otelkeys.To, candidatePath),
					)
					return candidatePath, false, nil
				}
			}
		}
	}

	slog.InfoContext(ctx, "source file no longer exists and could not be located, skipping",
		slog.String(otelkeys.Path, p.Path),
	)
	return "", true, nil
}

// checkDuplicate returns true if the file at the given path is already indexed.
// When already indexed, it best-effort links the existing book to libraryID.
func checkDuplicate(ctx context.Context, database *db.DB, path, libraryID string) (bool, error) {
	bookFile, err := database.GetBookFileByPath(ctx, path)
	if err == nil {
		slog.InfoContext(ctx, "file already indexed, skipping full processing",
			slog.String(otelkeys.Path, path),
		)

		// Best-effort: if this job was scoped to a specific library, ensure the
		// existing book is associated with that library as well.
		if libraryID != "" {
			if err := database.AddBookToLibrary(ctx, libraryID, bookFile.BookID); err != nil {
				wrappedErr := fmt.Errorf("process book file: add existing book %s to library %s: %w", bookFile.BookID, libraryID, err)
				slog.WarnContext(ctx, "could not associate existing book with library",
					slog.Any(otelkeys.Error, wrappedErr),
					slog.String(otelkeys.Path, path),
				)
			}
		}

		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		wrappedErr := fmt.Errorf("process book file: get existing book file by path %q: %w", path, err)
		slog.ErrorContext(ctx, "book processing failed: error checking for existing file",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, path),
		)
		return false, wrappedErr
	}

	return false, nil
}

// deriveTitle returns an initial book title from the file name, stripping the
// extension when it matches the declared file type.
func deriveTitle(ctx context.Context, fileName, fileType, path string) string {
	title := fileName
	if ext := filepath.Ext(fileName); ext != "" && strings.EqualFold(ext[1:], fileType) {
		slog.DebugContext(ctx, "filename has expected extension, using it to derive title",
			slog.String(otelkeys.FileName, fileName),
			slog.String(otelkeys.FileType, fileType),
			slog.String(otelkeys.Path, path),
		)
		title = strings.TrimSuffix(fileName, ext)
	}
	return title
}

// extractBookMetadata attempts to extract metadata from a book file.
// Returns nil when extraction fails; errors are logged, not propagated.
func extractBookMetadata(ctx context.Context, extractor *metadata.Extractor, path, initialTitle string) *metadata.BookMetadata {
	meta, err := extractor.ExtractMetadata(ctx, path)
	if err != nil {
		// In environments without ExifTool, metadata extraction is expected to fail
		// for many files. Downgrade those expected errors to DEBUG to avoid log flooding,
		// but keep WARN for unexpected extraction failures.
		if errors.Is(err, metadata.ErrExifToolUnavailable) {
			slog.DebugContext(ctx, "metadata extraction failed due to missing exiftool, continuing with filename-derived metadata",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, err),
				slog.String(otelkeys.Title, initialTitle),
			)
		} else {
			slog.WarnContext(ctx, "metadata extraction failed, continuing with filename-derived metadata",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, err),
				slog.String(otelkeys.Title, initialTitle),
			)
		}
		return nil
	}

	slog.DebugContext(ctx, "metadata extracted successfully",
		slog.String(otelkeys.Path, path),
		slog.String(otelkeys.Title, initialTitle),
	)

	// Normalize ISBN in-place for downstream consumers.
	if meta.ISBN != "" {
		if normalized := metadata.NormalizeISBN(meta.ISBN); normalized != "" {
			meta.ISBN = normalized
		}
	}

	slog.DebugContext(ctx, "metadata extracted",
		slog.String(otelkeys.Title, meta.Title),
		slog.String(otelkeys.Format, meta.Format),
		slog.Any(otelkeys.BookMetadata, meta),
	)

	return meta
}

// resolveAuthorAndTitle merges metadata-derived and path-derived author and
// title values into final resolved values.
func resolveAuthorAndTitle(meta *metadata.BookMetadata, pathInfo pathparser.PathInfo, currentTitle string) (string, string) {
	title := currentTitle

	// Override title with metadata if available.
	if meta != nil && meta.Title != "" {
		title = meta.Title
	}

	// Fall back to path-derived title if metadata had none.
	if (meta == nil || meta.Title == "") && pathInfo.Title != "" {
		title = pathInfo.Title
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

	return authorName, title
}

// maybeReorganizeFile moves the file into an Author/Title/ directory structure
// when the organize_files setting is enabled. Returns the final file path,
// whether processing should be skipped, and any hard error.
func maybeReorganizeFile(ctx context.Context, database *db.DB, filePath, libraryRoot, author, title, libraryID string) (string, bool, error) {
	if libraryRoot == "" || author == "" || title == "" {
		return filePath, false, nil
	}

	shouldOrganize := false
	setting, settingErr := database.GetSetting(ctx, "organize_files")
	if settingErr != nil {
		if !errors.Is(settingErr, sql.ErrNoRows) {
			slog.WarnContext(ctx, "could not read organize_files setting, skipping reorganization",
				slog.Any(otelkeys.Error, settingErr),
			)
		}
	} else if setting == "true" {
		shouldOrganize = true
	}

	if !shouldOrganize {
		return filePath, false, nil
	}

	newPath, reorgErr := organize.ReorganizeFile(filePath, libraryRoot, author, title)
	if reorgErr != nil {
		slog.WarnContext(ctx, "failed to reorganize file, using original path",
			slog.String(otelkeys.Path, filePath),
			slog.Any(otelkeys.Error, reorgErr),
		)
		// If the source file disappeared between the initial stat and
		// this reorganization attempt, abort processing so we do not
		// create a book_files row pointing at a non-existent path.
		if os.IsNotExist(reorgErr) {
			return "", false, fmt.Errorf("source file missing during reorganize: %w", reorgErr)
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
		return filePath, false, nil
	}

	if newPath != filePath {
		slog.InfoContext(ctx, "file reorganized",
			slog.String(otelkeys.From, filePath),
			slog.String(otelkeys.To, newPath),
		)

		// Re-check for duplicates at the new path — another worker may
		// have already indexed the reorganized location.
		if existingBF, err := database.GetBookFileByPath(ctx, newPath); err == nil {
			slog.InfoContext(ctx, "reorganized path already indexed, skipping",
				slog.String(otelkeys.Path, newPath),
			)
			if libraryID != "" {
				if linkErr := database.AddBookToLibrary(ctx, libraryID, existingBF.BookID); linkErr != nil {
					slog.WarnContext(ctx, "could not associate existing book with library after reorg",
						slog.String(otelkeys.BookID, existingBF.BookID),
						slog.String(otelkeys.LibraryID, libraryID),
						slog.Any(otelkeys.Error, linkErr),
					)
				}
			}
			return "", true, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return "", false, fmt.Errorf("check duplicate at reorganized path %q: %w", newPath, err)
		}

		return newPath, false, nil
	}

	return filePath, false, nil
}

// createBookRecord builds a book and book_file record from extracted metadata
// and payload fields.
func createBookRecord(ctx context.Context, database *db.DB, title string, meta *metadata.BookMetadata, p ProcessFilePayload, filePath string) (*db.Book, error) {
	var description, isbn10, isbn13, coverImageURL *string
	var numPages *int
	var publicationDate, publisher, language *string

	if meta != nil {
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
		return nil, fmt.Errorf("create book with file for %s: %w", filePath, err)
	}

	return book, nil
}

// linkBookAssociations creates author, series, and library associations for a
// newly created book. All operations are best-effort with WARN-level logging
// on failure.
func linkBookAssociations(ctx context.Context, database *db.DB, bookID, authorName, libraryID string, pathInfo pathparser.PathInfo, filePath string) {
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
			if err := database.SetBookAuthors(ctx, bookID, []string{author.ID}); err != nil {
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
			if err := database.SetBookSeries(ctx, bookID, []db.BookSeriesInput{
				{SeriesID: series.ID, Position: pathInfo.SeriesPosition},
			}); err != nil {
				slog.WarnContext(ctx, "failed to set book series",
					slog.String(otelkeys.BookID, bookID),
					slog.Any(otelkeys.Error, err),
				)
			}
		}
	}

	// Link book to library.
	if libraryID != "" {
		if err := database.AddBookToLibrary(ctx, libraryID, bookID); err != nil {
			slog.WarnContext(ctx, "failed to add book to library",
				slog.String(otelkeys.BookID, bookID),
				slog.String(otelkeys.LibraryID, libraryID),
				slog.Any(otelkeys.Error, err),
			)
		}
	}
}
