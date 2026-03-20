package sidecar

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteOPF_AllFields(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:       "The Great Gatsby",
		Author:      "F. Scott Fitzgerald",
		ISBN:        "9780743273565",
		Language:    "en",
		Date:        "1925-04-10",
		Publisher:   "Scribner",
		Description: "A novel about the American Dream.",
		HasCover:    true,
	}

	if err := WriteOPF(dir, data); err != nil {
		t.Fatalf("WriteOPF: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	if err != nil {
		t.Fatalf("read metadata.opf: %v", err)
	}

	s := string(content)

	checks := []string{
		`<dc:title>The Great Gatsby</dc:title>`,
		`<dc:creator>F. Scott Fitzgerald</dc:creator>`,
		`9780743273565`,
		`<dc:language>en</dc:language>`,
		`<dc:date>1925-04-10</dc:date>`,
		`<dc:publisher>Scribner</dc:publisher>`,
		`<dc:description>A novel about the American Dream.</dc:description>`,
		`name="cover"`,
		`content="cover-image"`,
		`href="cover.jpg"`,
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

	if err := WriteOPF(dir, data); err != nil {
		t.Fatalf("WriteOPF: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	if err != nil {
		t.Fatalf("read metadata.opf: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, `<dc:title>Untitled</dc:title>`) {
		t.Errorf("metadata.opf missing title\nContent:\n%s", s)
	}
	// Should not contain empty elements for omitted fields.
	if strings.Contains(s, `<dc:creator>`) {
		t.Errorf("metadata.opf should not contain empty creator\nContent:\n%s", s)
	}
}

func TestWriteOPF_NoCover(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:    "Test Book",
		HasCover: false,
	}

	if err := WriteOPF(dir, data); err != nil {
		t.Fatalf("WriteOPF: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	if err != nil {
		t.Fatalf("read metadata.opf: %v", err)
	}

	s := string(content)
	if strings.Contains(s, `name="cover"`) {
		t.Errorf("metadata.opf should not contain cover meta when HasCover is false\nContent:\n%s", s)
	}
	if strings.Contains(s, `<manifest>`) {
		t.Errorf("metadata.opf should not contain manifest when HasCover is false\nContent:\n%s", s)
	}
}

func TestWriteOPF_XMLSpecialChars(t *testing.T) {
	dir := t.TempDir()
	data := OPFData{
		Title:       `"Quotes" & <Angles>`,
		Description: `A "special" <description> with & chars`,
	}

	if err := WriteOPF(dir, data); err != nil {
		t.Fatalf("WriteOPF: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "metadata.opf"))
	if err != nil {
		t.Fatalf("read metadata.opf: %v", err)
	}

	// Verify it's valid XML (special chars properly escaped).
	if err := xml.Unmarshal(content, new(any)); err != nil {
		t.Errorf("metadata.opf is not valid XML: %v\nContent:\n%s", err, string(content))
	}
}
