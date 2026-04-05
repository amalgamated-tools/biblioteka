package exif

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/sblinch/mobi"
)

// GetMobiCover extracts the cover image from a MOBI or AZW3 file and returns
// it as a base64-encoded data URL (data:image/jpeg;base64,...). The raw cover
// bytes are decoded as an image and re-encoded to JPEG to normalize the
// format. Returns ErrNoCover if the file has no embedded cover image.
func GetMobiCover(ctx context.Context, path string) (string, error) {
	e, err := mobi.NewReader(path)
	if err != nil {
		return "", fmt.Errorf("failed to open MOBI file: %w", err)
	}
	defer e.Close()

	coverstart, coverlength := e.CoverOffsetLength()
	if coverstart <= 0 || coverlength <= 0 {
		return "", ErrNoCover
	}

	// mobi.NewReader opens the file internally but its file handle is unexported,
	// so we must open the file again to seek to the raw cover bytes.
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("unable to open MOBI file: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(coverstart, 0); err != nil {
		return "", fmt.Errorf("unable to seek to cover offset: %w", err)
	}

	img, _, err := image.Decode(io.LimitReader(f, coverlength))
	if err != nil {
		return "", fmt.Errorf("unable to decode MOBI cover image: %w", err)
	}
	slog.DebugContext(ctx, "extracted MOBI cover image", slog.String(otelkeys.Path, path))

	var buf strings.Builder
	buf.WriteString("data:image/jpeg;base64,")
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := jpeg.Encode(encoder, img, nil); err != nil {
		slog.WarnContext(ctx, "failed to encode MOBI cover image as JPEG",
			slog.String(otelkeys.Path, path),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("failed to encode MOBI cover image as JPEG: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize base64 encoding: %w", err)
	}

	return buf.String(), nil
}
