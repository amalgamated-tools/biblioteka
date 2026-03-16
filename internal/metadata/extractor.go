package metadata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/barasher/go-exiftool"
	"github.com/taylorskalyo/goreader/epub"
)

var ErrExifToolUnavailable = errors.New("exiftool is not available on this system")

type BookMetadata struct {
	Author          string
	Description     string
	Format          string
	ISBN            string
	IsNative        bool
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

func NewExtractor() (*Extractor, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		slog.WarnContext(context.Background(), "exiftool not available; exif-based metadata extraction disabled", slog.Any(otelkeys.Error, err))
		return &Extractor{
			et: nil,
		}, nil
	}
	return &Extractor{
		et: et,
	}, nil
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
	ext := strings.ToLower(filepath.Ext(path))
	// 1. Try Native EPUB parsing first (Zero-dependency, Very Fast)
	if ext == ".epub" {
		return e.extractNativeEpub(ctx, path)
	}

	if e.et == nil {
		return nil, ErrExifToolUnavailable
	}
	return e.extractExif(ctx, path)
}

func (e *Extractor) extractNativeEpub(ctx context.Context, path string) (*BookMetadata, error) {
	slog.DebugContext(ctx, "extracting metadata via native EPUB parser", slog.String(otelkeys.Path, path))
	rc, err := epub.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return nil, fmt.Errorf("epub file %s contains no rootfiles", path)
	}

	book := rc.Rootfiles[0]
	var publicationDate string
	// if the book has a slice of Dates, we want to find one where the Event attribute is "publication"
	for _, d := range book.Dates {
		if d.Event == "publication" {
			publicationDate = d.Date
			break
		}
	}

	return &BookMetadata{
		Author:          book.Creator,
		Description:     book.Description,
		Format:          "EPUB",
		ISBN:            findISBN(book),
		IsNative:        true,
		Language:        book.Language,
		PublicationDate: publicationDate,
		Publisher:       book.Publisher,
		Title:           book.Title,
	}, nil
}

func (e *Extractor) extractExif(ctx context.Context, path string) (*BookMetadata, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	results := e.et.ExtractMetadata(path)
	if len(results) == 0 {
		return nil, fmt.Errorf("no metadata found for %s", path)
	}
	if results[0].Err != nil {
		return nil, fmt.Errorf("failed to extract metadata for %s: %w", path, results[0].Err)
	}
	book := results[0]
	// ExifTool normalization: mapping various tags to our struct
	title, err := book.GetString("Title")
	if err != nil {
		slog.WarnContext(ctx, "title not found in metadata, using filename as fallback", slog.String(otelkeys.Path, path))
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	author, err := book.GetString("Author")
	if err != nil {
		slog.WarnContext(ctx, "author not found in metadata", slog.String(otelkeys.Path, path))
		author = ""
	}
	isbn, err := book.GetString("ISBN")
	if err != nil {
		slog.WarnContext(ctx, "ISBN not found in metadata", slog.String(otelkeys.Path, path))
		isbn, err = book.GetString("Identifier") // Fallback for many MOBI files
		if err != nil {
			slog.WarnContext(ctx, "Identifier not found in metadata", slog.String(otelkeys.Path, path))
			isbn = ""
		}
	}

	return &BookMetadata{
		Title:    title,
		Author:   author,
		ISBN:     isbn,
		Format:   strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")),
		IsNative: false,
	}, nil
}

// NormalizeISBN strips common prefixes (urn:isbn:, isbn:), whitespace, hyphens,
// and spaces from a raw ISBN string. It returns the cleaned value only if it looks
// like a valid ISBN (10 or 13 digits); otherwise it returns "".
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
	if len(s) == 10 || len(s) == 13 {
		return s
	}
	return ""
}

// findISBN searches the Identifier field for a valid ISBN pattern
func findISBN(book *epub.Rootfile) string {
	return NormalizeISBN(book.Identifier.Content)
}
