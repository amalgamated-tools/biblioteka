package metadata

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

var ErrExifToolUnavailable = errors.New("exiftool is not available on this system")

// Extractor extracts metadata from book files. Concurrent ExtractMetadata calls are safe,
// but Close must not be called concurrently with other methods.
type Extractor struct {
	et *exif.Exiftool
}

func NewExtractor(ctx context.Context) (*Extractor, error) {
	et, err := exif.NewExiftool(ctx)
	if err != nil {
		slog.WarnContext(ctx, "exiftool not available; all metadata extraction disabled — only filename-derived metadata will be used", slog.Any(otelkeys.Error, err))
		return &Extractor{}, nil
	}
	return &Extractor{et: et}, nil
}

func (e *Extractor) Close(ctx context.Context) {
	if e.et != nil {
		if err := e.et.Close(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to close exiftool", slog.Any(otelkeys.Error, err))
		}
		e.et = nil
	}
}

func (e *Extractor) ExtractMetadata(ctx context.Context, path string) (*exif.ExifToolOutput, error) {
	if e.et == nil {
		return nil, ErrExifToolUnavailable
	}
	return e.extractExif(ctx, path)
}

func (e *Extractor) extractExif(ctx context.Context, path string) (*exif.ExifToolOutput, error) {
	output, err := e.et.ExtractMetadataFromFile(ctx, path)
	if err != nil {
		return nil, err
	}

	if output.Title == "" {
		output.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	if output.PublicationDate != "" {
		output.PublicationDate = normalizeExifDate(output.PublicationDate)
	}

	return output, nil
}

// normalizeExifDate converts ExifTool's "YYYY:MM:DD" date format to "YYYY-MM-DD".
func normalizeExifDate(s string) string {
	if len(s) >= 10 && s[4] == ':' && s[7] == ':' {
		return s[:4] + "-" + s[5:7] + "-" + s[8:10]
	}
	return s
}

// NormalizeISBN strips common prefixes (urn:isbn:, isbn:), whitespace, hyphens,
// and spaces from a raw ISBN string. It returns the cleaned value only if it looks
// like an ISBN-10 or ISBN-13; otherwise it returns "".
func NormalizeISBN(raw string) string {
	return exif.NormalizeISBN(raw)
}
