package exif

import (
	"archive/zip"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testCoverPNG is a 1×1 transparent PNG used as a minimal cover image in tests.
// It is distinct from testutils.TinyPNG to avoid importing that package
// (testutils/pdf.go imports internal/exif, which would create a cycle).
var testCoverPNG = func() []byte {
	b, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+lmRcAAAAASUVORK5CYII=",
	)
	if err != nil {
		panic("cover_test: invalid testCoverPNG base64 constant: " + err.Error())
	}
	return b
}()

// makeEPUBWithCover writes a minimal EPUB file at path.  When epub3Style is
// true the cover manifest item carries properties="cover-image" (EPUB3);
// otherwise a <meta name="cover"> element is added (EPUB2).  The cover image
// is stored at OEBPS/images/cover.png and the OPF root is at
// OEBPS/content.opf, matching the path resolution logic in cover.go.
func makeEPUBWithCover(t *testing.T, path string, coverData []byte, epub3Style bool) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	w := zip.NewWriter(f)
	t.Cleanup(func() { _ = w.Close() })

	// mimetype must be the first entry and stored without compression.
	mh := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	mw, err := w.CreateHeader(mh)
	require.NoError(t, err)
	_, err = mw.Write([]byte("application/epub+zip"))
	require.NoError(t, err)

	addZipEntry(t, w, "META-INF/container.xml", `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	// Cover image placed under the same directory as the OPF.
	fw, err := w.Create("OEBPS/images/cover.png")
	require.NoError(t, err)
	_, err = fw.Write(coverData)
	require.NoError(t, err)

	var coverManifestItem, coverMetaElement string
	if epub3Style {
		// EPUB3: the manifest item itself carries the cover-image property.
		coverManifestItem = `    <item id="cover-image" href="images/cover.png" media-type="image/png" properties="cover-image"/>`
	} else {
		// EPUB2: a <meta name="cover"> element references the manifest item id.
		coverManifestItem = `    <item id="cover-image" href="images/cover.png" media-type="image/png"/>`
		coverMetaElement = "    <meta name=\"cover\" content=\"cover-image\"/>"
	}

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:identifier id="uid">test-id</dc:identifier>
` + coverMetaElement + `
  </metadata>
  <manifest>
` + coverManifestItem + `
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`

	addZipEntry(t, w, "OEBPS/content.opf", opf)
	addZipEntry(t, w, "OEBPS/chapter1.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Test</title></head><body/></html>`)

	// Explicitly close so the ZIP is flushed before tests read it.
	// t.Cleanup handles the case where an earlier require fails.
	require.NoError(t, w.Close(), "close zip writer")
	require.NoError(t, f.Close(), "close epub file")
}

func addZipEntry(t *testing.T, w *zip.Writer, name, content string) {
	t.Helper()
	fw, err := w.Create(name)
	require.NoError(t, err)
	_, err = fw.Write([]byte(content))
	require.NoError(t, err)
}

// TestParseTSV_EPUB3CoverExtractionE2E is the primary regression test for the
// silent cover-extraction failure described in issue #1094 (EPUB3 books
// imported without covers).
//
// The cover discovery strategies in finishEPUB are exercised by the unit tests
// in tsv_test.go, but those tests use a non-existent file path ("test.epub").
// When ParseTSV calls finishEPUB and then extractEPUBCoverDataURL, the ZIP
// read silently fails and CoverImageURL is left empty.  The existing tests
// only assert CoverImage != nil (the manifest item was found) — they never
// check CoverImageURL.  This test closes that gap: it creates a real EPUB3
// file on disk and asserts that the full extraction pipeline produces a
// non-empty data URL.
func TestParseTSV_EPUB3CoverExtractionE2E(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3.epub")
	makeEPUBWithCover(t, epubPath, testCoverPNG, true /* epub3Style */)

	input := "File Type\tEPUB\n" +
		"Directory\t" + dir + "\n" +
		"File Name\tepub3.epub\n" +
		"Manifest Item Href\timages/cover.png\n" +
		"Manifest Item Id\tcover-image\n" +
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Properties\tcover-image\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage,
		"CoverImage manifest item must be found via properties=\"cover-image\"")
	require.NotEmpty(t, out.CoverImageURL,
		"CoverImageURL must be populated — cover should be extracted from the ZIP, not silently dropped")
	require.Truef(t, strings.HasPrefix(out.CoverImageURL, "data:image/png;base64,"),
		"expected PNG data URL, got %q", out.CoverImageURL)

	b64 := strings.TrimPrefix(out.CoverImageURL, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	require.Equal(t, testCoverPNG, imgBytes, "extracted cover bytes should match the original image")
}

// TestParseTSV_EPUB2CoverExtractionE2E mirrors TestParseTSV_EPUB3CoverExtractionE2E
// for the EPUB2 <meta name="cover"> discovery path, providing a baseline that
// both cover-detection strategies reach the ZIP extraction step.
func TestParseTSV_EPUB2CoverExtractionE2E(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub2.epub")
	makeEPUBWithCover(t, epubPath, testCoverPNG, false /* epub3Style */)

	input := "File Type\tEPUB\n" +
		"Directory\t" + dir + "\n" +
		"File Name\tepub2.epub\n" +
		"Meta Content\tcover-image\nMeta Name\tcover\n" +
		"Manifest Item Href\timages/cover.png\n" +
		"Manifest Item Id\tcover-image\n" +
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage,
		"CoverImage manifest item must be found via <meta name=\"cover\">")
	require.NotEmpty(t, out.CoverImageURL,
		"CoverImageURL must be populated for EPUB2 cover via meta tag")
	require.Truef(t, strings.HasPrefix(out.CoverImageURL, "data:image/png;base64,"),
		"expected PNG data URL, got %q", out.CoverImageURL)
}

// TestExtractEPUBCoverDataURL_EPUB3Cover tests extractEPUBCoverDataURL directly
// with a real EPUB3 ZIP file, verifying that path resolution (cover href
// relative to OPF directory) works correctly.
func TestExtractEPUBCoverDataURL_EPUB3Cover(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3.epub")
	makeEPUBWithCover(t, epubPath, testCoverPNG, true /* epub3Style */)

	item := &ManifestItem{
		Href:       "images/cover.png",
		ID:         "cover-image",
		MediaType:  "image/png",
		Properties: "cover-image",
	}

	dataURL, err := extractEPUBCoverDataURL(t.Context(), item, epubPath)
	require.NoError(t, err)
	require.Truef(t, strings.HasPrefix(dataURL, "data:image/png;base64,"),
		"expected PNG data URL, got %q", dataURL)

	b64 := strings.TrimPrefix(dataURL, "data:image/png;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err)
	require.Equal(t, testCoverPNG, imgBytes, "extracted cover bytes should match the original image")
}

// TestExtractEPUBCoverDataURL_MissingArchiveFile verifies that when the cover
// href cannot be located in the ZIP archive, an error is returned (not a
// silent empty string).
func TestExtractEPUBCoverDataURL_MissingArchiveFile(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "epub3.epub")
	makeEPUBWithCover(t, epubPath, testCoverPNG, true /* epub3Style */)

	// Reference an href that does not exist in the archive.
	item := &ManifestItem{
		Href:      "images/nonexistent.png",
		ID:        "cover-image",
		MediaType: "image/png",
	}

	_, err := extractEPUBCoverDataURL(t.Context(), item, epubPath)
	require.Error(t, err, "a missing cover file in the archive should return an error")
}

// TestExtractEPUBCoverDataURL_NilManifestItem verifies that a nil ManifestItem
// returns an empty string without error — the no-cover case is not an error.
func TestExtractEPUBCoverDataURL_NilManifestItem(t *testing.T) {
	dataURL, err := extractEPUBCoverDataURL(t.Context(), nil, "/any/path.epub")
	require.NoError(t, err)
	require.Empty(t, dataURL)
}
