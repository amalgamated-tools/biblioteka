package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// errorResponse represents a JSON error returned by the API.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON response", slog.Any(otelkeys.Error, err))
	}
}

// writeError sends a JSON error response.
func writeError(ctx context.Context, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		slog.ErrorContext(ctx, "failed to encode JSON error response", slog.Any(otelkeys.Error, err))
	}
}

// decodeJSON reads and decodes the JSON request body into v.
// Returns true on success; on failure writes a 400 error response and
// returns false, so callers can simply return.
func decodeJSON(r *http.Request, w http.ResponseWriter, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB cap
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		slog.DebugContext(r.Context(), "failed to decode request body", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// writeSecretTokenResponse sets cache-prevention headers and writes a JSON
// response. It should be used whenever the response body contains a plaintext
// secret token or key that should not be stored in HTTP caches. Note that
// these headers cannot fully prevent storage in browser history or other
// user-controlled storage.
func writeSecretTokenResponse(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(ctx, w, status, data)
}
