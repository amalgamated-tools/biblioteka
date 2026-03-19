package metadata

import (
	"context"
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

func TestExtractMetadata_NativeEPUB(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if !meta.IsNative {
		t.Error("expected IsNative to be true for EPUB")
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
}

func TestExtractMetadata_EPUBWithISBN10(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "Short Book", "Jane Doe", "isbn:0743273567")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Same goreader fix — ISBN-10 should now parse correctly
	if meta.ISBN != "0743273567" {
		t.Errorf("expected ISBN %q, got %q", "0743273567", meta.ISBN)
	}
}

func TestExtractMetadata_EPUBWithNoISBN(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "No ISBN Book", "Author", "some-random-uuid-1234")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
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

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(context.Background(), epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if !meta.IsNative {
		t.Error("expected .EPUB to use native parser")
	}
	if meta.Title != "Upper Case" {
		t.Errorf("expected title %q, got %q", "Upper Case", meta.Title)
	}
}

func TestExtractMetadata_PDF(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "test.pdf")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	testutils.MakeTestPDF(t, pdfPath, "PDF Title", "PDF Author", ext.exiftool())

	meta, err := ext.ExtractMetadata(context.Background(), pdfPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.IsNative {
		t.Error("expected IsNative to be false for PDF")
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
	os.WriteFile(badPath, []byte("not a real epub"), 0o644)

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	_, err = ext.ExtractMetadata(context.Background(), badPath)
	if err == nil {
		t.Fatal("expected error for invalid EPUB")
	}
}

func TestExtractMetadata_NonexistentFile(t *testing.T) {
	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	_, err = ext.ExtractMetadata(context.Background(), "/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
