package exif

import (
	"context"
	"fmt"
	"log/slog"
	pathpkg "path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ParseTSV parses exiftool's tab-separated output into an ExifToolOutput.
// Each line is expected to be "Key\tValue". Repeated keys (Identifier,
// Manifest Item, Meta, Guide Reference) are collected into their
// respective slices using a flush-on-new-record strategy.
func ParseTSV(ctx context.Context, data, fileFormat string) (*ExifToolOutput, error) {
	normalizedFormat := strings.ToLower(strings.TrimPrefix(fileFormat, "."))

	out := &ExifToolOutput{
		Identifiers:   []Identifier{},
		MetaTags:      []MetaTag{},
		ManifestItems: []ManifestItem{},
		Extras:        map[string]string{},
		Format:        normalizedFormat,
	}

	var curIdent *Identifier
	var curMeta *MetaTag
	var curManifest *ManifestItem

	for line := range strings.SplitSeq(data, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		switch {
		case strings.HasPrefix(key, "Identifier"):
			curIdent = parseIdentifierLine(ctx, key, value, curIdent, out)

		case strings.HasPrefix(key, "Meta "):
			curMeta = parseMetaLine(ctx, key, value, curMeta, out)

		case strings.HasPrefix(key, "Manifest Item "):
			curManifest = parseManifestLine(ctx, key, value, curManifest, out)

		case strings.HasPrefix(key, "Guide Reference "), key == "Spine Toc", key == "Spine Itemref Idref":
			// Skip this, we don't care

		default:
			parseScalar(ctx, key, value, out)
		}
	}

	// Flush any pending grouped records.
	flushIdent(ctx, curIdent, out)
	flushMeta(ctx, curMeta, out)
	flushManifest(ctx, curManifest, out)
	finishBook(ctx, out)

	return out, nil
}

// --- Identifier parsing ---
func parseIdentifierLine(ctx context.Context, key, value string, cur *Identifier, out *ExifToolOutput) *Identifier {
	switch key {
	case "Identifier":
		// "Identifier" always completes a record. Use the pending
		// placeholder (created by preceding Scheme/Id lines) or start
		// fresh, then flush immediately.
		if cur == nil {
			cur = &Identifier{}
		}
		cur.Value = value
		flushIdent(ctx, cur, out)
		return nil // reset — no pending identifier
	case "Identifier Scheme":
		// Scheme precedes the Identifier value line it belongs to.
		if cur == nil {
			cur = &Identifier{}
		}
		cur.Scheme = value
		return cur
	case "Identifier Id":
		// Id precedes the Identifier value line it belongs to.
		if cur == nil {
			cur = &Identifier{}
		}
		cur.ID = value
		return cur
	default:
		return cur
	}
}

func flushIdent(ctx context.Context, cur *Identifier, out *ExifToolOutput) {
	if cur != nil {
		switch {
		case strings.EqualFold(cur.Scheme, "CALIBRE"), strings.HasPrefix(cur.Value, "urn:calibre:"):
			cur.Value = strings.TrimPrefix(cur.Value, "urn:calibre:")
			if out.CalibreID == "" || out.CalibreID == cur.Value {
				out.CalibreID = cur.Value
			} else {
				// If we already have a Calibre ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Calibre ID values.
				slog.DebugContext(ctx,
					"multiple Calibre ID values found; keeping first",
					slog.String(otelkeys.Existing, out.CalibreID),
					slog.String(otelkeys.CalibreID, cur.Value))
				out.Extras["Duplicate Calibre ID"] = cur.Value
			}
		case strings.EqualFold(cur.Scheme, "ISBN"), strings.HasPrefix(cur.Value, "urn:isbn"):
			// Normalize ISBNs to a digit-only string (10 or 13 characters), without a "urn:isbn:" prefix.
			isbn := NormalizeISBN(cur.Value)
			if len(isbn) == 10 {
				if out.ISBN10 == "" || out.ISBN10 == isbn {
					out.ISBN10 = isbn
				} else {
					// If we already have an ISBN-10, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-10 values.
					slog.WarnContext(ctx,
						"multiple isbn-10 values found; keeping first",
						slog.String(otelkeys.Existing, out.ISBN10),
						slog.String(otelkeys.ISBN, isbn))
					out.Extras["Duplicate ISBN-10"] = isbn
				}
			} else if len(isbn) == 13 {
				if out.ISBN13 == "" || out.ISBN13 == isbn {
					out.ISBN13 = isbn
				} else {
					// If we already have an ISBN-13, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-13 values.
					slog.WarnContext(ctx,
						"multiple isbn-13 values found; keeping first",
						slog.String(otelkeys.Existing, out.ISBN13),
						slog.String(otelkeys.ISBN, isbn))
					out.Extras["Duplicate ISBN-13"] = isbn
				}
			}
		case strings.EqualFold(cur.Scheme, "GOODREADS"), strings.HasPrefix(cur.Value, "urn:goodreads"):
			cur.Value = strings.TrimPrefix(cur.Value, "urn:goodreads:")
			if out.GoodreadsID == "" || out.GoodreadsID == cur.Value {
				out.GoodreadsID = cur.Value
			} else {
				// If we already have a Goodreads ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Goodreads ID values.
				slog.DebugContext(ctx,
					"multiple Goodreads ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoodreadsID),
					slog.String(otelkeys.GoodreadsID, cur.Value))
				out.Extras["Duplicate Goodreads ID"] = cur.Value
			}
		case strings.EqualFold(cur.Scheme, "AMAZON"), strings.EqualFold(cur.Scheme, "MOBI-ASIN"), strings.HasPrefix(cur.Value, "urn:amazon"):
			cur.Value = strings.TrimPrefix(cur.Value, "urn:amazon:")
			if out.ASIN == "" || out.ASIN == cur.Value {
				out.ASIN = cur.Value
			} else {
				// If we already have an ASIN, we could log a warning or decide which one to keep. For now, we'll just ignore additional ASIN values.
				slog.DebugContext(ctx,
					"multiple ASIN values found; keeping first",
					slog.String(otelkeys.Existing, out.ASIN),
					slog.String(otelkeys.ASIN, cur.Value))
				out.Extras["Duplicate ASIN"] = cur.Value
			}
		case strings.EqualFold(cur.Scheme, "GOOGLE"), strings.HasPrefix(cur.Value, "urn:google"):
			cur.Value = strings.TrimPrefix(cur.Value, "urn:google:")
			if out.GoogleID == "" || out.GoogleID == cur.Value {
				out.GoogleID = cur.Value
			} else {
				// If we already have a Google ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Google ID values.
				slog.DebugContext(ctx,
					"multiple Google ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoogleID),
					slog.String(otelkeys.GoogleID, cur.Value))
				out.Extras["Duplicate Google ID"] = cur.Value
			}
		case strings.HasPrefix(cur.Value, "urn:hardcoverbook:"):
			cur.Value = strings.TrimPrefix(cur.Value, "urn:hardcoverbook:")
			if out.HardcoverID == "" || out.HardcoverID == cur.Value {
				out.HardcoverID = cur.Value
			} else {
				// If we already have a Hardcover ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Hardcover ID values.
				slog.DebugContext(ctx,
					"multiple Hardcover ID values found; keeping first",
					slog.String(otelkeys.Existing, out.HardcoverID),
					slog.String(otelkeys.HardcoverID, cur.Value))
				out.Extras["Duplicate Hardcover ID"] = cur.Value
			}
		default:
			key := cur.Scheme
			if key == "" {
				key = cur.ID
				if key == "" {
					key = "Unknown"
				}
			}
			out.Extras[fmt.Sprintf("Identifier (%s)", key)] = cur.Value
		}
		out.Identifiers = append(out.Identifiers, *cur)
	}
}

// --- Meta parsing ---
func parseMetaLine(ctx context.Context, key, value string, cur *MetaTag, out *ExifToolOutput) *MetaTag {
	switch key {
	case "Meta Content":
		// "Meta Content" always appears first in a pair.
		flushMeta(ctx, cur, out)
		return &MetaTag{Content: value}
	case "Meta Name":
		if cur != nil {
			cur.Name = value
		}
		return cur
	default:
		return cur
	}
}

func flushMeta(ctx context.Context, cur *MetaTag, out *ExifToolOutput) {
	if cur != nil {
		slog.DebugContext(
			ctx,
			"parsed meta tag",
			slog.String(otelkeys.MetaName, cur.Name),
			slog.String(otelkeys.MetaContent, cur.Content),
		)
		out.MetaTags = append(out.MetaTags, *cur)
	}
}

// --- Manifest Item parsing ---

func parseManifestLine(ctx context.Context, key, value string, cur *ManifestItem, out *ExifToolOutput) *ManifestItem {
	switch key {
	case "Manifest Item Href":
		// "Manifest Item Href" starts a new manifest record.
		flushManifest(ctx, cur, out)
		return &ManifestItem{Href: value}
	case "Manifest Item Id":
		if cur != nil {
			cur.ID = value
		}
		return cur
	case "Manifest Item Media-type":
		if cur != nil {
			cur.MediaType = value
		}
		return cur
	case "Manifest Item Properties":
		if cur != nil {
			cur.Properties = value
		}
		return cur
	default:
		return cur
	}
}

func flushManifest(ctx context.Context, cur *ManifestItem, out *ExifToolOutput) {
	if cur != nil {
		if strings.EqualFold(cur.ID, "cover") && isLikelyImage(cur.Href, cur.MediaType) {
			slog.DebugContext(
				ctx,
				"identified cover image from manifest item",
				slog.String(otelkeys.ID, cur.ID),
				slog.String(otelkeys.Href, cur.Href),
				slog.String(otelkeys.MediaType, cur.MediaType),
				slog.String(otelkeys.Properties, cur.Properties),
			)
			out.CoverImage = cur
		}
		out.ManifestItems = append(out.ManifestItems, *cur)
	}
}

func isLikelyImage(href, mimeType string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") {
		return true
	}

	switch strings.ToLower(pathpkg.Ext(href)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".svg":
		return true
	default:
		return false
	}
}

// --- Guide Reference parsing ---

// --- Scalar fields ---

func parseScalar(ctx context.Context, key, value string, out *ExifToolOutput) {
	switch key {
	case "ExifTool Version Number":
		out.ExifToolVersion = value
	case "File Name":
		out.FileName = value
	case "Directory":
		out.Directory = value
	case "File Path":
		out.FilePath = value
	case "File Size":
		out.FileSize = value
	case "File Type":
		out.FileType = value
	case "File Type Extension":
		out.Format = value
	case "MIME Type":
		out.MIMEType = value
	case "Title", "Updated Title", "Book Name":
		if out.Title == "" {
			out.Title = value
		} else {
			slog.DebugContext(
				ctx,
				"multiple title values found; keeping first",
				slog.String(otelkeys.Existing, out.Title),
				slog.String(otelkeys.New, value))
			out.Extras[fmt.Sprintf("Duplicate Title (%s)", key)] = value
		}
	case "Creator File-as":
		out.CreatorFileAs = value
	case "Creator Role":
		out.CreatorRole = value
	case "Language":
		out.Language = value
	case "Date", "Publication Date", "Publish Date":
		if out.PublicationDate == "" {
			out.PublicationDate = value
		} else {
			// If we already have a publication date, we could log a warning or decide which one to keep. For now, we'll just ignore additional publication date values.
			slog.DebugContext(
				ctx,
				"multiple publication date values found; keeping first",
				slog.String(otelkeys.Existing, out.PublicationDate),
				slog.String(otelkeys.New, value),
			)
		}
	case "Publisher":
		out.Publisher = value
	case "Description":
		out.Description = value
	case "Subject":
		subjects := strings.Split(value, ", ")
		for _, subject := range subjects {
			subject = strings.TrimSpace(subject)
			if subject == "" {
				continue
			}
			// Avoid adding duplicate subjects if multiple Subject lines contain overlapping values.
			exists := false
			for _, existing := range out.Subjects {
				if existing == subject {
					exists = true
					break
				}
			}
			if !exists {
				out.Subjects = append(out.Subjects, subject)
			}
		}
	case "ISBN":
		rawValue := value
		value = NormalizeISBN(value)
		if len(value) == 10 {
			out.ISBN10 = value
		} else if len(value) == 13 {
			out.ISBN13 = value
		} else {
			// Preserve the original, unnormalized ISBN value in Extras
			out.Extras[key] = rawValue
		}
	case "ASIN":
		if out.ASIN == "" {
			out.ASIN = value
		}
	case "Author", "Creator":
		// Some formats (like MOBI) use "Author" instead of "Creator".
		if out.Author == "" {
			out.Author = value
		} else {
			// If we already have a Creator, we could log a warning or decide which one to keep. For now, we'll just ignore additional Author values.
			slog.DebugContext(
				ctx,
				"multiple Author/Creator values found; keeping first",
				slog.String(otelkeys.Existing, out.Author),
				slog.String(otelkeys.New, value))
			out.Extras[key] = value
		}
	default:
		slog.DebugContext(
			ctx,
			"unrecognized scalar field in exiftool output; storing in Extra",
			slog.String(otelkeys.Key, key),
			slog.String(otelkeys.Value, value),
		)
		out.Extras[key] = value
	}
}
