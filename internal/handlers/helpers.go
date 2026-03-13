package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

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

// tmdbLogoURL returns a full TMDB image URL by joining a base URL and path.
// Returns an empty string if path is empty.
func tmdbLogoURL(base, path string) string {
	if path == "" {
		return ""
	}
	return base + path
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
