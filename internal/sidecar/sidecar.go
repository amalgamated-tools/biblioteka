package sidecar

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/metadata"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// WriteSidecarFiles writes a cover image and OPF metadata file alongside the
// book file. When baseName is empty, files are named "cover.{ext}" and
// "metadata.opf". When baseName is set (e.g. for book_per_file mode), files
// are named "{baseName}.{ext}" and "{baseName}.opf" so that multiple books can
// share a directory without overwriting each other's sidecar files.
// All operations are best-effort with WARN-level logging on failure.
func WriteSidecarFiles(ctx context.Context, dir string, meta *metadata.BookMetadata, title, authorName, baseName string) {
	var coverFilename, coverMediaType string

	if meta != nil && meta.CoverImageURL != "" {
		var err error
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
