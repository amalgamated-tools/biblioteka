package exif

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasProperty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		properties string
		token      string
		want       bool
	}{
		{name: "exact match", properties: "cover-image", token: "cover-image", want: true},
		{name: "multiple tokens, first matches", properties: "cover-image scripted", token: "cover-image", want: true},
		{name: "multiple tokens, last matches", properties: "scripted cover-image", token: "cover-image", want: true},
		{name: "case insensitive", properties: "Cover-Image", token: "cover-image", want: true},
		{name: "no match", properties: "scripted nav", token: "cover-image", want: false},
		{name: "empty properties", properties: "", token: "cover-image", want: false},
		{name: "empty token", properties: "cover-image", token: "", want: false},
		{name: "partial match not counted", properties: "my-cover-image", token: "cover-image", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasProperty(tt.properties, tt.token)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestFinishEPUB_Strategy2_MetaTagCover verifies that a <meta name="cover">
// pointing to a manifest item sets CoverImage via strategy 2.
func TestFinishEPUB_Strategy2_MetaTagCover(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		MetaTags: []MetaTag{
			{Name: "cover", Content: "img-cover"},
		},
		ManifestItems: []ManifestItem{
			{ID: "img-cover", Href: "images/cover.jpg", MediaType: "image/jpeg"},
			{ID: "chapter1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 2")
	require.Equal(t, "img-cover", out.CoverImage.ID)
	require.Equal(t, "images/cover.jpg", out.CoverImage.Href)
}

// TestFinishEPUB_Strategy3_PropertiesCoverImage verifies that a manifest item
// with properties="cover-image" sets CoverImage via strategy 3.
func TestFinishEPUB_Strategy3_PropertiesCoverImage(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "ch1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "cover-img", Href: "images/front.jpg", MediaType: "image/jpeg", Properties: "cover-image"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 3")
	require.Equal(t, "cover-img", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy3_PropertiesWithMultipleTokens verifies that a
// manifest item with multiple space-separated properties is recognized.
func TestFinishEPUB_Strategy3_PropertiesWithMultipleTokens(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "front", Href: "images/front.png", MediaType: "image/png", Properties: "cover-image scripted"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 3 (multi-token properties)")
	require.Equal(t, "front", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy4_IDContainsCover verifies that a manifest item whose
// ID contains "cover" is selected via strategy 4.
func TestFinishEPUB_Strategy4_IDContainsCover(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "chapter1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "cover-image-png", Href: "images/img.png", MediaType: "image/png"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 4 (ID contains cover)")
	require.Equal(t, "cover-image-png", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy4_HrefContainsCover verifies that a manifest item
// whose href contains "cover" is selected via strategy 4.
func TestFinishEPUB_Strategy4_HrefContainsCover(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "chapter1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "img001", Href: "images/cover.png", MediaType: "image/png"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 4 (href contains cover)")
	require.Equal(t, "img001", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy5_SingleImageFallback verifies that a single image in
// the manifest is selected via strategy 5 when no other strategy applies.
func TestFinishEPUB_Strategy5_SingleImageFallback(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "ch1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "img001", Href: "images/illustration.png", MediaType: "image/png"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected via strategy 5 (single image fallback)")
	require.Equal(t, "img001", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy5_MultipleImagesNoFallback verifies that the
// single-image fallback is NOT applied when there are multiple images.
func TestFinishEPUB_Strategy5_MultipleImagesNoFallback(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "img001", Href: "images/illustration1.png", MediaType: "image/png"},
			{ID: "img002", Href: "images/illustration2.png", MediaType: "image/png"},
		},
	}

	finishEPUB(t.Context(), out)

	// Two images — no clear cover — CoverImage should not be set.
	require.Nil(t, out.CoverImage, "expected CoverImage to remain nil with multiple images and no strategy match")
}

// TestFinishEPUB_NoCoverFound verifies that CoverImage stays nil when none of
// the strategies can identify a cover in the manifest.
func TestFinishEPUB_NoCoverFound(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "chapter1", Href: "text/ch1.xhtml", MediaType: "application/xhtml+xml"},
			{ID: "toc", Href: "toc.ncx", MediaType: "application/x-dtbncx+xml"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Nil(t, out.CoverImage, "expected CoverImage to remain nil when no images are in the manifest")
}

// TestFinishEPUB_Strategy2IgnoresNonImageItems verifies that strategy 2 ignores
// manifest items that are not images, even if their ID matches the meta tag.
func TestFinishEPUB_Strategy2IgnoresNonImageItems(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		MetaTags: []MetaTag{
			{Name: "cover", Content: "cover-item"},
		},
		ManifestItems: []ManifestItem{
			// This item matches the meta tag ID but is not an image.
			{ID: "cover-item", Href: "text/cover.xhtml", MediaType: "application/xhtml+xml"},
		},
	}

	finishEPUB(t.Context(), out)

	// Strategy 2 should not pick an xhtml item; no other strategy applies.
	require.Nil(t, out.CoverImage, "expected CoverImage to remain nil when meta cover points to a non-image item")
}

// TestFinishEPUB_Strategy1PreservesCoverSetByTSVParser verifies that a
// CoverImage pre-populated by the TSV parser (strategy 1) is not overwritten.
func TestFinishEPUB_Strategy1PreservesCoverSetByTSVParser(t *testing.T) {
	t.Parallel()

	existing := &ManifestItem{ID: "preloaded", Href: "images/preloaded.jpg", MediaType: "image/jpeg"}
	out := &ExifToolOutput{
		FileType:   "EPUB",
		CoverImage: existing,
		MetaTags: []MetaTag{
			{Name: "cover", Content: "other-img"},
		},
		ManifestItems: []ManifestItem{
			{ID: "other-img", Href: "images/other.jpg", MediaType: "image/jpeg"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Same(t, existing, out.CoverImage, "expected pre-populated CoverImage to be preserved by finishEPUB")
}

// TestFinishEPUB_ISBNFromIdentifiers verifies that ISBN extraction from
// <identifier> elements works when explicit isbn10/isbn13 fields are absent.
func TestFinishEPUB_ISBNFromIdentifiers(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		Identifiers: []Identifier{
			{Value: "urn:isbn:9781234567890"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Equal(t, "9781234567890", out.ISBN13, "expected ISBN13 to be extracted from identifier")
}

// TestFinishEPUB_ISBNSkippedWhenAlreadyPresent verifies that identifier-based
// ISBN extraction is skipped when ISBN13/ISBN10 are already set.
func TestFinishEPUB_ISBNSkippedWhenAlreadyPresent(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ISBN13:   "9780000000001",
		Identifiers: []Identifier{
			{Value: "urn:isbn:9781234567890"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Equal(t, "9780000000001", out.ISBN13, "pre-existing ISBN13 should not be overwritten")
}

// TestFinishBook_DispatchesEPUB verifies that finishBook dispatches to
// finishEPUB for EPUB file types.
func TestFinishBook_DispatchesEPUB(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ManifestItems: []ManifestItem{
			{ID: "img001", Href: "images/illustration.png", MediaType: "image/png"},
		},
	}

	finishBook(t.Context(), out)

	// Single image → strategy 5 should fire.
	require.NotNil(t, out.CoverImage, "expected finishBook to dispatch to finishEPUB for EPUB files")
}

// TestFinishEPUB_CoverImageURLExtracted verifies that CoverImageURL is populated
// when a real EPUB file exists at the path with an actual cover image.
func TestFinishEPUB_CoverImageURLExtracted(t *testing.T) {
	dir := t.TempDir()
	epubPath := filepath.Join(dir, "test.epub")
	makeEPUBWithCover(t, epubPath, testCoverPNG, true /* epub3Style */)

	out := &ExifToolOutput{
		FileType:  "EPUB",
		Directory: dir,
		FileName:  "test.epub",
		ManifestItems: []ManifestItem{
			{ID: "cover-img", Href: "OEBPS/images/cover.png", MediaType: "image/png", Properties: "cover-image"},
		},
	}

	finishEPUB(t.Context(), out)

	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected")
	require.NotEmpty(t, out.CoverImageURL, "expected CoverImageURL to be populated from real EPUB file")
	require.Contains(t, out.CoverImageURL, "data:image/", "expected CoverImageURL to be a data URL")
}

// TestFinishEPUB_CoverImageURLMissingFile verifies that CoverImage is selected
// but CoverImageURL is left empty when the EPUB file does not exist.
func TestFinishEPUB_CoverImageURLMissingFile(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType:  "EPUB",
		Directory: "/nonexistent",
		FileName:  "missing.epub",
		ManifestItems: []ManifestItem{
			{ID: "img001", Href: "images/cover.png", MediaType: "image/png"},
		},
	}

	finishEPUB(t.Context(), out)

	// Strategy 5 should select the single image, but extraction must fail
	// because the file does not exist.
	require.NotNil(t, out.CoverImage, "expected CoverImage to be selected")
	require.Empty(t, out.CoverImageURL, "expected CoverImageURL to be empty when file is missing")
}

// TestFinishEPUB_ASINFromIdentifiers verifies ASIN extraction from
// <identifier> elements.
func TestFinishEPUB_ASINFromIdentifiers(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		Identifiers: []Identifier{
			{Value: "B08FHBV4ZX"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Equal(t, "B08FHBV4ZX", out.ASIN, "expected ASIN to be extracted from identifier")
}

// TestFinishEPUB_ASINSkippedForKnownSchemes verifies that identifiers with
// known schemes (ISBN, calibre, etc.) are not treated as ASINs.
func TestFinishEPUB_ASINSkippedForKnownSchemes(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		Identifiers: []Identifier{
			{Value: "B08FHBV4ZX", Scheme: "calibre"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Empty(t, out.ASIN, "expected ASIN to remain empty for identifier with known scheme")
}

// TestFinishEPUB_ASINSkippedWhenAlreadyPresent verifies that identifier-based
// ASIN extraction is skipped when ASIN is already set.
func TestFinishEPUB_ASINSkippedWhenAlreadyPresent(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		ASIN:     "B000000001",
		Identifiers: []Identifier{
			{Value: "B08FHBV4ZX"},
		},
	}

	finishEPUB(t.Context(), out)

	require.Equal(t, "B000000001", out.ASIN, "pre-existing ASIN should not be overwritten")
}

// TestFinishBook_UnknownTypeIsNoop verifies that finishBook is a no-op for
// unrecognized file types.
func TestFinishBook_UnknownTypeIsNoop(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "PDF",
		ManifestItems: []ManifestItem{
			{ID: "img001", Href: "cover.png", MediaType: "image/png"},
		},
	}

	finishBook(t.Context(), out)

	require.Nil(t, out.CoverImage, "expected finishBook to be a no-op for PDF file type")
	require.Empty(t, out.CoverImageURL)
}

// TestFinishEPUB_Strategy2CaseInsensitiveMetaName verifies strategy 2 works
// when the <meta> name attribute has non-lowercase casing.
func TestFinishEPUB_Strategy2CaseInsensitiveMetaName(t *testing.T) {
	t.Parallel()

	out := &ExifToolOutput{
		FileType: "EPUB",
		MetaTags: []MetaTag{
			{Name: "Cover", Content: "cover-img"},
		},
		ManifestItems: []ManifestItem{
			{ID: "cover-img", Href: "images/cover.jpg", MediaType: "image/jpeg"},
		},
	}

	finishEPUB(t.Context(), out)

	// The implementation uses strings.EqualFold for the meta name check.
	require.NotNil(t, out.CoverImage)
	require.Equal(t, "cover-img", out.CoverImage.ID)
}

// TestFinishEPUB_Strategy4CaseInsensitiveIDAndHref verifies strategy 4 treats
// ID and href comparison case-insensitively.
func TestFinishEPUB_Strategy4CaseInsensitiveIDAndHref(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item ManifestItem
	}{
		{
			name: "uppercase ID",
			item: ManifestItem{ID: "COVER", Href: "images/img.png", MediaType: "image/png"},
		},
		{
			name: "uppercase href",
			item: ManifestItem{ID: "img001", Href: "images/COVER.png", MediaType: "image/png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &ExifToolOutput{
				FileType:      "EPUB",
				ManifestItems: []ManifestItem{tt.item},
			}
			finishEPUB(t.Context(), out)
			require.NotNil(t, out.CoverImage, "expected cover image to be found")
		})
	}
}

// TestFinishBook_DispatchesMOBI verifies that finishBook does not panic for
// MOBI/AZW3 file types (finishMOBI is called and handles a missing file path
// gracefully).
func TestFinishBook_DispatchesMOBI(t *testing.T) {
	t.Parallel()

	for _, ft := range []string{"MOBI", "AZW3"} {
		t.Run(ft, func(t *testing.T) {
			t.Parallel()
			out := &ExifToolOutput{
				FileType:  ft,
				Directory: os.TempDir(),
				FileName:  "nonexistent.mobi",
			}
			// finishMOBI calls GetMobiCover, which should return ErrNoCover or
			// an error for a missing file. Either way it must not panic.
			require.NotPanics(t, func() { finishBook(t.Context(), out) })
		})
	}
}
