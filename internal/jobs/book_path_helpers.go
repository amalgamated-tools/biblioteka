package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/organize"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
)

// bookFileLookupFunc looks up a book file record by its path.
type bookFileLookupFunc func(ctx context.Context, database *db.DB, path string) (*db.BookFile, error)

// defaultBookFileLookup is the production implementation that queries the database.
func defaultBookFileLookup(ctx context.Context, database *db.DB, path string) (*db.BookFile, error) {
	return database.GetBookFileByPath(ctx, path)
}

func reorganizedCandidatePaths(ctx context.Context, p ProcessFilePayload, pathInfo pathparser.PathInfo, organizationType string) []string {
	candidates := make([]string, 0, 2)
	addCandidate := func(path string) {
		if path == "" {
			return
		}
		if slices.Contains(candidates, path) {
			return
		}
		candidates = append(candidates, path)
	}

	switch organizationType {
	case db.LibraryOrganizationBookPerFolder:
		if pathInfo.Author != "" && pathInfo.Title != "" {
			addCandidate(organize.TargetPath(ctx, p.Path, p.LibraryRoot, pathInfo.Author, pathInfo.Title))
		}
	case db.LibraryOrganizationBookPerFile:
		if pathInfo.Author != "" {
			addCandidate(organize.TargetPathFlat(ctx, p.Path, p.LibraryRoot, pathInfo.Author))
		}
	}

	return candidates
}

// validatePayload checks that required payload string fields are non-empty.
func validatePayload(ctx context.Context, p ProcessFilePayload) error {
	if strings.TrimSpace(p.Path) == "" {
		err := fmt.Errorf("process book file: payload path is empty")
		slog.ErrorContext(ctx, "book processing failed: empty path in payload",
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("invalid payload: %w", err)
	}

	if strings.TrimSpace(p.FileName) == "" {
		err := fmt.Errorf("process book file: payload file name is empty")
		slog.ErrorContext(ctx, "book processing failed: empty file name in payload",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Path, p.Path),
		)
		return fmt.Errorf("invalid payload: %w", err)
	}

	if strings.TrimSpace(p.FileType) == "" {
		err := fmt.Errorf("process book file: payload file type is empty")
		slog.ErrorContext(ctx, "book processing failed: empty file type in payload",
			slog.Any(otelkeys.Error, err),
			slog.String(otelkeys.Path, p.Path),
			slog.String(otelkeys.FileName, p.FileName),
		)
		return fmt.Errorf("invalid payload: %w", err)
	}

	return nil
}

// resolveSourcePath ensures the source file exists. If it is missing, attempts
// to recover from a prior processing attempt that moved the file but failed to
// commit DB records. Returns the resolved path, whether processing should be
// skipped, and any hard error.
func resolveSourcePath(ctx context.Context, database *db.DB, p ProcessFilePayload, organizationType string, lookup bookFileLookupFunc) (string, bool, error) {
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
	bf, dbErr := lookup(ctx, database, p.Path)
	if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) {
		wrappedErr := fmt.Errorf("process book file: get book file by path %q: %w", p.Path, dbErr)
		slog.ErrorContext(ctx, "book processing failed: error looking up book file by path",
			slog.Any(otelkeys.Error, wrappedErr),
			slog.String(otelkeys.Path, p.Path),
		)
		return "", false, wrappedErr
	}
	if dbErr == nil {
		return linkExistingBookAndSkip(ctx, database, bf, p.LibraryID, p.Path)
	}

	// Attempt to find the file at any expected reorganized location.
	if p.LibraryRoot != "" {
		pathInfo := pathparser.ParseBookPath(p.Path, p.LibraryRoot)
		for _, candidatePath := range reorganizedCandidatePaths(ctx, p, pathInfo, organizationType) {
			if _, candidateStatErr := os.Stat(candidatePath); candidateStatErr == nil {
				// Check if the reorganized path is already indexed.
				bf, dbErr := lookup(ctx, database, candidatePath)
				if dbErr != nil && !errors.Is(dbErr, sql.ErrNoRows) {
					wrappedErr := fmt.Errorf("process book file: get book file by path %q: %w", candidatePath, dbErr)
					slog.ErrorContext(ctx, "book processing failed: error looking up reorganized book file by path",
						slog.Any(otelkeys.Error, wrappedErr),
						slog.String(otelkeys.Path, candidatePath),
					)
					return "", false, wrappedErr
				}
				if dbErr == nil {
					return linkExistingBookAndSkip(ctx, database, bf, p.LibraryID, candidatePath)
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

	slog.InfoContext(ctx, "source file no longer exists and could not be located, skipping",
		slog.String(otelkeys.Path, p.Path),
	)
	return "", true, nil
}

// linkExistingBookAndSkip logs that a book at path is already indexed,
// best-effort links it to libraryID, and returns the skip signal.
func linkExistingBookAndSkip(ctx context.Context, database *db.DB, bf *db.BookFile, libraryID, path string) (string, bool, error) {
	slog.InfoContext(ctx, "file already indexed, skipping",
		slog.String(otelkeys.Path, path),
		slog.String(otelkeys.BookID, bf.BookID),
	)
	if libraryID != "" {
		if err := database.AddBookToLibrary(ctx, libraryID, bf.BookID); err != nil {
			slog.WarnContext(ctx, "failed to associate existing book with library",
				slog.Any(otelkeys.Error, err),
				slog.String(otelkeys.Path, path),
				slog.String(otelkeys.BookID, bf.BookID),
				slog.String(otelkeys.LibraryID, libraryID),
			)
		}
	}
	return "", true, nil
}

// checkDuplicate returns true if the file at the given path is already indexed.
// When already indexed, it best-effort links the existing book to libraryID.
func checkDuplicate(ctx context.Context, database *db.DB, path, libraryID string, lookup bookFileLookupFunc) (bool, error) {
	bookFile, err := lookup(ctx, database, path)
	if err == nil {
		_, skip, skipErr := linkExistingBookAndSkip(ctx, database, bookFile, libraryID, path)
		return skip, skipErr
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
