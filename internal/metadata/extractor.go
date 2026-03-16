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

func (e *Extractor) ExtractMetadata(path string) (*BookMetadata, error) {
	ext := strings.ToLower(filepath.Ext(path))
	// 1. Try Native EPUB parsing first (Zero-dependency, Very Fast)
	if ext == ".epub" {
		return e.extractNativeEpub(path)
	}

	if e.et == nil {
		return nil, ErrExifToolUnavailable
	}
	return e.extractExif(path)
}

func (e *Extractor) extractNativeEpub(path string) (*BookMetadata, error) {
	rc, err := epub.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return nil, fmt.Errorf("epub file %s contains no rootfiles", path)
	}

	book := rc.Rootfiles[0]
	publicationDate := "Unknown"
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

func (e *Extractor) extractExif(path string) (*BookMetadata, error) {
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
		slog.WarnContext(context.Background(), "title not found in metadata, using filename as fallback", slog.String(otelkeys.Path, path))
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	author, err := book.GetString("Author")
	if err != nil {
		slog.WarnContext(context.Background(), "author not found in metadata", slog.String(otelkeys.Path, path))
		author = "Unknown"
	}
	isbn, err := book.GetString("ISBN")
	if err != nil {
		slog.WarnContext(context.Background(), "ISBN not found in metadata", slog.String(otelkeys.Path, path))
		isbn, err = book.GetString("Identifier") // Fallback for many MOBI files
		if err != nil {
			slog.WarnContext(context.Background(), "Identifier not found in metadata", slog.String(otelkeys.Path, path))
			isbn = "Not Found"
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

// findISBN searches the Identifier field for a valid ISBN pattern
func findISBN(book *epub.Rootfile) string {
	// In goreader v2, Identifier is a struct with Content and Scheme fields
	id := book.Identifier.Content

	// Strip common prefixes like "urn:isbn:" or "isbn:"
	id = strings.ToLower(id)
	id = strings.ReplaceAll(id, "urn:isbn:", "")
	id = strings.ReplaceAll(id, "isbn:", "")
	id = strings.TrimSpace(id)

	// A basic ISBN check: typically 10 or 13 digits (ignoring dashes)
	cleanID := strings.ReplaceAll(id, "-", "")
	if len(cleanID) == 10 || len(cleanID) == 13 {
		return cleanID
	}

	return "Not Found"
}
