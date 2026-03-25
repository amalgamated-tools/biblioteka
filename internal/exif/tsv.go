package exif

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	pathpkg "path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
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
	Creator         string
	CreatorFileAs   string
	CreatorRole     string
	Date            string
	Description     string
	Format          string
	GoodreadsID     string
	GoogleID        string
	HardcoverID     string
	ISBN            string
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
	SpineItemrefs []string
	GuideRefs     []GuideReference

	// Catch-all for any unrecognized scalar fields.
	Extra map[string]string
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
func ParseTSV(data, fileFormat string) (*ExifToolOutput, error) {
	out := &ExifToolOutput{
		Identifiers:   []Identifier{},
		MetaTags:      []MetaTag{},
		ManifestItems: []ManifestItem{},
		SpineItemrefs: []string{},
		GuideRefs:     []GuideReference{},
		Extra:         map[string]string{},
		Format:        fileFormat,
	}

	var curIdent *Identifier
	var curMeta *MetaTag
	var curManifest *ManifestItem
	var curGuide *GuideReference

	for line := range strings.SplitSeq(data, "\n") {
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
			curIdent = parseIdentifierLine(key, value, curIdent, out)

		case strings.HasPrefix(key, "Meta "):
			curMeta = parseMetaLine(key, value, curMeta, out)

		case strings.HasPrefix(key, "Manifest Item "):
			curManifest = parseManifestLine(key, value, curManifest, out)

		case strings.HasPrefix(key, "Guide Reference "):
			curGuide = parseGuideLine(key, value, curGuide, out)

		case key == "Spine Toc":
			// Spine Toc is a scalar; ignore or store in Extra.
			out.Extra[key] = value

		case key == "Spine Itemref Idref":
			out.SpineItemrefs = append(out.SpineItemrefs, value)

		default:
			parseScalar(key, value, out)
		}
	}

	// Flush any pending grouped records.
	flushIdent(curIdent, out)
	flushMeta(curMeta, out)
	flushManifest(curManifest, out)
	flushGuide(curGuide, out)

	finishBook(out)

	jsonOut, err := json.MarshalIndent(out, "", "  ")
	if err == nil {
		outputFile := fmt.Sprintf("exif_output_%s.json", pathpkg.Base(out.FileName))
		os.WriteFile(outputFile, jsonOut, 0644)
	}

	return out, nil
}

// --- Identifier parsing ---
func parseIdentifierLine(key, value string, cur *Identifier, out *ExifToolOutput) *Identifier {
	switch key {
	case "Identifier":
		// "Identifier" always completes a record. Use the pending
		// placeholder (created by preceding Scheme/Id lines) or start
		// fresh, then flush immediately.
		if cur == nil {
			cur = &Identifier{}
		}
		cur.Value = value
		flushIdent(cur, out)
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

func flushIdent(cur *Identifier, out *ExifToolOutput) {
	if cur != nil {
		switch {
		case strings.EqualFold(cur.Scheme, "ISBN"), strings.HasPrefix(cur.Value, "urn:isbn"):
			// Normalize ISBNs to 13-digit format with "urn:isbn:" prefix.
			isbn := NormalizeISBN(cur.Value)
			if len(isbn) == 10 {
				if out.ISBN == "" {
					out.ISBN = isbn
				} else {
					// If we already have an ISBN-10, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-10 values.
					slog.Warn(
						"multiple isbn-10 values found; keeping first",
						slog.String(otelkeys.Existing, out.ISBN),
						slog.String(otelkeys.ISBN, isbn))
				}
			} else if len(isbn) == 13 {
				if out.ISBN13 == "" {
					out.ISBN13 = isbn
				} else {
					// If we already have an ISBN-13, we could log a warning or decide which one to keep. For now, we'll just ignore additional ISBN-13 values.
					slog.Warn(
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
				slog.Warn(
					"multiple ASIN values found; keeping first",
					slog.String(otelkeys.Existing, out.ASIN),
					slog.String(otelkeys.ASIN, cur.Value))
			}
		case strings.HasPrefix(cur.Value, "urn:amazon"):
			if out.ASIN == "" {
				out.ASIN = strings.TrimPrefix(cur.Value, "urn:amazon:")
			} else {
				// If we already have an ASIN, we could log a warning or decide which one to keep. For now, we'll just ignore additional ASIN values.
				slog.Warn(
					"multiple ASIN values found; keeping first",
					slog.String(otelkeys.Existing, out.ASIN),
					slog.String(otelkeys.ASIN, cur.Value))
			}
		case strings.EqualFold(cur.Scheme, "GOOGLE"):
			if out.GoogleID == "" {
				out.GoogleID = cur.Value
			} else {
				// If we already have a Google ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Google ID values.
				slog.Warn(
					"multiple Google ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoogleID),
					slog.String(otelkeys.GoogleID, cur.Value))
			}
		case strings.EqualFold(cur.Scheme, "GOODREADS"):
			if out.GoodreadsID == "" {
				out.GoodreadsID = cur.Value
			} else {
				// If we already have a Goodreads ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Goodreads ID values.
				slog.Warn(
					"multiple Goodreads ID values found; keeping first",
					slog.String(otelkeys.Existing, out.GoodreadsID),
					slog.String(otelkeys.GoodreadsID, cur.Value))
			}
		case strings.HasPrefix(cur.Value, "urn:hardcoverbook:"):
			if out.HardcoverID == "" {
				out.HardcoverID = strings.TrimPrefix(cur.Value, "urn:hardcoverbook:")
			} else {
				// If we already have a Hardcover ID, we could log a warning or decide which one to keep. For now, we'll just ignore additional Hardcover ID values.
				slog.Warn(
					"multiple Hardcover ID values found; keeping first",
					slog.String(otelkeys.Existing, out.HardcoverID),
					slog.String(otelkeys.HardcoverID, cur.Value))
			}
		}
		out.Identifiers = append(out.Identifiers, *cur)
	}
}

// --- Meta parsing ---
func parseMetaLine(key, value string, cur *MetaTag, out *ExifToolOutput) *MetaTag {
	switch key {
	case "Meta Content":
		// "Meta Content" always appears first in a pair.
		flushMeta(cur, out)
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

func flushMeta(cur *MetaTag, out *ExifToolOutput) {
	if cur != nil {
		out.MetaTags = append(out.MetaTags, *cur)
	}
}

// --- Manifest Item parsing ---

func parseManifestLine(key, value string, cur *ManifestItem, out *ExifToolOutput) *ManifestItem {
	switch key {
	case "Manifest Item Href":
		// "Manifest Item Href" starts a new manifest record.
		flushManifest(cur, out)
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

func flushManifest(cur *ManifestItem, out *ExifToolOutput) {
	if cur != nil {
		if strings.EqualFold(cur.ID, "cover") && isLikelyImage(cur.Href, cur.MediaType) {
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

func parseGuideLine(key, value string, cur *GuideReference, out *ExifToolOutput) *GuideReference {
	switch key {
	case "Guide Reference Href":
		flushGuide(cur, out)
		return &GuideReference{Href: value}
	case "Guide Reference Title":
		if cur != nil {
			cur.Title = value
		}
		return cur
	case "Guide Reference Type":
		if cur != nil {
			cur.Type = value
		}
		return cur
	default:
		return cur
	}
}

func flushGuide(cur *GuideReference, out *ExifToolOutput) {
	if cur != nil {
		out.GuideRefs = append(out.GuideRefs, *cur)
	}
}

// --- Scalar fields ---

func parseScalar(key, value string, out *ExifToolOutput) {
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
	case "Creator":
		out.Creator = value
	case "Creator File-as":
		out.CreatorFileAs = value
	case "Creator Role":
		out.CreatorRole = value
	case "Language":
		out.Language = value
	case "Date":
		out.Date = value
	case "Publisher":
		out.Publisher = value
	case "Description":
		out.Description = value
	case "Subject":
		out.Subjects = strings.Split(value, ", ")
	case "ISBN":
		if len(value) == 10 {
			out.ISBN = value
		} else if len(value) == 13 {
			out.ISBN13 = value
		} else {
			out.Extra[key] = value
		}
	case "Publication Date", "Publish Date":
		out.PublicationDate = value
	case "Author":
		// Some formats (like MOBI) use "Author" instead of "Creator".
		if out.Creator == "" {
			out.Creator = value
		} else {
			// If we already have a Creator, we could log a warning or decide which one to keep. For now, we'll just ignore additional Author values.
			slog.Warn(
				"multiple Author/Creator values found; keeping first",
				slog.String(otelkeys.Existing, out.Creator),
				slog.String(otelkeys.Author, value))
			out.Extra[key] = value
		}
	default:
		out.Extra[key] = value
	}
}

func finishBook(out *ExifToolOutput) {
	switch out.FileType {
	case "EPUB":
		finishEPUB(out)
	case "MOBI":
		finishMOBI(out)
	}
}

func finishEPUB(out *ExifToolOutput) {
	// let's see if we have an ISBN10 or ISBN13
	if out.ISBN == "" && out.ISBN13 == "" {
		// we don't have either, but maybe we have an identifier that looks like an ISBN
		for _, ident := range out.Identifiers {
			// sometimes this gets put into an id of "bookid"
			var assumedISBN string
			if strings.EqualFold(ident.ID, "bookid") {
				assumedISBN = ident.Value
			} else {
				// let's see if value is 10 or 13 chars and looks like an ISBN
				assumedISBN = NormalizeISBN(ident.Value)
			}
			if assumedISBN != "" {
				if len(assumedISBN) == 10 {
					out.ISBN = assumedISBN
				} else if len(assumedISBN) == 13 {
					out.ISBN13 = assumedISBN
				}
			}
			// if we found something that looks like an ISBN, we can stop looking
			if out.ISBN != "" || out.ISBN13 != "" {
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
		coverImageURL, err := extractEPUBCoverDataURL(context.Background(), out.CoverImage, pathpkg.Join(out.Directory, out.FileName))
		if err != nil {
			slog.Warn("failed to extract cover image", slog.String(otelkeys.Error, err.Error()))
		} else {
			out.CoverImageURL = coverImageURL
		}
	}
}

func finishMOBI(out *ExifToolOutput) {

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
func NormalizeISBN(raw string) string {
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
