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
)

var ErrExifToolUnavailable = errors.New("exiftool is not available on this system")

type BookMetadata struct {
	Author          string
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

func NewExtractor() (*Extractor, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		slog.WarnContext(context.Background(), "exiftool not available; all metadata extraction disabled — only filename-derived metadata will be used", slog.Any(otelkeys.Error, err))
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
	defer e.mu.Unlock()

	results := e.et.ExtractMetadata(path)
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

	return &BookMetadata{
		Title:           title,
		Author:          author,
		Description:     getStringOr(&book, "Description", ""),
		ISBN:            isbn,
		Format:          strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), ".")),
		Language:        getStringOr(&book, "Language", ""),
		PublicationDate: pubDate,
		Publisher:       getStringOr(&book, "Publisher", ""),
	}, nil
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
