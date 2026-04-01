package testutils

import (
	"fmt"
	"os"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
)

// MakeTestPDF creates a minimal valid PDF file with metadata written by exiftool.
func MakeTestPDF(t *testing.T, path, title, author string, et *exif.Exiftool) {
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

	if err := os.WriteFile(path, []byte(pdf), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	// Use exiftool to write proper metadata into the PDF. If exiftool is not
	// available in the environment, skip the test rather than failing.
	if et == nil {
		var exerr error
		et, exerr = exif.NewExiftool()
		if exerr != nil {
			t.Skipf("skipping PDF metadata test: exiftool not available: %v", exerr)
		}
		defer et.Close()
	}

	fm := exif.EmptyFileMetadata()
	fm.File = path
	fm.SetString("Title", title)
	fm.SetString("Author", author)
	metas := []exif.FileMetadata{fm}
	et.WriteMetadata(t.Context(), metas)
	if metas[0].Err != nil {
		t.Fatalf("write pdf metadata: %v", metas[0].Err)
	}
}
