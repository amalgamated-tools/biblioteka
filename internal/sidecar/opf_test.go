package sidecar

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteOPF_AllFields(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:          "The Great Gatsby",
		Author:         "F. Scott Fitzgerald",
		ISBN:           "9780743273565",
		Language:       "en",
		Date:           "1925-04-10",
		Publisher:      "Scribner",
		Description:    "A novel about the American Dream.",
		CoverFilename:  "cover.jpg",
		CoverMediaType: "image/jpeg",
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "WriteOPF")

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)

	checks := []string{
		`<dc:title>The Great Gatsby</dc:title>`,
		`<dc:creator opf:role="aut">F. Scott Fitzgerald</dc:creator>`,
		`opf:scheme="ISBN"`,
		`9780743273565`,
		`<dc:language>en</dc:language>`,
		`<dc:date>1925-04-10</dc:date>`,
		`<dc:publisher>Scribner</dc:publisher>`,
		`<dc:description>A novel about the American Dream.</dc:description>`,
		`name="cover"`,
		`content="cover-image"`,
		`href="cover.jpg"`,
		`media-type="image/jpeg"`,
	}

	for _, check := range checks {
		require.Contains(t, s, check, "metadata.opf missing %q", check)
	}

	// Verify it's valid XML.
	require.NoError(t, xml.Unmarshal(content, new(any)), "metadata.opf is not valid XML")
}

func TestWriteOPF_MinimalData(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Untitled",
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "WriteOPF")

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	require.Contains(t, s, `<dc:title>Untitled</dc:title>`, "metadata.opf missing title")
	require.Contains(t, s, `<dc:language>und</dc:language>`, "metadata.opf missing fallback language")
	// Should not contain empty elements for omitted fields.
	require.NotContains(t, s, `<dc:creator`, "metadata.opf should not contain empty creator")
	// Should always have a dc:identifier with UUID scheme when no ISBN.
	require.Contains(t, s, `opf:scheme="UUID"`, "metadata.opf missing UUID identifier")
	require.Contains(t, s, `id="uid"`, "metadata.opf missing id=uid on identifier")
	// Verify the identifier uses the urn:uuid: URN format required by OPF 2.0 §2.2.10.
	uuidRe := regexp.MustCompile(`urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	require.Regexp(t, uuidRe, s, "metadata.opf identifier does not contain a urn:uuid: URN")
}

func TestWriteOPF_EmptyTitle(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{}

	require.Error(t, WriteOPF(t.Context(), dir, data, ""))
}

func TestWriteOPF_UUIDIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:  "Determinism Test",
		Author: "Author A",
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "first WriteOPF")
	first, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read first metadata.opf")

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "second WriteOPF")
	second, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read second metadata.opf")

	uuidRe := regexp.MustCompile(`urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	uuid1 := uuidRe.FindString(string(first))
	uuid2 := uuidRe.FindString(string(second))
	require.NotEmpty(t, uuid1)
	require.NotEmpty(t, uuid2)
	require.Equal(t, uuid1, uuid2, "UUID changed between calls")
}

func TestWriteOPF_NoCover(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Test Book",
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "WriteOPF")

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	require.NotContains(t, s, `name="cover"`, "metadata.opf should not contain cover meta without cover")
	require.NotContains(t, s, `<manifest>`, "metadata.opf should not contain manifest without cover")
	// Identifier should still be present even without a cover.
	require.Contains(t, s, `<dc:identifier`, "metadata.opf missing dc:identifier")
}

func TestWriteOPF_PNGCover(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:          "PNG Cover Book",
		CoverFilename:  "cover.png",
		CoverMediaType: "image/png",
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "WriteOPF")

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	checks := []string{
		`href="cover.png"`,
		`media-type="image/png"`,
		`name="cover"`,
	}
	for _, check := range checks {
		require.Contains(t, s, check, "metadata.opf missing %q", check)
	}
}

func TestWriteOPF_XMLSpecialChars(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:       `"Quotes" & <Angles>`,
		Description: `A "special" <description> with & chars`,
	}

	require.NoError(t, WriteOPF(t.Context(), dir, data, ""), "WriteOPF")

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	// Verify it's valid XML (special chars properly escaped).
	require.NoError(t, xml.Unmarshal(content, new(any)), "metadata.opf is not valid XML")
}

func TestWriteOPF_InconsistentCoverFields(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		media    string
	}{
		{"filename without media type", "cover.jpg", ""},
		{"media type without filename", "", "image/jpeg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := OPFData{
				Title:          "Test",
				CoverFilename:  tc.filename,
				CoverMediaType: tc.media,
			}
			require.Error(t, WriteOPF(t.Context(), dir, data, ""), "expected error for inconsistent CoverFilename/CoverMediaType")
		})
	}
}

func TestWriteOPF_CustomBaseName(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Alice's Adventures in Wonderland",
	}

	baseName := "Alice's Adventures in Wonderland by Lewis Carroll"
	require.NoError(t, WriteOPF(t.Context(), dir, data, baseName), "WriteOPF")

	expectedFile := baseName + ".opf"
	_, err := os.Stat(filepath.Join(dir, expectedFile))
	require.NoError(t, err, "expected %q", expectedFile)
	// Default "metadata.opf" should NOT exist.
	_, err = os.Stat(filepath.Join(dir, "metadata.opf"))
	require.True(t, os.IsNotExist(err), "metadata.opf should not exist when using custom baseName")
}

func TestWriteOPF_InvalidBaseName(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{Title: "Unsafe"}

	require.Error(t, WriteOPF(t.Context(), dir, data, "../escape"))
}
