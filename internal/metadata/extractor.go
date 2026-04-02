package metadata

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	pathpkg "path"
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

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubCoverRef struct {
	Href     string
	MIMEType string
}

func readEPUBArchiveFile(ctx context.Context, filePath string, ref epubCoverRef) ([]byte, string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("open epub archive: %w", err)
	}
	defer reader.Close()
	slog.DebugContext(
		ctx,
		"opened EPUB archive for cover extraction",
		slog.String(otelkeys.Path, filePath),
		slog.Int(otelkeys.FilesFound, len(reader.File)),
	)

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

	mimeType := strings.ToLower(strings.TrimSpace(ref.MIMEType))
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(coverBytes)
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
