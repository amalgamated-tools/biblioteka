package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/organize"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
)

// maybeReorganizeFile moves the file into an organized directory structure
// based on the library's organization_type setting. Returns the final file
// path, whether processing should be skipped, and any hard error.
func maybeReorganizeFile(ctx context.Context, database *db.DB, filePath, libraryRoot, author, title, libraryID, organizationType string, lookup bookFileLookupFunc) (string, bool, error) {
	if libraryRoot == "" || author == "" {
		return filePath, false, nil
	}

	if organizationType == "" || organizationType == db.LibraryOrganizationNone {
		return filePath, false, nil
	}

	var newPath string
	var reorgErr error

	switch organizationType {
	case db.LibraryOrganizationBookPerFolder:
		if title == "" {
			return filePath, false, nil
		}
		newPath, reorgErr = organize.ReorganizeFile(filePath, libraryRoot, author, title)
	case db.LibraryOrganizationBookPerFile:
		newPath, reorgErr = organize.ReorganizeFileFlat(filePath, libraryRoot, author)
	default:
		return filePath, false, nil
	}

	if reorgErr != nil {
		slog.WarnContext(ctx, "reorganize file encountered an error",
			slog.String(otelkeys.Path, filePath),
			slog.Any(otelkeys.Error, reorgErr),
		)

		if os.IsNotExist(reorgErr) {
			// If ReorganizeFile returned a newPath and it exists on disk,
			// the copy succeeded but the cleanup remove of the original
			// failed with a not-exist error. Continue from newPath.
			if newPath != "" && newPath != filePath {
				if _, statErr := os.Stat(newPath); statErr == nil {
					slog.InfoContext(ctx, "reorganize partially succeeded, continuing from new path",
						slog.String(otelkeys.From, filePath),
						slog.String(otelkeys.To, newPath),
						slog.Any(otelkeys.Error, reorgErr),
					)
					// Fall through to the success path below.
					reorgErr = nil
				}
			}
			// Source file truly disappeared and no successful copy exists.
			if reorgErr != nil {
				return "", false, fmt.Errorf("source file missing during reorganize: %w", reorgErr)
			}
		}

		// For non-not-exist errors, clean up any orphaned copy and
		// fall back to the original path.
		if reorgErr != nil {
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
	}

	if newPath != filePath {
		slog.InfoContext(ctx, "file reorganized",
			slog.String(otelkeys.From, filePath),
			slog.String(otelkeys.To, newPath),
		)

		// Re-check for duplicates at the new path — another worker may
		// have already indexed the reorganized location.
		if existingBF, err := lookup(ctx, database, newPath); err == nil {
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
		if meta.CoverImageURL != "" {
			v := meta.CoverImageURL
			coverImageURL = &v
		}
		if meta.ISBN != "" {
			if normalizedISBN := metadata.NormalizeISBN(meta.ISBN); normalizedISBN != "" {
				switch len(normalizedISBN) {
				case 10:
					v := normalizedISBN
					isbn10 = &v
				case 13:
					v := normalizedISBN
					isbn13 = &v
				}
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
