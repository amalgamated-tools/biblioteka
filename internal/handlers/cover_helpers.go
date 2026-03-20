package handlers

import (
	"net/url"
	"path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/coverutil"
)

var errNotDataURL = coverutil.ErrNotDataURL

func coverMIMEType(imageURL string) string {
	if mimeType, ok := dataURLMIMEType(imageURL); ok && strings.HasPrefix(mimeType, "image/") {
		return mimeType
	}

	extSource := imageURL
	if u, err := url.Parse(imageURL); err == nil {
		extSource = u.Path
	}

	switch strings.ToLower(path.Ext(extSource)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/jpeg"
	}
}

func decodeDataURL(raw string) (string, []byte, error) {
	return coverutil.DecodeDataURL(raw)
}

func dataURLMIMEType(raw string) (string, bool) {
	meta, _, ok := strings.Cut(raw, ",")
	if !ok || !strings.HasPrefix(meta, "data:") {
		return "", false
	}

	header := strings.TrimPrefix(meta, "data:")
	if idx := strings.Index(header, ";"); idx >= 0 {
		header = header[:idx]
	}
	if header == "" {
		return "text/plain;charset=US-ASCII", true
	}

	return header, true
}
