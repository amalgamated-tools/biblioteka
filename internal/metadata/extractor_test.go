package metadata

import (
	"encoding/base64"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/exif"
	"github.com/amalgamated-tools/biblioteka/internal/testutils"
	"github.com/stretchr/testify/require"
)

// exiftool returns the underlying *exiftool.Exiftool instance.
// It is intended for use only within the metadata package (e.g., in tests).
func (e *Extractor) exiftool() *exif.Exiftool {
	return e.et
}

// requireExifTool creates an Extractor and skips the test if ExifTool is not installed.
func requireExifTool(t *testing.T) *Extractor {
	t.Helper()
	ext, err := NewExtractor(t.Context())
	require.NoError(t, err, "new extractor")
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
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	if meta.Format != "epub" {
		t.Errorf("expected format epub, got %q", meta.Format)
	}
	if meta.Title != "The Great Gatsby" {
		t.Errorf("expected title %q, got %q", "The Great Gatsby", meta.Title)
	}
	if meta.Author != "F. Scott Fitzgerald" {
		t.Errorf("expected author %q, got %q", "F. Scott Fitzgerald", meta.Author)
	}
	if meta.ISBN13 != "9780743273565" {
		t.Errorf("expected ISBN %q, got %q", "9780743273565", meta.ISBN13)
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
	if meta.CoverImageURL != "" {
		t.Errorf("expected empty cover image url, got %q", meta.CoverImageURL)
	}
}

func TestExtractMetadata_EPUBCoverImage(t *testing.T) {
	testPNG := testutils.TinyPNG()
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "cover.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "Cover Test", "Author", "urn:isbn:9780000000000", testutils.EPUBOptions{
		CoverImageData: testPNG,
		CoverImageHref: "images/cover.png",
		CoverMediaType: "image/png",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	if !strings.HasPrefix(meta.CoverImageURL, "data:image/png;base64,") {
		require.Failf(t, "failed", "expected PNG data URL, got %q", meta.CoverImageURL)
	}

	b64 := strings.TrimPrefix(meta.CoverImageURL, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	require.Equal(t, testPNG, imgBytes, "cover image bytes should match original")
}

func TestExtractMetadata_EPUBOversizedCover(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "large-cover.epub")

	// Create an EPUB whose cover image exceeds the 20 MB cap.
	oversized := make([]byte, 20<<20+1) // 20 MB + 1 byte
	testutils.MakeTestEPUBWithOptions(t, epubPath, "Big Cover", "Author", "urn:isbn:0000000000", testutils.EPUBOptions{
		CoverImageData: oversized,
		CoverImageHref: "images/cover.png",
		CoverMediaType: "image/png",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	// The oversized cover should be silently skipped, not cause an extraction error.
	if meta.CoverImageURL != "" {
		t.Errorf("expected empty cover URL for oversized cover, got %d-byte data URL", len(meta.CoverImageURL))
	}
}

func TestExtractMetadata_EPUBWithISBN10(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "Short Book", "Jane Doe", "isbn:0743273567")

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	if meta.ISBN10 != "0743273567" {
		t.Errorf("expected ISBN %q, got %q", "0743273567", meta.ISBN10)
	}
}

func TestExtractMetadata_EPUBWithNoISBN(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	testutils.MakeTestEPUB(t, epubPath, "No ISBN Book", "Author", "some-random-uuid-1234")

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	if meta.ISBN() != "" {
		t.Errorf("expected ISBN %q, got %q", "", meta.ISBN())
	}
}

func TestExtractMetadata_EPUBCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.EPUB")
	testutils.MakeTestEPUB(t, epubPath, "Upper Case", "Author", "urn:isbn:9780743273565")

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err, "extract")

	if meta.Title != "Upper Case" {
		t.Errorf("expected title %q, got %q", "Upper Case", meta.Title)
	}
}

// --- EPUB3-specific integration tests ---

func TestExtractMetadata_EPUB3CoverViaProperties(t *testing.T) {
	// EPUB3 uses properties="cover-image" on the manifest item instead of
	// the EPUB2 <meta name="cover" content="..."/> element.
	testPNG := testutils.TinyPNG()
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3-cover.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB3 Cover", "Author", "urn:isbn:9780000000001", testutils.EPUBOptions{
		Version:        "3.0",
		EPUB3Cover:     true,
		CoverImageData: testPNG,
		CoverImageHref: "images/cover.png",
		CoverMediaType: "image/png",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(meta.CoverImageURL, "data:image/png;base64,"),
		"EPUB3 cover via properties should be extracted as data URL, got %q", meta.CoverImageURL)

	b64 := strings.TrimPrefix(meta.CoverImageURL, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	require.Equal(t, testPNG, imgBytes, "cover image bytes should match original")
}

func TestExtractMetadata_EPUB2CoverViaMeta(t *testing.T) {
	// EPUB2 uses <meta name="cover" content="cover-image"/> referencing a manifest item id.
	testPNG := testutils.TinyPNG()
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub2-cover.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB2 Cover", "Author", "urn:isbn:9780000000002", testutils.EPUBOptions{
		Version:        "2.0",
		EPUB3Cover:     false,
		CoverImageData: testPNG,
		CoverImageHref: "images/cover.png",
		CoverMediaType: "image/png",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(meta.CoverImageURL, "data:image/png;base64,"),
		"EPUB2 cover via meta tag should be extracted as data URL, got %q", meta.CoverImageURL)

	b64 := strings.TrimPrefix(meta.CoverImageURL, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	require.Equal(t, testPNG, imgBytes, "cover image bytes should match original")
}

func TestExtractMetadata_EPUB3Metadata(t *testing.T) {
	// EPUB3 uses plain <dc:date> instead of <dc:date opf:event="publication">.
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3-meta.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB3 Book", "Jane Author", "urn:isbn:9780000000003", testutils.EPUBOptions{
		Version:         "3.0",
		Description:     "An EPUB3 book with modern metadata",
		Publisher:       "Modern Press",
		PublicationDate: "2023-06-15",
		Language:        "en",
		Subjects:        []string{"Fiction", "Science Fiction"},
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.Equal(t, "epub", meta.Format)
	require.Equal(t, "EPUB3 Book", meta.Title)
	require.Equal(t, "Jane Author", meta.Author)
	require.Equal(t, "9780000000003", meta.ISBN13)
	require.Equal(t, "An EPUB3 book with modern metadata", meta.Description)
	require.Equal(t, "Modern Press", meta.Publisher)
	require.Equal(t, "2023-06-15", meta.PublicationDate)
	require.Equal(t, "en", meta.Language)
	require.Contains(t, meta.Subjects, "Fiction")
	require.Contains(t, meta.Subjects, "Science Fiction")
}

func TestExtractMetadata_EPUB2Metadata(t *testing.T) {
	// EPUB2 uses <dc:date opf:event="publication"> and <meta name="cover">.
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub2-meta.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "EPUB2 Book", "Classic Author", "urn:isbn:9780000000004", testutils.EPUBOptions{
		Version:         "2.0",
		Description:     "An EPUB2 book with traditional metadata",
		Publisher:       "Classic Press",
		PublicationDate: "1999-01-01",
		Language:        "en",
		Subjects:        []string{"History", "Biography"},
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.Equal(t, "epub", meta.Format)
	require.Equal(t, "EPUB2 Book", meta.Title)
	require.Equal(t, "Classic Author", meta.Author)
	require.Equal(t, "9780000000004", meta.ISBN13)
	require.Equal(t, "An EPUB2 book with traditional metadata", meta.Description)
	require.Equal(t, "Classic Press", meta.Publisher)
	require.Equal(t, "1999-01-01", meta.PublicationDate)
	require.Equal(t, "en", meta.Language)
	require.Contains(t, meta.Subjects, "History")
	require.Contains(t, meta.Subjects, "Biography")
}

func TestExtractMetadata_EPUB3WithMultipleSubjects(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "subjects.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "Many Subjects", "Author", "some-id", testutils.EPUBOptions{
		Version:  "3.0",
		Subjects: []string{"Fantasy", "Adventure", "Young Adult", "Magic"},
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.Contains(t, meta.Subjects, "Fantasy")
	require.Contains(t, meta.Subjects, "Adventure")
	require.Contains(t, meta.Subjects, "Young Adult")
	require.Contains(t, meta.Subjects, "Magic")
}

func TestExtractMetadata_EPUB3NoCover(t *testing.T) {
	// EPUB3 with no cover image at all should not error.
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "no-cover.epub")
	testutils.MakeTestEPUBWithOptions(t, epubPath, "No Cover EPUB3", "Author", "urn:isbn:9780000000005", testutils.EPUBOptions{
		Version: "3.0",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), epubPath)
	require.NoError(t, err)
	require.Equal(t, "No Cover EPUB3", meta.Title)
	require.Empty(t, meta.CoverImageURL, "EPUB3 with no cover should have empty cover URL")
}

func TestExtractMetadata_EPUB2And3ProduceSameMetadata(t *testing.T) {
	// Both EPUB versions should extract the same core metadata when given
	// equivalent content. This verifies version-agnostic extraction.
	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	dir := t.TempDir()

	epub2Path := filepath.Join(dir, "v2.epub")
	testutils.MakeTestEPUBWithOptions(t, epub2Path, "Same Book", "Same Author", "urn:isbn:9780000000006", testutils.EPUBOptions{
		Version:         "2.0",
		Description:     "Same description",
		Publisher:       "Same Publisher",
		PublicationDate: "2020-01-01",
		Language:        "fr",
	})

	epub3Path := filepath.Join(dir, "v3.epub")
	testutils.MakeTestEPUBWithOptions(t, epub3Path, "Same Book", "Same Author", "urn:isbn:9780000000006", testutils.EPUBOptions{
		Version:         "3.0",
		Description:     "Same description",
		Publisher:       "Same Publisher",
		PublicationDate: "2020-01-01",
		Language:        "fr",
	})

	meta2, err := ext.ExtractMetadata(t.Context(), epub2Path)
	require.NoError(t, err)

	meta3, err := ext.ExtractMetadata(t.Context(), epub3Path)
	require.NoError(t, err)

	require.Equal(t, meta2.Title, meta3.Title, "title")
	require.Equal(t, meta2.Author, meta3.Author, "author")
	require.Equal(t, meta2.ISBN13, meta3.ISBN13, "ISBN13")
	require.Equal(t, meta2.Description, meta3.Description, "description")
	require.Equal(t, meta2.Publisher, meta3.Publisher, "publisher")
	require.Equal(t, meta2.Language, meta3.Language, "language")
	require.Equal(t, meta2.PublicationDate, meta3.PublicationDate, "publication date")
	require.Equal(t, meta2.Format, meta3.Format, "format should be 'epub' for both versions")
}

func TestExtractMetadata_PDF(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "test.pdf")

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	testutils.MakeTestPDF(t, pdfPath, "PDF Title", "PDF Author", ext.exiftool())

	meta, err := ext.ExtractMetadata(t.Context(), pdfPath)
	require.NoError(t, err, "extract")

	if meta.Format != "pdf" {
		t.Errorf("expected format pdf, got %q", meta.Format)
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
		require.NoError(t, err, "write test file")
	}

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	// ExifTool processes the file without error — it just won't find book metadata.
	// Title falls back to the filename stem.
	meta, err := ext.ExtractMetadata(t.Context(), badPath)
	require.NoError(t, err)
	if meta.Title != "bad" {
		t.Errorf("expected title %q (filename fallback), got %q", "bad", meta.Title)
	}
}

func TestExtractMetadata_NonexistentFile(t *testing.T) {
	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	_, err := ext.ExtractMetadata(t.Context(), "/nonexistent/file.pdf")
	require.Error(t, err, "expected error for nonexistent file")
}

func TestExtractMetadata_Unavailable(t *testing.T) {
	// An extractor with nil et should return ErrExifToolUnavailable.
	ext := &Extractor{}
	_, err := ext.ExtractMetadata(t.Context(), "/any/file.epub")
	require.ErrorIs(t, err, ErrExifToolUnavailable)
}

func TestNormalizeISBN(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"9780306406157", "9780306406157"},              // plain ISBN-13
		{"0-306-40615-2", "0306406152"},                 // ISBN-10 with hyphens
		{"978-0-306-40615-7", "9780306406157"},          // ISBN-13 with hyphens
		{"ISBN:978-0-306-40615-7", "9780306406157"},     // with ISBN: prefix
		{"urn:isbn:978-0-306-40615-7", "9780306406157"}, // with urn:isbn: prefix
		{"  978-0-306-40615-7  ", "9780306406157"},      // leading/trailing spaces
		{"155860832X", "155860832X"},                    // ISBN-10 with X check digit
		{"155860832x", "155860832X"},                    // lowercase x normalized
		{"-\t1234567890", "1234567890"},                 // tab exposed after hyphen removal
		{"\n978-0-306-40615-7\n", "9780306406157"},      // newlines around input
		{"", ""},                  // empty
		{"tooshort", ""},          // invalid length
		{"978-0-306-40615-!", ""}, // invalid character
	}
	for _, tt := range tests {
		got := NormalizeISBN(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeISBN(%q) = %q, want %q", tt.input, got, tt.want)
		}
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
