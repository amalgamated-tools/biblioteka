package coverutil

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// ErrNotDataURL is returned when the input is not a valid data: URL.
var ErrNotDataURL = fmt.Errorf("not a data URL")

// maxDecodedBytes is the maximum decoded size for a data URL payload (20 MB).
const maxDecodedBytes = 20 << 20

// DecodeDataURL decodes a base64-encoded data: URL into its MIME type and raw
// bytes. Returns ErrNotDataURL if the input is not a data URL.
func DecodeDataURL(raw string) (string, []byte, error) {
	meta, payload, ok := strings.Cut(raw, ",")
	if !ok || !strings.HasPrefix(meta, "data:") {
		return "", nil, ErrNotDataURL
	}

	header := strings.TrimPrefix(meta, "data:")
	if !strings.HasSuffix(header, ";base64") {
		return "", nil, fmt.Errorf("unsupported data URL encoding")
	}

	mimeType := strings.TrimSuffix(header, ";base64")
	if mimeType == "" {
		mimeType = "text/plain;charset=US-ASCII"
	}

	if base64.StdEncoding.DecodedLen(len(payload)) > maxDecodedBytes+2 {
		return "", nil, fmt.Errorf("data URL payload exceeds %d-byte limit", maxDecodedBytes)
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, fmt.Errorf("decode data URL payload: %w", err)
	}
	if len(data) > maxDecodedBytes {
		return "", nil, fmt.Errorf("data URL payload exceeds %d-byte limit", maxDecodedBytes)
	}

	return mimeType, data, nil
}
