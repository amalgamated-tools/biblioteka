package sidecar

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// WriteSidecarFiles writes a cover image and OPF metadata file alongside the
// book file. For book_per_file libraries, sidecar names are derived from the
// book filename so multiple books can share an author directory safely.
// All operations are best-effort with WARN-level logging on failure.
func WriteSidecarFiles(ctx context.Context, bookFilePath string, meta *metadata.BookMetadata, title, authorName, organizationType string) {
	dir, baseName, err := sidecarTarget(bookFilePath, organizationType)
	if err != nil {
		slog.WarnContext(ctx, "failed to resolve sidecar target",
			slog.String(otelkeys.Path, bookFilePath),
			slog.Any(otelkeys.Error, err),
		)
		return
	}

	var coverFilename, coverMediaType string

	if meta != nil && meta.CoverImageURL != "" {
		coverFilename, coverMediaType, err = WriteCover(dir, meta.CoverImageURL, baseName)
		if err != nil {
			slog.WarnContext(ctx, "failed to write cover image",
				slog.String(otelkeys.Path, dir),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	opfData := OPFData{
		Title:          title,
		Author:         authorName,
		CoverFilename:  coverFilename,
		CoverMediaType: coverMediaType,
	}

	if meta != nil {
		opfData.Description = meta.Description
		opfData.Language = meta.Language
		opfData.Date = meta.PublicationDate
		opfData.Publisher = meta.Publisher
		opfData.ISBN = meta.ISBN
	}

	if err := WriteOPF(dir, opfData, baseName); err != nil {
		slog.WarnContext(ctx, "failed to write metadata.opf",
			slog.String(otelkeys.Path, dir),
			slog.Any(otelkeys.Error, err),
		)
	}
}
