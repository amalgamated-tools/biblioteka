package metadata

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/barasher/go-exiftool"
)

// makeTestEPUB creates a minimal valid EPUB file at the given path.
// The EPUB spec requires: mimetype, META-INF/container.xml, and a content.opf.
func makeTestEPUB(t *testing.T, path, title, creator, identifier string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// mimetype must be the first entry, stored (not compressed)
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := w.CreateHeader(mh)
	if err != nil {
		t.Fatalf("create mimetype: %v", err)
	}
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	writeZipFile(t, w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	// OEBPS/content.opf
	writeZipFile(t, w, "OEBPS/content.opf", `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>`+title+`</dc:title>
    <dc:creator>`+creator+`</dc:creator>
    <dc:identifier id="uid">`+identifier+`</dc:identifier>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`)

	// A minimal chapter so the EPUB isn't completely empty
	writeZipFile(t, w, "OEBPS/chapter1.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter 1</title></head><body><p>Hello</p></body></html>`)
}

func writeZipFile(t *testing.T, w *zip.Writer, name, content string) {
	t.Helper()
	fw, err := w.Create(name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// makeTestPDF creates a minimal valid PDF file with metadata written by exiftool.
func makeTestPDF(t *testing.T, path, title, author string) {
	t.Helper()

	// Build a structurally valid PDF with correct xref offsets.
	// Each object offset must exactly match what's in the xref table.
	obj1 := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	obj2 := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	obj3 := "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n"

	header := "%PDF-1.4\n"
	off1 := len(header)
	off2 := off1 + len(obj1)
	off3 := off2 + len(obj2)
	xrefOff := off3 + len(obj3)

	xref := fmt.Sprintf("xref\n0 4\n%010d 65535 f \n%010d 00000 n \n%010d 00000 n \n%010d 00000 n \n", 0, off1, off2, off3)
	trailer := "<< /Size 4 /Root 1 0 R >>\nstartxref\n"
	startxref := fmt.Sprintf("%d\n%%%%EOF\n", xrefOff)

	pdf := header + obj1 + obj2 + obj3 + xref + "trailer\n" + trailer + startxref

	if err := os.WriteFile(path, []byte(pdf), 0644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	// Use exiftool to write proper metadata into the PDF. If exiftool is not
	// available in the environment, skip the test rather than failing.
	et, err := exiftool.NewExiftool()
	if err != nil {
		t.Skipf("skipping PDF metadata test: exiftool not available: %v", err)
	}
	defer et.Close()

	fm := exiftool.EmptyFileMetadata()
	fm.File = path
	fm.SetString("Title", title)
	fm.SetString("Author", author)
	et.WriteMetadata([]exiftool.FileMetadata{fm})
	if fm.Err != nil {
		t.Fatalf("write pdf metadata: %v", fm.Err)
	}
}

func TestExtractMetadata_NativeEPUB(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	makeTestEPUB(t, epubPath, "The Great Gatsby", "F. Scott Fitzgerald", "urn:isbn:9780743273565")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(epubPath)
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
	makeTestEPUB(t, epubPath, "Short Book", "Jane Doe", "isbn:0743273567")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(epubPath)
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
	makeTestEPUB(t, epubPath, "No ISBN Book", "Author", "some-random-uuid-1234")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(epubPath)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if meta.ISBN != "Not Found" {
		t.Errorf("expected ISBN %q, got %q", "Not Found", meta.ISBN)
	}
}

func TestExtractMetadata_EPUBCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.EPUB")
	makeTestEPUB(t, epubPath, "Upper Case", "Author", "urn:isbn:9780743273565")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(epubPath)
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
	makeTestPDF(t, pdfPath, "PDF Title", "PDF Author")

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	meta, err := ext.ExtractMetadata(pdfPath)
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
	os.WriteFile(badPath, []byte("not a real epub"), 0644)

	ext, err := NewExtractor()
	if err != nil {
		t.Fatalf("new extractor: %v", err)
	}
	defer ext.Close()

	_, err = ext.ExtractMetadata(badPath)
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

	_, err = ext.ExtractMetadata("/nonexistent/file.pdf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
