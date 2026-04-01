package exif

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	pathpkg "path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubCoverRef struct {
	Href     string
	MIMEType string
}

func extractEPUBCoverDataURL(ctx context.Context, book *ManifestItem, filePath string) (string, error) {
	ref, ok := findEPUBCoverRef(ctx, book)
	if !ok {
		slog.DebugContext(ctx, "no cover reference found in EPUB metadata", slog.String(otelkeys.Path, filePath))
		return "", nil
	}

	coverBytes, mimeType, err := readEPUBArchiveFile(ctx, filePath, ref)
	if err != nil {
		slog.WarnContext(
			ctx,
			"failed to read cover asset from EPUB archive",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.String(otelkeys.Error, err.Error()),
		)
		return "", err
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(coverBytes), nil
}

func findEPUBCoverRef(ctx context.Context, book *ManifestItem) (epubCoverRef, bool) {
	if book == nil {
		slog.DebugContext(ctx, "no manifest item provided for EPUB cover extraction")
		return epubCoverRef{}, false
	}

	// var firstImage *epubCoverRef
	ref := epubCoverRef{
		Href:     strings.TrimSpace(book.Href),
		MIMEType: strings.TrimSpace(book.MediaType),
	}

	return ref, true
}

func readEPUBArchiveFile(ctx context.Context, filePath string, ref epubCoverRef) ([]byte, string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		slog.WarnContext(
			ctx,
			"failed to open EPUB archive for cover extraction",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.String(otelkeys.Error, err.Error()),
		)
		return nil, "", fmt.Errorf("open epub archive: %w", err)
	}
	defer reader.Close()
	slog.DebugContext(
		ctx,
		"opened EPUB archive for cover extraction",
		slog.String(otelkeys.Path, filePath),
		slog.Int(otelkeys.FilesFound, len(reader.File)),
	)

	rootFilePath, rootErr := readEPUBRootFilePath(ctx, reader.File)
	candidates := archiveCandidates(ctx, rootFilePath, ref.Href)

	file := findArchiveFile(ctx, reader.File, candidates)
	if file == nil {
		if rootErr != nil {
			slog.WarnContext(
				ctx,
				"cover asset not found in EPUB archive",
				slog.String(otelkeys.Path, filePath),
				slog.String(otelkeys.CoverHref, ref.Href),
				slog.String(otelkeys.Error, rootErr.Error()),
			)
			return nil, "", fmt.Errorf("cover asset %q not found in archive (failed to determine EPUB root file path): %w", ref.Href, rootErr)
		}
		slog.WarnContext(
			ctx,
			"cover asset not found in EPUB archive",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
		)
		return nil, "", fmt.Errorf("cover asset %q not found in archive", ref.Href)
	}

	rc, err := file.Open()
	if err != nil {
		slog.WarnContext(
			ctx,
			"failed to open cover asset in EPUB archive",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.String(otelkeys.Error, err.Error()),
		)
		return nil, "", fmt.Errorf("open cover asset %q: %w", file.Name, err)
	}
	defer rc.Close()

	const maxCoverBytes = 20 << 20 // 20 MB
	coverBytes, err := io.ReadAll(io.LimitReader(rc, int64(maxCoverBytes)+1))
	if err != nil {
		slog.WarnContext(
			ctx,
			"failed to read cover asset from EPUB archive",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.String(otelkeys.Error, err.Error()),
		)
		return nil, "", fmt.Errorf("read cover asset %q: %w", file.Name, err)
	}
	if len(coverBytes) > maxCoverBytes {
		slog.WarnContext(
			ctx,
			"cover asset in EPUB archive exceeds size limit",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.Int(otelkeys.FileSize, len(coverBytes)),
			slog.Int(otelkeys.ErrorCode, http.StatusRequestEntityTooLarge),
		)
		return nil, "", fmt.Errorf("cover asset %q exceeds %d-byte limit", file.Name, maxCoverBytes)
	}

	mimeType := strings.ToLower(strings.TrimSpace(ref.MIMEType))
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = http.DetectContentType(coverBytes)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		slog.WarnContext(
			ctx,
			"cover asset in EPUB archive has non-image content type",
			slog.String(otelkeys.Path, filePath),
			slog.String(otelkeys.CoverHref, ref.Href),
			slog.String(otelkeys.MIMEType, mimeType),
		)

		return nil, "", fmt.Errorf("cover asset %q has non-image content type %q", file.Name, mimeType)
	}

	return coverBytes, mimeType, nil
}

func readEPUBRootFilePath(ctx context.Context, files []*zip.File) (string, error) {
	container := findArchiveFile(ctx, files, []string{"META-INF/container.xml"})
	if container == nil {
		slog.WarnContext(
			ctx,
			"container.xml not found in EPUB archive",
			slog.Int(otelkeys.FilesFound, len(files)),
		)
		return "", errors.New("container.xml not found")
	}

	rc, err := container.Open()
	if err != nil {
		slog.WarnContext(
			ctx,
			"failed to open container.xml in EPUB archive",
			slog.Int(otelkeys.FilesFound, len(files)),
			slog.String(otelkeys.Error, err.Error()),
		)
		return "", fmt.Errorf("open container.xml: %w", err)
	}
	defer rc.Close()

	const maxContainerXMLBytes = 1 << 20 // 1 MB
	var doc epubContainer
	if err := xml.NewDecoder(io.LimitReader(rc, maxContainerXMLBytes)).Decode(&doc); err != nil {
		slog.WarnContext(
			ctx,
			"failed to decode container.xml in EPUB archive",
			slog.Int(otelkeys.FilesFound, len(files)),
			slog.String(otelkeys.Error, err.Error()),
		)
		return "", fmt.Errorf("decode container.xml: %w", err)
	}
	if len(doc.Rootfiles) == 0 {
		slog.WarnContext(
			ctx,
			"no rootfile entries in container.xml",
			slog.Int(otelkeys.FilesFound, len(files)),
		)
		return "", errors.New("no rootfile entries in container.xml")
	}

	return cleanArchivePath(ctx, doc.Rootfiles[0].FullPath), nil
}

func archiveCandidates(ctx context.Context, rootFilePath, href string) []string {
	cleanHref := cleanArchivePath(ctx, href)
	if cleanHref == "" {
		slog.DebugContext(ctx, "cover href is empty after cleaning", slog.String(otelkeys.CoverHref, href))
		return nil
	}

	candidates := []string{cleanHref}
	if rootFilePath != "" {
		candidates = append([]string{cleanArchivePath(ctx, pathpkg.Join(pathpkg.Dir(rootFilePath), cleanHref))}, candidates...)
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

func findArchiveFile(ctx context.Context, files []*zip.File, candidates []string) *zip.File {
	for _, candidate := range candidates {
		for _, file := range files {
			name := cleanArchivePath(ctx, file.Name)
			if name == candidate {
				return file
			}
		}
	}
	return nil
}

func cleanArchivePath(ctx context.Context, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		slog.DebugContext(ctx, "archive path is empty after trimming", slog.String(otelkeys.Path, name))
		return ""
	}
	return strings.TrimPrefix(pathpkg.Clean(name), "./")
}
