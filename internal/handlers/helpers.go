package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// writeJSON sends a JSON response with the given status code.
func writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON response", slog.Any(otelkeys.Error, err))
	}
}

// errorResponse represents a JSON error returned by the API.
type errorResponse struct {
	Error string `json:"error"`
}

// writeError sends a JSON error response.
func writeError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON error response", slog.Any(otelkeys.Error, err))
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
