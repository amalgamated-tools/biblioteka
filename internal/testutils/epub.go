package testutils

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	pathpkg "path"
	"strings"
	"testing"
)

// MakeTestEPUB creates a minimal valid EPUB file at the given path.
// The EPUB spec requires: mimetype, META-INF/container.xml, and a content.opf.
func MakeTestEPUB(t *testing.T, path, title, creator, identifier string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}

	w := zip.NewWriter(f)
	defer func() {
		var errs []string

		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close epub zip writer: %v", err))
		}
		if err := f.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close epub file: %v", err))
		}

		if len(errs) > 0 {
			t.Fatalf("%s", strings.Join(errs, "; "))
		}
	}()

	// mimetype must be the first entry, stored (not compressed)
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := w.CreateHeader(mh)
	if err != nil {
		t.Fatalf("create mimetype: %v", err)
	}
	if _, err := mw.Write([]byte("application/epub+zip")); err != nil {
		t.Fatalf("write mimetype: %v", err)
	}

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

// EPUBOptions provides optional metadata fields for MakeTestEPUBWithOptions.
type EPUBOptions struct {
	Description     string
	Publisher       string
	PublicationDate string // e.g. "1925-04-10"
	Language        string // defaults to "en" if empty
	CoverImageData  []byte
	CoverImageHref  string
	CoverMediaType  string
	// Version sets the OPF package version attribute ("2.0" or "3.0"). Defaults to "3.0".
	Version string
	// EPUB3Cover uses the EPUB3 properties="cover-image" attribute on the manifest
	// item instead of the EPUB2 <meta name="cover" content="..."/> element.
	EPUB3Cover bool
	// Subjects adds separate <dc:subject> elements for each entry.
	Subjects []string
}

var tinyPNG []byte

func init() {
	var err error
	tinyPNG, err = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+lmRcAAAAASUVORK5CYII=")
	if err != nil {
		panic("testutils: invalid tinyPNG base64 constant: " + err.Error())
	}
}

// TinyPNG returns a copy of a 1×1 pixel white PNG image as raw bytes.
// It is useful in tests that need a valid PNG without caring about its content.
func TinyPNG() []byte {
	return append([]byte(nil), tinyPNG...)
}

// MakeTestEPUBWithOptions creates a minimal valid EPUB file with full metadata control.
func MakeTestEPUBWithOptions(t *testing.T, path, title, creator, identifier string, opts EPUBOptions) {
	t.Helper()

	lang := opts.Language
	if lang == "" {
		lang = "en"
	}

	version := opts.Version
	if version == "" {
		version = "3.0"
	}
	switch version {
	case "2.0", "3.0":
	default:
		t.Fatalf(`unsupported EPUB version %q; supported versions are "2.0" and "3.0"`, version)
	}

	if opts.EPUB3Cover && version == "2.0" {
		t.Fatalf("EPUB3Cover requires Version %q but got %q; properties='cover-image' is an EPUB3-only mechanism", "3.0", version)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}

	w := zip.NewWriter(f)
	defer func() {
		var errs []string
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close epub zip writer: %v", err))
		}
		if err := f.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("close epub file: %v", err))
		}
		if len(errs) > 0 {
			t.Fatalf("%s", strings.Join(errs, "; "))
		}
	}()

	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := w.CreateHeader(mh)
	if err != nil {
		t.Fatalf("create mimetype: %v", err)
	}
	if _, err := mw.Write([]byte("application/epub+zip")); err != nil {
		t.Fatalf("write mimetype: %v", err)
	}

	writeZipFile(t, w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	var extraMeta string
	if opts.Description != "" {
		extraMeta += "\n    <dc:description>" + xmlEscape(opts.Description) + "</dc:description>"
	}
	if opts.Publisher != "" {
		extraMeta += "\n    <dc:publisher>" + xmlEscape(opts.Publisher) + "</dc:publisher>"
	}
	if opts.PublicationDate != "" {
		if version == "2.0" {
			extraMeta += `
    <dc:date opf:event="publication">` + xmlEscape(opts.PublicationDate) + `</dc:date>`
		} else {
			extraMeta += `
    <dc:date>` + xmlEscape(opts.PublicationDate) + `</dc:date>`
		}
	}

	for _, subject := range opts.Subjects {
		extraMeta += "\n    <dc:subject>" + xmlEscape(subject) + "</dc:subject>"
	}

	escapedTitle := xmlEscape(title)
	escapedCreator := xmlEscape(creator)
	escapedIdentifier := xmlEscape(identifier)
	escapedLang := xmlEscape(lang)

	manifestItems := []string{
		`    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>`,
	}

	if len(opts.CoverImageData) > 0 {
		coverHref := opts.CoverImageHref
		if coverHref == "" {
			coverHref = "images/cover.png"
		}
		coverMediaType := opts.CoverMediaType
		if coverMediaType == "" {
			coverMediaType = "image/png"
		}
		if opts.EPUB3Cover {
			// EPUB3 cover: properties="cover-image" on manifest item, no <meta> tag.
			manifestItems = append([]string{
				`    <item id="cover-image" href="` + xmlEscape(coverHref) + `" media-type="` + xmlEscape(coverMediaType) + `" properties="cover-image"/>`,
			}, manifestItems...)
		} else {
			// EPUB2 cover: <meta name="cover" content="cover-image"/> referencing manifest item id.
			extraMeta += `
    <meta name="cover" content="cover-image"/>`
			manifestItems = append([]string{
				`    <item id="cover-image" href="` + xmlEscape(coverHref) + `" media-type="` + xmlEscape(coverMediaType) + `"/>`,
			}, manifestItems...)
		}
		writeZipFileBytes(t, w, pathpkg.Join("OEBPS", coverHref), opts.CoverImageData)
	}

	writeZipFile(t, w, "OEBPS/content.opf", `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="`+xmlEscape(version)+`" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>`+escapedTitle+`</dc:title>
    <dc:creator>`+escapedCreator+`</dc:creator>
    <dc:identifier id="uid">`+escapedIdentifier+`</dc:identifier>
    <dc:language>`+escapedLang+`</dc:language>`+extraMeta+`
  </metadata>
  <manifest>
`+strings.Join(manifestItems, "\n")+`
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`)

	writeZipFile(t, w, "OEBPS/chapter1.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Chapter 1</title></head><body><p>Hello</p></body></html>`)
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		// xml.EscapeText currently never returns an error, but fall back to the original string if it does.
		return s
	}
	return buf.String()
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

func writeZipFileBytes(t *testing.T, w *zip.Writer, name string, content []byte) {
	t.Helper()
	fw, err := w.Create(name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
