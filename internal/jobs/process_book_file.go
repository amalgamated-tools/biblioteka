package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
	"github.com/amalgamated-tools/biblioteka/internal/sidecar"
)

// ProcessBookFile orchestrates the complete processing of a single book file:
// validation, deduplication, metadata extraction, file reorganization, and
// database record creation.
func ProcessBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, p ProcessFilePayload) error {
	return processBookFile(ctx, database, extractor, p, defaultBookFileLookup)
}

func lookupOrganizationType(ctx context.Context, database *db.DB, p ProcessFilePayload) string {
	if p.LibraryRoot == "" || p.LibraryID == "" {
		return ""
	}

	lib, libErr := database.GetLibrary(ctx, p.LibraryID)
	if libErr != nil {
		slog.WarnContext(ctx, "could not look up library for organization type",
			slog.String(otelkeys.LibraryID, p.LibraryID),
			slog.Any(otelkeys.Error, libErr),
		)
		return ""
	}

	return lib.OrganizationType
}

func processBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, p ProcessFilePayload, lookup bookFileLookupFunc) error {
	if database == nil {
		err := fmt.Errorf("process book file: database is nil")
		slog.ErrorContext(ctx, "book processing failed: database is nil",
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if extractor == nil {
		err := fmt.Errorf("process book file: extractor is nil")
		slog.ErrorContext(ctx, "book processing failed: extractor is nil",
			slog.Any(otelkeys.Error, err),
		)
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := validatePayload(ctx, p); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	organizationType := lookupOrganizationType(ctx, database, p)

	resolvedPath, skip, err := resolveSourcePath(ctx, database, p, organizationType, lookup)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}
	if skip {
		return nil
	}
	p.Path = resolvedPath

	alreadyIndexed, err := checkDuplicate(ctx, database, p.Path, p.LibraryID, lookup)
	if err != nil {
		return fmt.Errorf("failed to check for duplicate: %w", err)
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

	filePath, skip, err := maybeReorganizeFile(ctx, database, p.Path, p.LibraryRoot, authorName, title, p.LibraryID, organizationType, lookup)
	if err != nil {
		return fmt.Errorf("failed to reorganize file: %w", err)
	}
	if skip {
		return nil
	}

	book, err := createBookRecord(ctx, database, title, meta, p, filePath)
	if err != nil {
		return fmt.Errorf("failed to create book record: %w", err)
	}

	linkBookAssociations(ctx, database, book.ID, authorName, p.LibraryID, pathInfo, filePath)

	sidecar.WriteSidecarFiles(ctx, filePath, meta, title, authorName, organizationType)

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
