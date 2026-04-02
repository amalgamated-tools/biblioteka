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
		// ExifTool is unavailable. For EPUB files, fall back to the native ZIP/OPF parser
		// so metadata can still be extracted without ExifTool.
		if strings.EqualFold(filepath.Ext(path), ".epub") {
			return extractEPUBNative(ctx, path)
		}
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

// epubOPFPackage represents the parsed OPF document of an EPUB file.
type epubOPFPackage struct {
	Metadata epubOPFMetadata `xml:"metadata"`
	Manifest epubOPFManifest `xml:"manifest"`
}

// epubOPFMetadata holds Dublin Core metadata fields from an EPUB OPF document.
type epubOPFMetadata struct {
	Titles       []string      `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creators     []string      `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Identifiers  []string      `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Languages    []string      `xml:"http://purl.org/dc/elements/1.1/ language"`
	Descriptions []string      `xml:"http://purl.org/dc/elements/1.1/ description"`
	Publishers   []string      `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Dates        []epubOPFDate `xml:"http://purl.org/dc/elements/1.1/ date"`
	Meta         []epubOPFMeta `xml:"meta"`
}

// epubOPFDate represents a dc:date element, optionally carrying an opf:event attribute.
type epubOPFDate struct {
	Event string `xml:"http://www.idpf.org/2007/opf event,attr"`
	Value string `xml:",chardata"`
}

// epubOPFMeta represents a <meta name="..." content="..."> element in OPF metadata.
type epubOPFMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

// epubOPFManifest lists all items declared in the EPUB manifest.
type epubOPFManifest struct {
	Items []epubOPFManifestItem `xml:"item"`
}

// epubOPFManifestItem represents a single item in the EPUB manifest.
type epubOPFManifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// parseEPUBOPF decodes an OPF XML document from r.
func parseEPUBOPF(r io.Reader) (*epubOPFPackage, error) {
	const maxOPFBytes = 10 << 20 // 10 MB
	var pkg epubOPFPackage
	if err := xml.NewDecoder(io.LimitReader(r, maxOPFBytes)).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("decode OPF: %w", err)
	}
	return &pkg, nil
}

// extractEPUBNative extracts book metadata directly from an EPUB ZIP/OPF file
// without ExifTool, by parsing the OPF XML from the archive.
func extractEPUBNative(ctx context.Context, path string) (*BookMetadata, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open epub archive: %w", err)
	}
	defer reader.Close()

	rootFilePath, err := readEPUBRootFilePath(reader.File)
	if err != nil {
		return nil, fmt.Errorf("find OPF file: %w", err)
	}

	opfFile := findArchiveFile(reader.File, []string{rootFilePath})
	if opfFile == nil {
		return nil, fmt.Errorf("OPF file %q not found in archive", rootFilePath)
	}

	rc, err := opfFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open OPF file: %w", err)
	}
	defer rc.Close()

	pkg, err := parseEPUBOPF(rc)
	if err != nil {
		return nil, err
	}

	title := firstNonEmpty(pkg.Metadata.Titles)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	author := firstNonEmpty(pkg.Metadata.Creators)

	isbn := ""
	for _, id := range pkg.Metadata.Identifiers {
		if n := NormalizeISBN(id); n != "" {
			isbn = n
			break
		}
	}

	language := firstNonEmpty(pkg.Metadata.Languages)
	description := firstNonEmpty(pkg.Metadata.Descriptions)
	publisher := firstNonEmpty(pkg.Metadata.Publishers)

	pubDate := ""
	for _, d := range pkg.Metadata.Dates {
		if strings.EqualFold(strings.TrimSpace(d.Event), "publication") {
			pubDate = strings.TrimSpace(d.Value)
			break
		}
	}
	if pubDate == "" && len(pkg.Metadata.Dates) > 0 {
		pubDate = strings.TrimSpace(pkg.Metadata.Dates[0].Value)
	}

	// Extract cover image if present.
	coverImageURL := ""
	coverRef := findEPUBCoverRefNative(ctx, pkg)
	if coverRef.Href != "" {
		coverBytes, mimeType, coverErr := readEPUBArchiveFile(ctx, path, coverRef)
		if coverErr != nil {
			slog.WarnContext(ctx, "failed to extract embedded EPUB cover image",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, coverErr),
			)
		} else {
			coverImageURL = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(coverBytes)
		}
	}

	slog.DebugContext(ctx, "epub metadata extracted natively",
		slog.String(otelkeys.Path, path),
		slog.String(otelkeys.Title, title),
	)

	return &BookMetadata{
		Title:           title,
		Author:          author,
		CoverImageURL:   coverImageURL,
		Description:     description,
		ISBN:            isbn,
		Format:          "EPUB",
		Language:        language,
		PublicationDate: pubDate,
		Publisher:       publisher,
	}, nil
}

// findEPUBCoverRefNative finds the cover image reference from a parsed OPF document.
func findEPUBCoverRefNative(ctx context.Context, pkg *epubOPFPackage) epubCoverRef {
	// Look for <meta name="cover" content="..."> to get the cover item ID.
	coverID := ""
	for _, m := range pkg.Metadata.Meta {
		if strings.EqualFold(strings.TrimSpace(m.Name), "cover") {
			coverID = strings.TrimSpace(m.Content)
			break
		}
	}

	if coverID != "" {
		for _, item := range pkg.Manifest.Items {
			if strings.TrimSpace(item.ID) == coverID {
				ref := epubCoverRef{
					Href:     strings.TrimSpace(item.Href),
					MIMEType: strings.TrimSpace(item.MediaType),
				}
				if ref.Href != "" && isLikelyImage(ref.Href, ref.MIMEType) {
					return ref
				}
			}
		}
		slog.WarnContext(ctx, "EPUB cover meta tag references unknown or non-image manifest item",
			slog.String(otelkeys.CoverID, coverID),
		)
	}

	// Fallback: look for the first image or an image with "cover" in its id/href.
	var firstImage *epubCoverRef
	imageCount := 0
	for _, item := range pkg.Manifest.Items {
		ref := epubCoverRef{
			Href:     strings.TrimSpace(item.Href),
			MIMEType: strings.TrimSpace(item.MediaType),
		}
		if ref.Href == "" || !isLikelyImage(ref.Href, ref.MIMEType) {
			continue
		}
		imageCount++
		if firstImage == nil {
			candidate := ref
			firstImage = &candidate
		}
		id := strings.ToLower(strings.TrimSpace(item.ID))
		lowerHref := strings.ToLower(pathpkg.Base(ref.Href))
		if strings.Contains(id, "cover") || strings.Contains(lowerHref, "cover") {
			return ref
		}
	}

	if firstImage != nil && imageCount == 1 {
		return *firstImage
	}

	return epubCoverRef{}
}

// firstNonEmpty returns the first non-empty trimmed string from ss, or "" if none.
func firstNonEmpty(ss []string) string {
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

// NormalizeISBN strips common prefixes (urn:isbn:, isbn:), whitespace, hyphens,
// and spaces from a raw ISBN string. It returns the cleaned value only if it looks
// like an ISBN-10 or ISBN-13; otherwise it returns "".
func NormalizeISBN(raw string) string {
	return exif.NormalizeISBN(raw)
}
