package sidecar

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

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
		if !strings.Contains(s, check) {
			t.Errorf("metadata.opf missing %q\nContent:\n%s", check, s)
		}
	}

	// Verify it's valid XML.
	if err := xml.Unmarshal(content, new(any)); err != nil {
		t.Errorf("metadata.opf is not valid XML: %v", err)
	}
}

func TestWriteOPF_MinimalData(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Untitled",
	}

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	if !strings.Contains(s, `<dc:title>Untitled</dc:title>`) {
		t.Errorf("metadata.opf missing title\nContent:\n%s", s)
	}
	if !strings.Contains(s, `<dc:language>und</dc:language>`) {
		t.Errorf("metadata.opf missing fallback language\nContent:\n%s", s)
	}
	// Should not contain empty elements for omitted fields.
	if strings.Contains(s, `<dc:creator`) {
		t.Errorf("metadata.opf should not contain empty creator\nContent:\n%s", s)
	}
	// Should always have a dc:identifier with UUID scheme when no ISBN.
	if !strings.Contains(s, `opf:scheme="UUID"`) {
		t.Errorf("metadata.opf missing UUID identifier\nContent:\n%s", s)
	}
	if !strings.Contains(s, `id="uid"`) {
		t.Errorf("metadata.opf missing id=uid on identifier\nContent:\n%s", s)
	}
	// Verify the identifier uses the urn:uuid: URN format required by OPF 2.0 §2.2.10.
	uuidRe := regexp.MustCompile(`urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	if !uuidRe.MatchString(s) {
		t.Errorf("metadata.opf identifier does not contain a urn:uuid: URN\nContent:\n%s", s)
	}
}

func TestWriteOPF_EmptyTitle(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{}

	if err := WriteOPF(t.Context(), dir, data, ""); err == nil {
		require.Fail(t, "expected error when title is empty")
	}
}

func TestWriteOPF_UUIDIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:  "Determinism Test",
		Author: "Author A",
	}

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "first WriteOPF")
	}
	first, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read first metadata.opf")

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "second WriteOPF")
	}
	second, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read second metadata.opf")

	uuidRe := regexp.MustCompile(`urn:uuid:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	uuid1 := uuidRe.FindString(string(first))
	uuid2 := uuidRe.FindString(string(second))
	if uuid1 == "" || uuid2 == "" {
		require.Failf(t, "failed", "expected UUIDs in both OPF files, got %q and %q", uuid1, uuid2)
	}
	if uuid1 != uuid2 {
		t.Errorf("UUID changed between calls: %q vs %q", uuid1, uuid2)
	}
}

func TestWriteOPF_NoCover(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Test Book",
	}

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	if strings.Contains(s, `name="cover"`) {
		t.Errorf("metadata.opf should not contain cover meta without cover\nContent:\n%s", s)
	}
	if strings.Contains(s, `<manifest>`) {
		t.Errorf("metadata.opf should not contain manifest without cover\nContent:\n%s", s)
	}
	// Identifier should still be present even without a cover.
	if !strings.Contains(s, `<dc:identifier`) {
		t.Errorf("metadata.opf missing dc:identifier\nContent:\n%s", s)
	}
}

func TestWriteOPF_PNGCover(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:          "PNG Cover Book",
		CoverFilename:  "cover.png",
		CoverMediaType: "image/png",
	}

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	s := string(content)
	checks := []string{
		`href="cover.png"`,
		`media-type="image/png"`,
		`name="cover"`,
	}
	for _, check := range checks {
		if !strings.Contains(s, check) {
			t.Errorf("metadata.opf missing %q\nContent:\n%s", check, s)
		}
	}
}

func TestWriteOPF_XMLSpecialChars(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:       `"Quotes" & <Angles>`,
		Description: `A "special" <description> with & chars`,
	}

	if err := WriteOPF(t.Context(), dir, data, ""); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	require.NoError(t, err, "read metadata.opf")

	// Verify it's valid XML (special chars properly escaped).
	if err := xml.Unmarshal(content, new(any)); err != nil {
		t.Errorf("metadata.opf is not valid XML: %v\nContent:\n%s", err, string(content))
	}
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
			if err := WriteOPF(t.Context(), dir, data, ""); err == nil {
				t.Error("expected error for inconsistent CoverFilename/CoverMediaType, got nil")
			}
		})
	}
}

func TestWriteOPF_CustomBaseName(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title: "Alice's Adventures in Wonderland",
	}

	baseName := "Alice's Adventures in Wonderland by Lewis Carroll"
	if err := WriteOPF(t.Context(), dir, data, baseName); err != nil {
		require.NoError(t, err, "WriteOPF")
	}

	expectedFile := baseName + ".opf"
	if _, err := os.Stat(filepath.Join(dir, expectedFile)); err != nil {
		t.Errorf("expected %q, got error: %v", expectedFile, err)
	}
	// Default "metadata.opf" should NOT exist.
	if _, err := os.Stat(filepath.Join(dir, "metadata.opf")); !os.IsNotExist(err) {
		t.Errorf("metadata.opf should not exist when using custom baseName")
	}
}

func TestWriteOPF_InvalidBaseName(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{Title: "Unsafe"}

	if err := WriteOPF(t.Context(), dir, data, "../escape"); err == nil {
		require.Fail(t, "expected error for invalid base name")
	}
}
