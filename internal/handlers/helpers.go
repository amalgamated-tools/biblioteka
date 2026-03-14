package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// errorResponse is used by swagger to document error responses.
type errorResponse struct {
	Error string `json:"error" example:"error message"`
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to encode JSON error response", "error", err)
	}
}

const minPasswordLength = 6

// validatePassword checks that a password meets the minimum length requirement.
// Returns an error message if invalid, or an empty string if valid.
func validatePassword(password string) string {
	if len(password) < minPasswordLength {
		return "password must be at least 6 characters"
	}
	return ""
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
