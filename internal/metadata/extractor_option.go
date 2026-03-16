package metadata

import (
	"context"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/barasher/go-exiftool"
)

type ExtractorOption func(*Extractor) error

func WithExiftoolBinaryPath(path string) ExtractorOption {
	return func(e *Extractor) error {
		et, err := exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(path))
		if err != nil {
			slog.WarnContext(context.Background(), "exiftool not available; exif-based metadata extraction disabled", slog.Any(otelkeys.Error, err))
			return err
		}

		e.et = et
		return nil
	}
}
