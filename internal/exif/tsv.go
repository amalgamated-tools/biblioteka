package exif

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"os"
	pathpkg "path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/sblinch/mobi"
)

// ExifToolOutput holds the parsed result of exiftool's tab-separated output
// (produced by `exiftool -a -u -f -t`).
type ExifToolOutput struct {
	// File info
	FileName  string
	Directory string
	FileSize  string
	FileType  string
	MIMEType  string

	// Book metadata
	ASIN            string
	CoverImage      *ManifestItem
	CoverImageURL   string
	Author          string
	CreatorFileAs   string
	CreatorRole     string
	Description     string
	Format          string
	GoodreadsID     string
	GoogleID        string
	HardcoverID     string
	ISBN10          string
	ISBN13          string
	Language        string
	PublicationDate string
	Publisher       string
	Subjects        []string
	Title           string

	// Repeated/nested structures
	Identifiers   []Identifier
	MetaTags      []MetaTag
	ManifestItems []ManifestItem

	// Catch-all for any unrecognized scalar fields.
	Extra map[string]string
}

func (e *ExifToolOutput) ISBN() string {
	if e.ISBN13 != "" {
		return e.ISBN13
	}
	if e.ISBN10 != "" {
		return e.ISBN10
	}
	return ""
}

func (e *ExifToolOutput) SetISBN(isbn string) {
	switch len(isbn) {
	case 0:
		e.ISBN10 = ""
		e.ISBN13 = ""
	case 10:
		e.ISBN10 = isbn
	case 13:
		e.ISBN13 = isbn
	}
}

// Identifier represents a Dublin Core identifier extracted from EPUB metadata.
// Scheme may be empty for bare URN-style identifiers.
type Identifier struct {
	Value  string
	Scheme string
	ID     string
}

// MetaTag represents a <meta> name/content pair from EPUB metadata.
type MetaTag struct {
	Content string
	Name    string
}

// ManifestItem represents an OPF manifest entry.
type ManifestItem struct {
	Href       string
	ID         string
	MediaType  string
	Properties string
}

// GuideReference represents an OPF guide reference.
type GuideReference struct {
	Href  string
	Title string
	Type  string
}

// ParseTSV parses exiftool's tab-separated output into an ExifToolOutput.
// Each line is expected to be "Key\tValue". Repeated keys (Identifier,
// Manifest Item, Meta, Guide Reference) are collected into their
// respective slices using a flush-on-new-record strategy.
func ParseTSV(ctx context.Context, data, fileFormat string) (*ExifToolOutput, error) {
	out := &ExifToolOutput{
		Identifiers:   []Identifier{},
		MetaTags:      []MetaTag{},
		ManifestItems: []ManifestItem{},
		Extra:         map[string]string{},
		Format:        fileFormat,
	}

	var curIdent *Identifier
	var curMeta *MetaTag
	var curManifest *ManifestItem

	for line := range strings.SplitSeq(data, "\n") {
		key, value, ok := strings.Cut(line, "\t")
		if !ok {
			slog.WarnContext(ctx, "malformed line in exiftool output (no tab separator)", slog.String(otelkeys.Line, line))
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
		case strings.EqualFold(cur.Scheme, "ISBN"), strings.HasPrefix(cur.Value, "urn:isbn"):
			// Normalize ISBNs to 13-digit format with "urn:isbn:" prefix.
			isbn := NormalizeISBN(ctx, cur.Value)
			if len(isbn) == 10 {
				if out.ISBN10 == "" {
					out.ISBN10 = isbn
				} else {
					// If we already have an ISBN-10, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-10 values.
					slog.WarnContext(ctx,
						"multiple isbn-10 values found; keeping first",
						slog.String(otelkeys.Existing, out.ISBN10),
						slog.String(otelkeys.ISBN, isbn))
				}
			} else if len(isbn) == 13 {
				if out.ISBN13 == "" {
					out.ISBN13 = isbn
				} else {
					// If we already have an ISBN-13, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-13 values.
					slog.WarnContext(ctx,
						"multiple isbn-13 values found; keeping first",
						slog.String(otelkeys.Existing, out.ISBN13),
						slog.String(otelkeys.ISBN, isbn))
				}
			}
		case strings.EqualFold(cur.Scheme, "AMAZON"), strings.EqualFold(cur.Scheme, "MOBI-ASIN"):
			if out.ASIN == "" {
				out.ASIN = cur.Value
			} else {
				// If we already have an ASIN, we could log a warning or decide which one to keep. For now, we'll just ignore additional ASIN values.
				slog.WarnContext(ctx,
					"multiple ASIN values found; keeping first",
					slog.String(otelkeys.Existing, out.ASIN),
					slog.String(otelkeys.ASIN, cur.Value))
			}
		case strings.HasPrefix(cur.Value, "urn:amazon"):
			if out.ASIN == "" {
				out.ASIN = strings.TrimPrefix(cur.Value, "urn:amazon:")
			} else {
				// If we already have an ASIN, we could log a warning or decide which one to keep. For now, we'll just ignore additional ASIN values.
				slog.WarnContext(ctx,
					"multiple ASIN values found; keeping first",
					slog.String(otelkeys.Existing, out.ASIN),
					slog.String(otelkeys.ASIN, cur.Value))
			}
		case strings.EqualFold(cur.Scheme, "GOOGLE"):
			if out.GoogleID == "" {
				out.GoogleID = cur.Value
			} else {
				// If we already have a Google ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Google ID values.
				slog.WarnContext(ctx,
					"multiple Google ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoogleID),
					slog.String(otelkeys.GoogleID, cur.Value))
			}
		case strings.EqualFold(cur.Scheme, "GOODREADS"):
			if out.GoodreadsID == "" {
				out.GoodreadsID = cur.Value
			} else {
				// If we already have a Goodreads ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Goodreads ID values.
				slog.WarnContext(ctx,
					"multiple Goodreads ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoodreadsID),
					slog.String(otelkeys.GoodreadsID, cur.Value))
			}
		case strings.HasPrefix(cur.Value, "urn:hardcoverbook:"):
			if out.HardcoverID == "" {
				out.HardcoverID = strings.TrimPrefix(cur.Value, "urn:hardcoverbook:")
			} else {
				// If we already have a Hardcover ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Hardcover ID values.
				slog.WarnContext(ctx,
					"multiple Hardcover ID values found; keeping first",
					slog.String(otelkeys.Existing, out.HardcoverID),
					slog.String(otelkeys.HardcoverID, cur.Value))
			}
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
	case "File Name":
		out.FileName = value
	case "Directory":
		out.Directory = value
	case "File Size":
		out.FileSize = value
	case "File Type":
		out.FileType = value
	case "MIME Type":
		out.MIMEType = value
	case "Title", "Updated Title":
		out.Title = value
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
		out.Subjects = strings.Split(value, ", ")
	case "ISBN":
		if len(value) == 10 {
			out.ISBN10 = value
		} else if len(value) == 13 {
			out.ISBN13 = value
		} else {
			out.Extra[key] = value
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
			out.Extra[key] = value
		}
	default:
		slog.DebugContext(
			ctx,
			"unrecognized scalar field in exiftool output; storing in Extra",
			slog.String(otelkeys.Key, key),
			slog.String(otelkeys.Value, value),
		)
		out.Extra[key] = value
	}
}

func finishBook(ctx context.Context, out *ExifToolOutput) {
	switch out.FileType {
	case "EPUB":
		finishEPUB(ctx, out)
	case "MOBI":
		finishMOBI(ctx, out)
	}
}

func finishEPUB(ctx context.Context, out *ExifToolOutput) {
	// let's see if we have an ISBN10 or ISBN13
	if out.ISBN10 == "" && out.ISBN13 == "" {
		// we don't have either, but maybe we have an identifier that looks like an ISBN
		for _, ident := range out.Identifiers {
			// sometimes this gets put into an id of "bookid"
			var assumedISBN string
			if strings.EqualFold(ident.ID, "bookid") {
				assumedISBN = ident.Value
			} else {
				// let's see if value is 10 or 13 chars and looks like an ISBN
				assumedISBN = NormalizeISBN(ctx, ident.Value)
			}
			if assumedISBN != "" {
				if len(assumedISBN) == 10 {
					out.ISBN10 = assumedISBN
				} else if len(assumedISBN) == 13 {
					out.ISBN13 = assumedISBN
				}
			}
			// if we found something that looks like an ISBN, we can stop looking
			if out.ISBN10 != "" || out.ISBN13 != "" {
				break
			}
		}
	}

	// let's see if we have something ASIN-like in our identifiers
	if out.ASIN == "" {
		for _, ident := range out.Identifiers {
			// ASINs are 10-character alphanumeric strings, often starting with "B0" for newer books.
			if len(ident.Value) == 10 && isASIN(ident.Value) {
				out.ASIN = ident.Value
				break
			}
		}
	}

	// let's see if we have a cover image
	if out.CoverImage == nil {
		for _, item := range out.ManifestItems {
			if isLikelyImage(item.Href, item.MediaType) {
				if strings.EqualFold(item.Properties, "cover-image") || strings.EqualFold(item.ID, "cover") {
					out.CoverImage = &item
					break
				}
			}
		}
	}

	if out.CoverImage != nil {
		// let's get the cover image
		coverImageURL, err := extractEPUBCoverDataURL(ctx, out.CoverImage, pathpkg.Join(out.Directory, out.FileName))
		if err != nil {
			slog.WarnContext(ctx, "failed to extract cover image", slog.String(otelkeys.Error, err.Error()))
		} else {
			out.CoverImageURL = coverImageURL
		}
	}
}

func finishMOBI(ctx context.Context, out *ExifToolOutput) {
	coverImageURL, err := GetMobiCover(ctx, pathpkg.Join(out.Directory, out.FileName))
	if err != nil {
		slog.WarnContext(ctx, "failed to extract MOBI cover image", slog.String(otelkeys.Error, err.Error()))
	} else {
		out.CoverImageURL = coverImageURL
	}
}

func isASIN(s string) bool {
	for i := range s {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

// NormalizeISBN strips common prefixes (urn:isbn:, isbn:), whitespace, hyphens,
// and spaces from a raw ISBN string. It returns the cleaned value only if it looks
// like an ISBN-10 or ISBN-13: 10 or 13 characters consisting of digits, with
// ISBN-10 allowing an 'X' (or 'x') as the final checksum character; otherwise it
// returns "".
func NormalizeISBN(ctx context.Context, raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "urn:isbn:"):
		s = s[len("urn:isbn:"):]
	case strings.HasPrefix(lower, "isbn:"):
		s = s[len("isbn:"):]
	}
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)

	switch len(s) {
	case 10:
		// First 9 characters must be digits.
		for i := range 9 {
			if s[i] < '0' || s[i] > '9' {
				return ""
			}
		}
		// Last character may be a digit or 'X'/'x'.
		last := s[9]
		if (last < '0' || last > '9') && last != 'X' && last != 'x' {
			return ""
		}
		// Normalize to upper-case 'X' if present.
		if last == 'x' {
			s = s[:9] + "X"
		}
		return s
	case 13:
		// All characters must be digits.
		for i := range 13 {
			if s[i] < '0' || s[i] > '9' {
				return ""
			}
		}
		return s
	default:
		return ""
	}
}

func GetMobiCover(ctx context.Context, path string) (string, error) {
	var i image.Image
	e, err := mobi.NewReader(path)
	if err != nil {
		return "", fmt.Errorf("failed to open MOBI file: %w", err)
	}
	defer e.Close()

	coverstart, coverlength := e.CoverOffsetLength()
	if coverstart <= 0 {
		return "", errors.New("no cover found in MOBI file")
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("unable to open MOBI file: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(coverstart, 0); err != nil {
		return "", fmt.Errorf("unable to seek to cover offset: %w", err)
	}

	ltd := io.LimitReader(f, coverlength)
	i, _, err = image.Decode(ltd)
	if err != nil {
		return "", fmt.Errorf("unable to decode MOBI cover image: %w", err)
	}
	slog.InfoContext(ctx, "extracted MOBI cover image")

	// name is the name of the format that was decoded (e.g. "jpeg", "png"). We want to convert it to a data URL, but image.Decode doesn't give us the original bytes or MIME type, so we'll have to re-encode it as JPEG (since that's the most common format for MOBI covers) and base64-encode that.

	var buf strings.Builder
	buf.WriteString("data:image/jpeg;base64,")
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := jpeg.Encode(encoder, i, nil); err != nil {
		slog.WarnContext(ctx, "failed to encode MOBI cover image as JPEG",
			// slog.String(otelkeys.Path, path),
			slog.Any(otelkeys.Error, err),
		)
		return "", fmt.Errorf("failed to encode MOBI cover image as JPEG: %w", err)
	} else {
		encoder.Close()
		return buf.String(), nil
	}
}
