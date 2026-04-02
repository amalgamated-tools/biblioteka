package jobs

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/pathparser"
)

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
func extractBookMetadata(ctx context.Context, extractor *metadata.Extractor, path, initialTitle string) *exif.ExifToolOutput {
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

	// Normalize ISBN in-place for downstream consumers.
	if meta.ISBN() != "" {
		if normalized := metadata.NormalizeISBN(meta.ISBN()); normalized != "" {
			meta.SetISBN(normalized)
		} else {
			meta.SetISBN("")
		}
	}

	slog.DebugContext(ctx, "metadata extracted successfully",
		slog.String(otelkeys.Path, path),
		slog.String(otelkeys.Title, meta.Title),
		slog.String(otelkeys.Format, meta.Format),
		slog.Any(otelkeys.BookMetadata, meta),
	)

	return meta
}

// resolveAuthorAndTitle merges metadata-derived and path-derived author and
// title values into final resolved values.
func resolveAuthorAndTitle(meta *exif.ExifToolOutput, pathInfo pathparser.PathInfo, currentTitle string) (string, string) {
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
