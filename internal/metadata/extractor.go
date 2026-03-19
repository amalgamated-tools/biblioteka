package metadata

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/barasher/go-exiftool"
)

var ErrExifToolUnavailable = errors.New("exiftool is not available on this system")

type BookMetadata struct {
	Author          string
	CoverImageURL   string
	Description     string
	Format          string
	ISBN            string
	Language        string
	PublicationDate string
	Publisher       string
	Title           string
}

// Extractor extracts metadata from book files. Concurrent ExtractMetadata calls are safe,
// but Close must not be called concurrently with other methods.
type Extractor struct {
	mu sync.Mutex
	et *exiftool.Exiftool
}

func NewExtractor(ctx context.Context) (*Extractor, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		slog.WarnContext(ctx, "exiftool not available; all metadata extraction disabled — only filename-derived metadata will be used", slog.Any(otelkeys.Error, err))
		return &Extractor{}, nil
	}
	return &Extractor{et: et}, nil
}

func (e *Extractor) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.et != nil {
		e.et.Close()
		e.et = nil
	}
}

func (e *Extractor) ExtractMetadata(ctx context.Context, path string) (*BookMetadata, error) {
	if e.et == nil {
		return nil, ErrExifToolUnavailable
	}
	return e.extractExif(ctx, path)
}

func (e *Extractor) extractExif(ctx context.Context, path string) (*BookMetadata, error) {
	e.mu.Lock()
	results := e.et.ExtractMetadata(path)
	e.mu.Unlock()

	if len(results) == 0 {
		return nil, fmt.Errorf("no metadata found for %s", path)
	}
	if results[0].Err != nil {
		slog.WarnContext(ctx, "exiftool extraction error",
			slog.String(otelkeys.Path, path),
			slog.Any(otelkeys.Error, results[0].Err),
		)
		return nil, fmt.Errorf("failed to extract metadata for %s: %w", path, results[0].Err)
	}
	book := results[0]

	title := getStringOr(&book, "Title", "")
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// ExifTool uses "Author" for most formats but "Creator" for EPUBs.
	author := getStringOr(&book, "Author", "")
	if author == "" {
		author = getStringOr(&book, "Creator", "")
	}

	isbn := getStringOr(&book, "ISBN", "")
	if isbn == "" {
		isbn = getStringOr(&book, "Identifier", "")
	}
	isbn = NormalizeISBN(isbn)

	pubDate := getStringOr(&book, "PublicationDate", "")
	pubDate = normalizeExifDate(pubDate)

	coverImageURL := ""
	if strings.EqualFold(filepath.Ext(path), ".epub") {
		var err error
		coverImageURL, err = extractEPUBCoverDataURL(&book, path)
		if err != nil {
			slog.WarnContext(ctx, "failed to extract embedded EPUB cover image",
				slog.String(otelkeys.Path, path),
				slog.Any(otelkeys.Error, err),
			)
		}
	}

	return &BookMetadata{
		Title:           title,
		Author:          author,
		CoverImageURL:   coverImageURL,
		Description:     getStringOr(&book, "Description", ""),
		ISBN:            isbn,
		Format:          strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")),
		Language:        getStringOr(&book, "Language", ""),
		PublicationDate: pubDate,
		Publisher:       getStringOr(&book, "Publisher", ""),
	}, nil
}

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubCoverRef struct {
	Href     string
	MIMEType string
}

func extractEPUBCoverDataURL(book *exiftool.FileMetadata, filePath string) (string, error) {
	ref, ok := findEPUBCoverRef(book)
	if !ok {
		return "", nil
	}

	coverBytes, mimeType, err := readEPUBArchiveFile(filePath, ref)
	if err != nil {
		return "", err
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(coverBytes), nil
}

func findEPUBCoverRef(book *exiftool.FileMetadata) (epubCoverRef, bool) {
	manifestIDs, _ := book.GetStrings("ManifestItemId")
	manifestHrefs, _ := book.GetStrings("ManifestItemHref")
	manifestMIMETypes, _ := book.GetStrings("ManifestItemMedia-type")
	if len(manifestHrefs) == 0 {
		return epubCoverRef{}, false
	}

	coverID := ""
	metaNames, metaNameErr := book.GetStrings("MetaName")
	metaContents, metaContentErr := book.GetStrings("MetaContent")
	if metaNameErr == nil && metaContentErr == nil && len(metaNames) == len(metaContents) {
		for i, name := range metaNames {
			if strings.EqualFold(strings.TrimSpace(name), "cover") {
				coverID = strings.TrimSpace(metaContents[i])
				break
			}
		}
	}
	if coverID == "" && strings.EqualFold(strings.TrimSpace(getStringOr(book, "MetaName", "")), "cover") {
		coverID = strings.TrimSpace(getStringOr(book, "MetaContent", ""))
	}

	if coverID != "" {
		for i, href := range manifestHrefs {
			if strings.TrimSpace(itemAt(manifestIDs, i)) != coverID {
				continue
			}
			ref := epubCoverRef{
				Href:     strings.TrimSpace(href),
				MIMEType: strings.TrimSpace(itemAt(manifestMIMETypes, i)),
			}
			if ref.Href != "" && isLikelyImage(ref.Href, ref.MIMEType) {
				return ref, true
			}
		}
	}

	var firstImage *epubCoverRef
	imageCount := 0
	for i, href := range manifestHrefs {
		ref := epubCoverRef{
			Href:     strings.TrimSpace(href),
			MIMEType: strings.TrimSpace(itemAt(manifestMIMETypes, i)),
		}
		if ref.Href == "" || !isLikelyImage(ref.Href, ref.MIMEType) {
			continue
		}
		imageCount++
		if firstImage == nil {
			candidate := ref
			firstImage = &candidate
		}
		id := strings.ToLower(strings.TrimSpace(itemAt(manifestIDs, i)))
		lowerHref := strings.ToLower(pathpkg.Base(ref.Href))
		if strings.Contains(id, "cover") || strings.Contains(lowerHref, "cover") {
			return ref, true
		}
	}

	if firstImage != nil && imageCount == 1 {
		return *firstImage, true
	}

	return epubCoverRef{}, false
}

func readEPUBArchiveFile(filePath string, ref epubCoverRef) ([]byte, string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open epub archive: %w", err)
	}
	defer reader.Close()

	rootFilePath, rootErr := readEPUBRootFilePath(reader.File)
	candidates := archiveCandidates(rootFilePath, ref.Href)

	file := findArchiveFile(reader.File, candidates)
	if file == nil {
		if rootErr != nil {
			return nil, "", fmt.Errorf("cover asset %q not found in archive (failed to determine EPUB root file path): %w", ref.Href, rootErr)
		}
		return nil, "", fmt.Errorf("cover asset %q not found in archive", ref.Href)
	}

	rc, err := file.Open()
	if err != nil {
		return nil, "", fmt.Errorf("open cover asset %q: %w", file.Name, err)
	}
	defer rc.Close()

	const maxCoverBytes = 20 << 20 // 20 MB
	coverBytes, err := io.ReadAll(io.LimitReader(rc, int64(maxCoverBytes)+1))
	if err != nil {
		return nil, "", fmt.Errorf("read cover asset %q: %w", file.Name, err)
	}
	if len(coverBytes) > maxCoverBytes {
		return nil, "", fmt.Errorf("cover asset %q exceeds %d-byte limit", file.Name, maxCoverBytes)
	}

	mimeType := strings.TrimSpace(ref.MIMEType)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(coverBytes)
	mimeType := strings.TrimSpace(ref.MIMEType)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(coverBytes)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("cover asset %q has non-image content type %q", file.Name, mimeType)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("cover asset %q has non-image content type %q", file.Name, mimeType)
	}

	return coverBytes, mimeType, nil
}

func readEPUBRootFilePath(files []*zip.File) (string, error) {
	container := findArchiveFile(files, []string{"META-INF/container.xml"})
	if container == nil {
		return "", errors.New("container.xml not found")
	}

	rc, err := container.Open()
	if err != nil {
		return "", fmt.Errorf("open container.xml: %w", err)
	}
	defer rc.Close()

	const maxContainerXMLBytes = 1 << 20 // 1 MB
	var doc epubContainer
	if err := xml.NewDecoder(io.LimitReader(rc, maxContainerXMLBytes)).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode container.xml: %w", err)
	}
	if len(doc.Rootfiles) == 0 {
		return "", errors.New("no rootfile entries in container.xml")
	}

	return cleanArchivePath(doc.Rootfiles[0].FullPath), nil
}

func archiveCandidates(rootFilePath, href string) []string {
	cleanHref := cleanArchivePath(href)
	if cleanHref == "" {
		return nil
	}

	candidates := []string{cleanHref}
	if rootFilePath != "" {
		candidates = append([]string{cleanArchivePath(pathpkg.Join(pathpkg.Dir(rootFilePath), cleanHref))}, candidates...)
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func findArchiveFile(files []*zip.File, candidates []string) *zip.File {
	for _, candidate := range candidates {
		for _, file := range files {
			name := cleanArchivePath(file.Name)
			if name == candidate {
				return file
			}
		}
	}
	return nil
}

func cleanArchivePath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return strings.TrimPrefix(pathpkg.Clean(name), "./")
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

func itemAt(items []string, i int) string {
	if i < 0 || i >= len(items) {
		return ""
	}
	return items[i]
}

// getStringOr extracts a string tag from an exiftool result, returning fallback if not found.
func getStringOr(fm *exiftool.FileMetadata, tag string, fallback string) string {
	v, err := fm.GetString(tag)
	if err != nil {
		return fallback
	}
	return v
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
		for i := 0; i < 9; i++ {
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
		for i := 0; i < 13; i++ {
			if s[i] < '0' || s[i] > '9' {
				return ""
			}
		}
		return s
	default:
		return ""
	}
}
