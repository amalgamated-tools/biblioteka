package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
)

var errNotDataURL = errors.New("not a data URL")

func coverMIMEType(imageURL string) string {
	if mimeType, ok := dataURLMIMEType(imageURL); ok {
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
	meta, payload, ok := strings.Cut(raw, ",")
	if !ok || !strings.HasPrefix(meta, "data:") {
		return "", nil, errNotDataURL
	}

	header := strings.TrimPrefix(meta, "data:")
	if !strings.HasSuffix(header, ";base64") {
		return "", nil, fmt.Errorf("unsupported data URL encoding")
	}

	mimeType := strings.TrimSuffix(header, ";base64")
	if mimeType == "" {
		mimeType = "text/plain;charset=US-ASCII"
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, fmt.Errorf("decode data URL payload: %w", err)
	}

	return mimeType, data, nil
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
