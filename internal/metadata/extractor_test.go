package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	"github.com/barasher/go-exiftool"
)

// exiftool returns the underlying *exiftool.Exiftool instance.
// It is intended for use only within the metadata package (e.g., in tests).
func (e *Extractor) exiftool() *exiftool.Exiftool {
	return e.et
}

// requireExifTool creates an Extractor and skips the test if ExifTool is not installed.
func requireExifTool(t *testing.T) *Extractor {
	t.Helper()
	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	if ext.exiftool() == nil {
		t.Skip("exiftool not available, skipping")
	}
	return ext
}

func TestExtractMetadata_EPUB(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565", testutils.EPUBOptions{
		Description:     "A novel about the Jazz Age",
		Publisher:       "Scribner",
		PublicationDate: "1925-04-10",
		Language:        "en",
	})

	ext := requireExifTool(t)
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.Format != "EPUB" {
		t.Errorf("expected format EPUB, got %q", meta.Format)
	}
	if meta.Title != "The Great Gatsby" {
		t.Errorf("expected title %q, got %q", "The Great Gatsby", meta.Title)
	}
	if meta.Author != "F. Scott Fitzgerald" {
		t.Errorf("expected author %q, got %q", "F. Scott Fitzgerald", meta.Author)
	}
	if meta.ISBN != "9780743273565" {
		t.Errorf("expected ISBN %q, got %q", "9780743273565", meta.ISBN)
	}
	if meta.Description != "A novel about the Jazz Age" {
		t.Errorf("expected description %q, got %q", "A novel about the Jazz Age", meta.Description)
	}
	if meta.Publisher != "Scribner" {
		t.Errorf("expected publisher %q, got %q", "Scribner", meta.Publisher)
	}
	if meta.Language != "en" {
		t.Errorf("expected language %q, got %q", "en", meta.Language)
	}
	if meta.PublicationDate != "1925-04-10" {
		t.Errorf("expected publication date %q, got %q", "1925-04-10", meta.PublicationDate)
	}
}

func TestExtractMetadata_EPUBWithISBN10(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "Short Book", "Jane Doe", "isbn:0743273567")

	ext := requireExifTool(t)
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.ISBN != "0743273567" {
		t.Errorf("expected ISBN %q, got %q", "0743273567", meta.ISBN)
	}
}

func TestExtractMetadata_EPUBWithNoISBN(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "No ISBN Book", "Author", "some-random-uuid-1234")

	ext := requireExifTool(t)
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.ISBN != "" {
		t.Errorf("expected ISBN %q, got %q", "", meta.ISBN)
	}
}

func TestExtractMetadata_EPUBCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.EPUB")
	testutils.MakeTestEPUB(t, epubPath, "Upper Case", "Author", "urn:isbn:9780743273565")

	ext := requireExifTool(t)
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.Title != "Upper Case" {
		t.Errorf("expected title %q, got %q", "Upper Case", meta.Title)
	}
}

func TestExtractMetadata_PDF(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "test.pdf")

	ext := requireExifTool(t)
	defer ext.Close()

	testutils.MakeTestPDF(t, pdfPath, "PDF Title", "PDF Author", ext.exiftool())

	meta, err := ext.ExtractMetadata(context.Background(), pdfPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.Format != "PDF" {
		t.Errorf("expected format PDF, got %q", meta.Format)
	}
	if meta.Title != "PDF Title" {
		t.Errorf("expected title %q, got %q", "PDF Title", meta.Title)
	}
	if meta.Author != "PDF Author" {
		t.Errorf("expected author %q, got %q", "PDF Author", meta.Author)
	}
}

func TestExtractMetadata_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.epub")
	if err := os.WriteFile(badPath, []byte("not a real epub"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	ext := requireExifTool(t)
	defer ext.Close()

	// ExifTool processes the file without error — it just won't find book metadata.
	// Title falls back to the filename stem.
	meta, err := ext.ExtractMetadata(context.Background(), badPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "bad" {
		t.Errorf("expected title %q (filename fallback), got %q", "bad", meta.Title)
	}
}

func TestExtractMetadata_NonexistentFile(t *testing.T) {
	ext := requireExifTool(t)
	defer ext.Close()

	_, err := ext.ExtractMetadata(context.Background(), "/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestExtractMetadata_Unavailable(t *testing.T) {
	// An extractor with nil et should return ErrExifToolUnavailable.
	ext := &Extractor{}
	_, err := ext.ExtractMetadata(context.Background(), "/any/file.epub")
	if !errors.Is(err, ErrExifToolUnavailable) {
		t.Fatalf("expected ErrExifToolUnavailable, got %v", err)
	}
}

func TestNormalizeExifDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1925:04:10", "1925-04-10"},
		{"1925:04:10 00:00:00", "1925-04-10"},
		{"2021:01:15 10:20:30+05:00", "2021-01-15"},
		{"1925-04-10", "1925-04-10"}, // already normalized (no colons at 4,7)
		{"short", "short"},           // too short
		{"", ""},                     // empty
	}
	for _, tt := range tests {
		got := normalizeExifDate(tt.input)
		if got != tt.want {
			t.Errorf("normalizeExifDate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
