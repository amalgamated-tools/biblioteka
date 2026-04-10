package coverutil

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeDataURL_ValidPNG(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:image/png;base64," + encoded

	mimeType, decoded, err := DecodeDataURL(raw)
	require.NoError(t, err)
	require.Equal(t, "image/png", mimeType)
	require.Len(t, decoded, len(data))
}

func TestDecodeDataURL_ValidJPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:image/jpeg;base64," + encoded

	mimeType, decoded, err := DecodeDataURL(raw)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mimeType)
	require.Len(t, decoded, len(data))
}

func TestDecodeDataURL_NotDataURL(t *testing.T) {
	_, _, err := DecodeDataURL("https://example.com/image.jpg")
	require.ErrorIs(t, err, ErrNotDataURL)
}

func TestDecodeDataURL_EmptyString(t *testing.T) {
	_, _, err := DecodeDataURL("")
	require.ErrorIs(t, err, ErrNotDataURL)
}

func TestDecodeDataURL_UnsupportedEncoding(t *testing.T) {
	_, _, err := DecodeDataURL("data:image/png;utf8,hello")
	require.Error(t, err, "expected error for unsupported encoding")
}

func TestDecodeDataURL_EmptyMIME(t *testing.T) {
	data := []byte("hello")
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:;base64," + encoded

	mimeType, _, err := DecodeDataURL(raw)
	require.NoError(t, err)
	require.Equal(t, "text/plain;charset=US-ASCII", mimeType)
}

func TestDecodeDataURL_InvalidBase64(t *testing.T) {
	_, _, err := DecodeDataURL("data:image/png;base64,!!!invalid!!!")
	require.Error(t, err, "expected error for invalid base64")
}

func TestDecodeDataURL_ExceedsSizeLimit(t *testing.T) {
	// Create a payload that would exceed 20 MB when decoded.
	bigData := strings.Repeat("A", 30*1024*1024) // 30 MB of base64
	raw := "data:image/png;base64," + bigData

	_, _, err := DecodeDataURL(raw)
	require.Error(t, err, "expected error for oversized payload")
}
