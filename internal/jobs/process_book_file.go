package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
	"github.com/amalgamated-tools/biblioteka/internal/sidecar"
)

// ProcessBookFile orchestrates the complete processing of a single book file:
// validation, deduplication, metadata extraction, file reorganization, and
// database record creation. When enqueuer is non-nil, a Goodreads enrichment
// job is enqueued after the book record is created.
func ProcessBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, enqueuer Enqueuer, p ProcessFilePayload) error {
	return processBookFile(ctx, database, extractor, enqueuer, p, defaultBookFileLookup)
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

func processBookFile(ctx context.Context, database *db.DB, extractor *metadata.Extractor, enqueuer Enqueuer, p ProcessFilePayload, lookup bookFileLookupFunc) error {
	if database == nil {
		err := errors.New("invalid configuration: process book file: database is nil")
		slog.ErrorContext(ctx, "book processing failed: database is nil",
			slog.Any(otelkeys.Error, err),
		)
		return err
	}

	if extractor == nil {
		err := errors.New("invalid configuration: process book file: extractor is nil")
		slog.ErrorContext(ctx, "book processing failed: extractor is nil",
			slog.Any(otelkeys.Error, err),
		)
		return err
	}

	if err := validatePayload(ctx, p); err != nil {
		// validatePayload already returns errors wrapped with the "invalid payload: %w" prefix
		return err
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

	// User-supplied overrides from the upload endpoint take precedence over
	// everything extracted from the file or derived from the path.
	if p.OverrideTitle != "" {
		title = p.OverrideTitle
	}
	if p.OverrideAuthor != "" {
		authorName = p.OverrideAuthor
	}

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

	maybeEnqueueGoodreads(ctx, enqueuer, book.ID, p.UserID)

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

// maybeEnqueueGoodreads enqueues an enrich:goodreads job for the given book
// when an enqueuer is available and a user ID is provided. When no user
// context is available (e.g., during automated library scans), enrichment
// is skipped. Failures are logged but never propagated.
func maybeEnqueueGoodreads(ctx context.Context, enqueuer Enqueuer, bookID, userID string) {
	if enqueuer == nil {
		return
	}

	if userID == "" {
		slog.DebugContext(ctx, "skipping Goodreads enrichment: no user context available",
			slog.String(otelkeys.BookID, bookID),
		)
		return
	}

	enqueueCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	if _, err := enqueuer.Enqueue(enqueueCtx, JobEnrichGoodreads, EnrichGoodreadsPayload{
		BookID: bookID,
		UserID: userID,
	}, WithUnique(24*time.Hour)); err != nil {
		slog.WarnContext(ctx, "failed to enqueue enrich:goodreads job",
			slog.String(otelkeys.BookID, bookID),
			slog.Any(otelkeys.Error, err),
		)
	}
}
