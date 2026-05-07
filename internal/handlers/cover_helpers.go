package handlers

import (
	"net"
	"net/url"
	"path"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/coverutil"
	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
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

// isSafeCoverRedirectURL reports whether rawURL is safe to redirect to for a
// cover image. Only absolute HTTPS URLs with a non-empty host are permitted;
// all other schemes (http, javascript, data, protocol-relative, etc.) are
// rejected to prevent open-redirect attacks. IP-literal private, loopback, and
// link-local addresses are also rejected to prevent SSRF via cover metadata.
func isSafeCoverRedirectURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsedURL.Scheme, "https") || parsedURL.Host == "" {
		return false
	}
	host := parsedURL.Hostname()
	// Reject IPv6 zone identifiers (e.g. "fe80::1%lo0"): net.ParseIP returns
	// nil for such strings, letting them slip through to the hostname branch.
	// ssrf.ValidateURL applies the same guard.
	if strings.Contains(host, "%") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ssrf.IsPrivateIP(ip)
	}
	return true
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
