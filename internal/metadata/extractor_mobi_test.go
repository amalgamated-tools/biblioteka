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

	require.Equal(t, "MOBI Title", meta.Title)
	require.Equal(t, "MOBI Author", meta.Author)
	require.Equal(t, "B08FHBV4ZX", meta.ASIN)
	require.Equal(t, "Test Publisher", meta.Publisher)
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

	require.True(t, strings.HasPrefix(meta.CoverImageURL, "data:image/jpeg;base64,"))

	b64 := strings.TrimPrefix(meta.CoverImageURL, "data:image/jpeg;base64,")
	imgBytes, err := base64.StdEncoding.DecodeString(b64)
	require.NoError(t, err, "invalid base64 in cover data URL")
	_, _, err = image.Decode(bytes.NewReader(imgBytes))
	require.NoError(t, err, "cover base64 does not decode as a valid image")
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

	require.Equal(t, "AZW3 Title", meta.Title)
	require.Equal(t, "AZW3 Author", meta.Author)
}
