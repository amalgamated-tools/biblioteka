package exif

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

func finishBook(ctx context.Context, out *ExifToolOutput) {
	switch out.FileType {
	case "EPUB":
		finishEPUB(ctx, out)
	case "MOBI", "AZW3":
		finishMOBI(ctx, out)
	}
}

func finishEPUB(ctx context.Context, out *ExifToolOutput) {
	if out.ISBN10 == "" && out.ISBN13 == "" {
		for _, ident := range out.Identifiers {
			assumedISBN := NormalizeISBN(ident.Value)
			if assumedISBN != "" {
				if len(assumedISBN) == 10 {
					out.ISBN10 = assumedISBN
				} else if len(assumedISBN) == 13 {
					out.ISBN13 = assumedISBN
				}
			}
			if out.ISBN10 != "" || out.ISBN13 != "" {
				break
			}
		}
	}

	if out.ASIN == "" {
		for _, ident := range out.Identifiers {
			switch strings.ToUpper(ident.Scheme) {
			case "ISBN", "CALIBRE", "GOODREADS", "GOOGLE", "HARDCOVERBOOK":
				continue
			}
			if NormalizeISBN(ident.Value) != "" {
				continue
			}
			if len(ident.Value) == 10 && isASIN(ident.Value) {
				out.ASIN = ident.Value
				break
			}
		}
	}

	// Cover image discovery, in priority order:
	// 1. Already found during manifest flush (id == "cover" with image type)
	// 2. <meta name="cover" content="ITEM_ID"> pointing to a manifest item
	// 3. Manifest item with properties="cover-image"
	// 4. Manifest item whose ID or href contains "cover" and has image type
	// 5. Single-image fallback (only one image in the manifest)

	// Strategy 2: <meta name="cover" content="ITEM_ID">
	if out.CoverImage == nil {
		coverID := ""
		for _, mt := range out.MetaTags {
			if strings.EqualFold(mt.Name, "cover") {
				coverID = strings.TrimSpace(mt.Content)
				break
			}
		}
		if coverID != "" {
			for i, item := range out.ManifestItems {
				if strings.TrimSpace(item.ID) == coverID && isLikelyImage(item.Href, item.MediaType) {
					out.CoverImage = &out.ManifestItems[i]
					break
				}
			}
		}
	}

	// Strategy 3: properties="cover-image" (EPUB3)
	// The properties attribute is a space-separated token list (e.g. "cover-image scripted"),
	// so we must check individual tokens rather than the whole string.
	if out.CoverImage == nil {
		for i, item := range out.ManifestItems {
			if !isLikelyImage(item.Href, item.MediaType) {
				continue
			}
			if hasProperty(item.Properties, "cover-image") {
				out.CoverImage = &out.ManifestItems[i]
				break
			}
		}
	}

	// Strategy 4: ID/href contains "cover"
	if out.CoverImage == nil {
		for i, item := range out.ManifestItems {
			if !isLikelyImage(item.Href, item.MediaType) {
				continue
			}
			if strings.Contains(strings.ToLower(item.ID), "cover") || strings.Contains(strings.ToLower(item.Href), "cover") {
				out.CoverImage = &out.ManifestItems[i]
				break
			}
		}
	}

	// Strategy 5: single-image fallback
	if out.CoverImage == nil {
		var onlyImage *ManifestItem
		imageCount := 0
		for i, item := range out.ManifestItems {
			if isLikelyImage(item.Href, item.MediaType) {
				onlyImage = &out.ManifestItems[i]
				imageCount++
				if imageCount > 1 {
					break
				}
			}
		}
		if imageCount == 1 {
			out.CoverImage = onlyImage
		}
	}

	if out.CoverImage == nil {
		return
	}

	coverImageURL, err := extractEPUBCoverDataURL(ctx, out.CoverImage, filepath.Join(out.Directory, out.FileName))
	if err != nil {
		slog.WarnContext(ctx, "failed to extract cover image", slog.Any(otelkeys.Error, err))
		return
	}
	out.CoverImageURL = coverImageURL
}

func finishMOBI(ctx context.Context, out *ExifToolOutput) {
	coverImageURL, err := GetMobiCover(ctx, filepath.Join(out.Directory, out.FileName))
	if err != nil {
		if errors.Is(err, ErrNoCover) {
			slog.DebugContext(ctx, "no embedded MOBI cover image found", slog.String(otelkeys.Path, out.FileName))
		} else {
			slog.WarnContext(ctx, "failed to extract MOBI cover image", slog.Any(otelkeys.Error, err))
		}
		return
	}
	out.CoverImageURL = coverImageURL
}

// hasProperty reports whether the space-separated properties string contains
// the given token (case-insensitive). EPUB3 manifest items may have multiple
// properties like "cover-image scripted".
func hasProperty(properties, token string) bool {
	for prop := range strings.SplitSeq(properties, " ") {
		if strings.EqualFold(prop, token) {
			return true
		}
	}
	return false
}
