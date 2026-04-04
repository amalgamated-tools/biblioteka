package metadata

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/testutils"

	"github.com/stretchr/testify/require"
)

func TestExtractMetadata_MOBI(t *testing.T) {
	dir := t.TempDir()
	mobiPath := filepath.Join(dir, "test.mobi")
	testutils.MakeTestMOBI(t, mobiPath, "MOBI Title", "MOBI Author", testutils.MOBIOptions{
		ASIN:      "B08FHBV4ZX",
		Publisher: "Test Publisher",
		Language:  "en",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), mobiPath)
	require.NoError(t, err, "extract")

	if meta.Title != "MOBI Title" {
		t.Errorf("expected title %q, got %q", "MOBI Title", meta.Title)
	}
	if meta.Author != "MOBI Author" {
		t.Errorf("expected author %q, got %q", "MOBI Author", meta.Author)
	}
	if meta.ASIN != "B08FHBV4ZX" {
		t.Errorf("expected ASIN %q, got %q", "B08FHBV4ZX", meta.ASIN)
	}
	if meta.Publisher != "Test Publisher" {
		t.Errorf("expected publisher %q, got %q", "Test Publisher", meta.Publisher)
	}
}

func TestExtractMetadata_MOBIWithCover(t *testing.T) {
	dir := t.TempDir()
	mobiPath := filepath.Join(dir, "cover.mobi")
	testutils.MakeTestMOBI(t, mobiPath, "Cover Book", "Cover Author", testutils.MOBIOptions{
		CoverImageData: testutils.TinyJPEG(),
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), mobiPath)
	require.NoError(t, err, "extract")

	if !strings.HasPrefix(meta.CoverImageURL, "data:image/jpeg;base64,") {
		t.Errorf("expected JPEG data URL, got %q", meta.CoverImageURL)
	}

	b64 := strings.TrimPrefix(meta.CoverImageURL, "data:image/jpeg;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err, "invalid base64 in cover data URL")
	if _, _, err := image.Decode(bytes.NewReader(imgBytes)); err != nil {
		require.NoError(t, err, "cover base64 does not decode as a valid image")
	}
}

func TestExtractMetadata_AZW3(t *testing.T) {
	dir := t.TempDir()
	azw3Path := filepath.Join(dir, "test.azw3")
	testutils.MakeTestAZW3(t, azw3Path, "AZW3 Title", "AZW3 Author", testutils.MOBIOptions{
		ISBN: "9780743273565",
	})

	ext := requireExifTool(t)
	defer ext.Close(t.Context())

	meta, err := ext.ExtractMetadata(t.Context(), azw3Path)
	require.NoError(t, err, "extract")

	if meta.Title != "AZW3 Title" {
		t.Errorf("expected title %q, got %q", "AZW3 Title", meta.Title)
	}
	if meta.Author != "AZW3 Author" {
		t.Errorf("expected author %q, got %q", "AZW3 Author", meta.Author)
	}
}
