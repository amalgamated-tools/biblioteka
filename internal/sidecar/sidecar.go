package sidecar

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// WriteSidecarFiles writes a cover image and metadata.opf alongside the book file.
// All operations are best-effort with WARN-level logging on failure.
func WriteSidecarFiles(ctx context.Context, dir string, meta *metadata.BookMetadata, title, authorName string) {
	var coverFilename, coverMediaType string

	if meta != nil && meta.CoverImageURL != "" {
		var err error
		coverFilename, coverMediaType, err = WriteCover(dir, meta.CoverImageURL)
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

	if err := WriteOPF(dir, opfData); err != nil {
		slog.WarnContext(ctx, "failed to write metadata.opf",
			slog.String(otelkeys.Path, dir),
			slog.Any(otelkeys.Error, err),
		)
	}
}
