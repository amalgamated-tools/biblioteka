package sidecar

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// WriteSidecarFiles writes cover.jpg and metadata.opf alongside the book file.
// All operations are best-effort with WARN-level logging on failure.
func WriteSidecarFiles(ctx context.Context, dir string, meta *metadata.BookMetadata, title, authorName string) {
	var hasCover bool

	if meta != nil && meta.CoverImageURL != "" {
		if err := WriteCover(dir, meta.CoverImageURL); err != nil {
			slog.WarnContext(ctx, "failed to write cover.jpg",
				slog.String(otelkeys.Path, dir),
				slog.Any(otelkeys.Error, err),
			)
		} else {
			hasCover = true
		}
	}

	opfData := OPFData{
		Title:    title,
		Author:   authorName,
		HasCover: hasCover,
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
