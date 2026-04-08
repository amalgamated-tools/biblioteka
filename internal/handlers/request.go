package handlers

import (
	"net/http"
	"strings"
)

// requestScheme returns the HTTP scheme for the given request. It checks
// r.TLS first, then honors the X-Forwarded-Proto header (normalized to
// lowercase, trimmed). Only "http" and "https" are accepted; any other
// value is ignored and the function falls back to the TLS-based default.
func requestScheme(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		normalized := strings.ToLower(strings.TrimSpace(proto))
		if normalized == "http" || normalized == "https" {
			scheme = normalized
		}
	}
	return scheme
}

// extractPathID extracts a single resource ID from a URL path by stripping the
// given prefix. It trims a trailing slash so that both /api/foo/123 and
// /api/foo/123/ resolve to "123". Returns the ID and true on success, or an
// empty string and false if the result is empty or contains additional segments.
func extractPathID(path, prefix string) (string, bool) {
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractPathSegments extracts a resource ID and optional sub-resource from a URL path.
// For "/api/books/abc123/authors" with prefix "/api/books/", it returns ("abc123", "authors", true).
// For "/api/books/abc123" with prefix "/api/books/", it returns ("abc123", "", true).
func extractPathSegments(path, prefix string) (id, sub string, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	return parts[0], parts[1], true
}
