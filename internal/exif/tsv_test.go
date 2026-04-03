package exif

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTSV_EmptyInput(t *testing.T) {
	out, err := ParseTSV(t.Context(), "", "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if len(out.Identifiers) != 0 {
		t.Errorf("expected 0 identifiers, got %d", len(out.Identifiers))
	}
}

func TestParseTSV_IdentifierWithoutScheme(t *testing.T) {
	input := "Identifier\turn:isbn:1234567890\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 1 {
		t.Fatalf("expected 1 identifier, got %d", got)
	}
	require.Equal(t, "urn:isbn:1234567890", out.Identifiers[0].Value, "Identifiers[0].Value")
	require.Equal(t, "", out.Identifiers[0].Scheme, "Identifiers[0].Scheme")
}

func TestParseTSV_IdentifierWithScheme(t *testing.T) {
	input := "Identifier Scheme\tAMAZON\nIdentifier\tB08FHBV4ZX\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 1 {
		t.Fatalf("expected 1 identifier, got %d", got)
	}
	require.Equal(t, "B08FHBV4ZX", out.Identifiers[0].Value, "Identifiers[0].Value")
	require.Equal(t, "AMAZON", out.Identifiers[0].Scheme, "Identifiers[0].Scheme")
}

func TestParseTSV_IdentifierIdPrecedesValue(t *testing.T) {
	input := "Identifier Id\tuid\nIdentifier\t12345\nIdentifier Scheme\tcalibre\nIdentifier\tabcdef\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 2 {
		t.Fatalf("expected 2 identifiers, got %d", got)
	}
	require.Equal(t, "12345", out.Identifiers[0].Value, "Identifiers[0].Value")
	require.Equal(t, "uid", out.Identifiers[0].ID, "Identifiers[0].ID")
	require.Equal(t, "", out.Identifiers[0].Scheme, "Identifiers[0].Scheme")
	require.Equal(t, "abcdef", out.Identifiers[1].Value, "Identifiers[1].Value")
	require.Equal(t, "calibre", out.Identifiers[1].Scheme, "Identifiers[1].Scheme")
	require.Equal(t, "", out.Identifiers[1].ID, "Identifiers[1].ID")
}

func TestParseTSV_MultipleIdentifiers(t *testing.T) {
	input := "Identifier Scheme\tISBN\nIdentifier\tAAA\nIdentifier Scheme\tAMAZON\nIdentifier\tBBB\nIdentifier\tCCC\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.Identifiers); got != 3 {
		t.Fatalf("expected 3 identifiers, got %d", got)
	}
	require.Equal(t, "AAA", out.Identifiers[0].Value, "Identifiers[0].Value")
	require.Equal(t, "ISBN", out.Identifiers[0].Scheme, "Identifiers[0].Scheme")
	require.Equal(t, "BBB", out.Identifiers[1].Value, "Identifiers[1].Value")
	require.Equal(t, "AMAZON", out.Identifiers[1].Scheme, "Identifiers[1].Scheme")
	require.Equal(t, "CCC", out.Identifiers[2].Value, "Identifiers[2].Value")
	require.Equal(t, "", out.Identifiers[2].Scheme, "Identifiers[2].Scheme")
}

func TestParseTSV_MetaPairs(t *testing.T) {
	input := "Meta Content\tcover\nMeta Name\tcover\nMeta Content\tA Novel\nMeta Name\tbooklore:subtitle\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	if got := len(out.MetaTags); got != 2 {
		t.Fatalf("expected 2 meta tags, got %d", got)
	}
	require.Equal(t, "cover", out.MetaTags[0].Content, "MetaTags[0].Content")
	require.Equal(t, "cover", out.MetaTags[0].Name, "MetaTags[0].Name")
	require.Equal(t, "A Novel", out.MetaTags[1].Content, "MetaTags[1].Content")
	require.Equal(t, "booklore:subtitle", out.MetaTags[1].Name, "MetaTags[1].Name")
}

func TestParseTSV_UnknownFieldsGoToExtra(t *testing.T) {
	input := "ExifTool Version Number\t13.50\nFile Permissions\t-rw-r--r--\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err, "ParseTSV should not return an error")
	require.Equal(t, "-rw-r--r--", out.Extras["File Permissions"], "Extra[File Permissions]")
}

func TestFinishEPUB_ISBN10NotMistakenForASIN(t *testing.T) {
	// An ISBN-10 identifier should not be treated as an ASIN.
	input := "Identifier Scheme\tISBN\nIdentifier\t0743273567\nFile Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "0743273567", out.ISBN10, "ISBN10 should be set")
	require.Equal(t, "", out.ASIN, "ASIN should not be set from an ISBN-10 value")
}

func TestFinishEPUB_ISBN10WithoutSchemeNotMistakenForASIN(t *testing.T) {
	// Even without a scheme, a bare ISBN-10 value should not become an ASIN.
	input := "Identifier\t0743273567\nFile Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "0743273567", out.ISBN10, "ISBN10 should be set from bare identifier")
	require.Equal(t, "", out.ASIN, "ASIN should not be set from an ISBN-10 value")
}

func TestFinishEPUB_RealASINIsPreserved(t *testing.T) {
	// A real ASIN (e.g. B08FHBV4ZX) should still be detected by the heuristic.
	input := "Identifier\tB08FHBV4ZX\nFile Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "B08FHBV4ZX", out.ASIN, "ASIN should be set from unschemed ASIN-like identifier")
}

func TestParseTSV_SubjectsDeduplication(t *testing.T) {
	// Multiple Subject lines with overlapping comma-separated values.
	input := "Subject\tFiction, Science Fiction\nSubject\tScience Fiction, Fantasy, Horror\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, []string{"Fiction", "Science Fiction", "Fantasy", "Horror"}, out.Subjects)
}

func TestParseTSV_SubjectsSingleLine(t *testing.T) {
	input := "Subject\tHistory, Biography\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, []string{"History", "Biography"}, out.Subjects)
}

func TestParseTSV_SubjectsWhitespaceHandling(t *testing.T) {
	// Extra whitespace around values should be trimmed, empty segments skipped.
	input := "Subject\t  Romance , , Thriller \n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, []string{"Romance", "Thriller"}, out.Subjects)
}

func TestParseTSV_DuplicateCalibreID(t *testing.T) {
	// Second Calibre ID should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tCALIBRE\nIdentifier\tabc123\nIdentifier Scheme\tCALIBRE\nIdentifier\tdef456\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "abc123", out.CalibreID, "first Calibre ID should be kept")
	require.Equal(t, "def456", out.Extras["Duplicate Calibre ID"], "second Calibre ID should be in Extras")
}

func TestParseTSV_DuplicateGoodreadsID(t *testing.T) {
	// Second Goodreads ID should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tGOODREADS\nIdentifier\t12345\nIdentifier Scheme\tGOODREADS\nIdentifier\t67890\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "12345", out.GoodreadsID, "first Goodreads ID should be kept")
	require.Equal(t, "67890", out.Extras["Duplicate Goodreads ID"], "second Goodreads ID should be in Extras")
}

func TestParseTSV_DuplicateASIN(t *testing.T) {
	// Second ASIN should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tAMAZON\nIdentifier\tB001\nIdentifier Scheme\tAMAZON\nIdentifier\tB002\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "B001", out.ASIN, "first ASIN should be kept")
	require.Equal(t, "B002", out.Extras["Duplicate ASIN"], "second ASIN should be in Extras")
}

func TestParseTSV_DuplicateISBN10(t *testing.T) {
	// Second ISBN-10 should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tISBN\nIdentifier\turn:isbn:0743273567\nIdentifier Scheme\tISBN\nIdentifier\turn:isbn:0306406152\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "0743273567", out.ISBN10, "first ISBN-10 should be kept")
	require.Equal(t, "0306406152", out.Extras["Duplicate ISBN-10"], "second ISBN-10 should be in Extras")
}

func TestParseTSV_DuplicateISBN13(t *testing.T) {
	// Second ISBN-13 should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tISBN\nIdentifier\turn:isbn:9780743273565\nIdentifier Scheme\tISBN\nIdentifier\turn:isbn:9780306406157\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "9780743273565", out.ISBN13, "first ISBN-13 should be kept")
	require.Equal(t, "9780306406157", out.Extras["Duplicate ISBN-13"], "second ISBN-13 should be in Extras")
}

func TestParseTSV_DuplicateGoogleID(t *testing.T) {
	// Second Google ID should be kept in Extras, first is preserved.
	input := "Identifier Scheme\tGOOGLE\nIdentifier\tXYZ1\nIdentifier Scheme\tGOOGLE\nIdentifier\tXYZ2\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "XYZ1", out.GoogleID, "first Google ID should be kept")
	require.Equal(t, "XYZ2", out.Extras["Duplicate Google ID"], "second Google ID should be in Extras")
}

func TestParseTSV_DuplicateHardcoverID(t *testing.T) {
	// Second Hardcover ID should be kept in Extras, first is preserved.
	input := "Identifier\turn:hardcoverbook:111\nIdentifier\turn:hardcoverbook:222\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "111", out.HardcoverID, "first Hardcover ID should be kept")
	require.Equal(t, "222", out.Extras["Duplicate Hardcover ID"], "second Hardcover ID should be in Extras")
}

func TestParseTSV_SameIDTwiceIsIdempotent(t *testing.T) {
	// Seeing the same Calibre ID twice should not put it in Extras.
	input := "Identifier Scheme\tCALIBRE\nIdentifier\tabc123\nIdentifier Scheme\tCALIBRE\nIdentifier\tabc123\n"
	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Equal(t, "abc123", out.CalibreID, "Calibre ID should be set")
	require.NotContains(t, out.Extras, "Duplicate Calibre ID", "identical repeat should not go to Extras")
}

// --- EPUB3-specific tests ---

func TestParseTSV_EPUB3CoverViaProperties(t *testing.T) {
	// EPUB3 identifies cover images via properties="cover-image" on the manifest item,
	// without a <meta name="cover"> tag. This tests Strategy 3 in finishEPUB.
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Manifest Item Href\timages/cover.jpg\n" +
		"Manifest Item Id\tcover-img\n" +
		"Manifest Item Media-type\timage/jpeg\n" +
		"Manifest Item Properties\tcover-image\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage, "cover image should be discovered via properties")
	require.Equal(t, "images/cover.jpg", out.CoverImage.Href, "cover href")
	require.Equal(t, "cover-img", out.CoverImage.ID, "cover ID")
	require.Equal(t, "image/jpeg", out.CoverImage.MediaType, "cover media type")
	require.Equal(t, "cover-image", out.CoverImage.Properties, "cover properties")
}

func TestParseTSV_EPUB3CoverPropertiesOverrideFallback(t *testing.T) {
	// Strategy 3 (properties="cover-image") should beat Strategy 4
	// (ID containing "cover") even when the Strategy 4 candidate appears
	// earlier in the manifest.
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Manifest Item Href\timages/cover-fallback.jpg\n" +
		"Manifest Item Id\tcover-fallback\n" + // Strategy 4: ID contains "cover"
		"Manifest Item Media-type\timage/jpeg\n" +
		"Manifest Item Href\timages/real-cover.png\n" +
		"Manifest Item Id\tfront\n" + // no "cover" in ID or href
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Properties\tcover-image\n" + // Strategy 3
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage)
	require.Equal(t, "images/real-cover.png", out.CoverImage.Href,
		"properties='cover-image' (Strategy 3) should beat ID-heuristic (Strategy 4) regardless of manifest order")
}

func TestParseTSV_EPUB2MetaCoverBeatsEPUB3Properties(t *testing.T) {
	// Strategy 2 (<meta name="cover">) has higher priority than Strategy 3
	// (properties="cover-image"). Verify the EPUB2-style meta tag wins.
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Meta Content\tlegacy-cover\nMeta Name\tcover\n" +
		"Manifest Item Href\timages/old-cover.jpg\n" +
		"Manifest Item Id\tlegacy-cover\n" +
		"Manifest Item Media-type\timage/jpeg\n" +
		"Manifest Item Href\timages/new-cover.png\n" +
		"Manifest Item Id\tnew-cover\n" +
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Properties\tcover-image\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage)
	require.Equal(t, "images/old-cover.jpg", out.CoverImage.Href,
		"EPUB2 meta cover (Strategy 2) should take priority over EPUB3 properties (Strategy 3)")
}

func TestParseTSV_EPUB3NoCoverFallsBackToSingleImage(t *testing.T) {
	// EPUB3 with no cover metadata at all — single image fallback (Strategy 5).
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Manifest Item Href\timages/illustration.png\n" +
		"Manifest Item Id\tillustration\n" +
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.NotNil(t, out.CoverImage, "single image should be used as fallback cover")
	require.Equal(t, "images/illustration.png", out.CoverImage.Href)
}

func TestParseTSV_EPUB3NoCoverMultipleImagesNilCover(t *testing.T) {
	// EPUB3 with no cover metadata and multiple images — no cover should be selected.
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Manifest Item Href\timages/fig1.png\n" +
		"Manifest Item Id\tfig1\n" +
		"Manifest Item Media-type\timage/png\n" +
		"Manifest Item Href\timages/fig2.jpg\n" +
		"Manifest Item Id\tfig2\n" +
		"Manifest Item Media-type\timage/jpeg\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Nil(t, out.CoverImage, "multiple images without cover metadata should not select a cover")
}

func TestParseTSV_EPUB3ManifestItemProperties(t *testing.T) {
	// Verify that the Properties field is correctly parsed from manifest items.
	input := "File Type\tEPUB\nDirectory\t.\nFile Name\ttest.epub\n" +
		"Manifest Item Href\tnav.xhtml\n" +
		"Manifest Item Id\tnav\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n" +
		"Manifest Item Properties\tnav\n" +
		"Manifest Item Href\tchapter1.xhtml\n" +
		"Manifest Item Id\tchapter1\n" +
		"Manifest Item Media-type\tapplication/xhtml+xml\n" +
		"Manifest Item Properties\tscripted\n"

	out, err := ParseTSV(t.Context(), input, "epub")
	require.NoError(t, err)
	require.Len(t, out.ManifestItems, 2)
	require.Equal(t, "nav", out.ManifestItems[0].Properties, "first item properties")
	require.Equal(t, "scripted", out.ManifestItems[1].Properties, "second item properties")
}
