package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	// maxTokenNameLength is the shared maximum length for API key and Kobo token names.
	maxTokenNameLength = 100

	minPasswordLength = 8
	// maxPasswordLength caps passwords at 72 bytes — bcrypt's internal truncation
	// limit. Without this cap, two passwords that share the same first 72 bytes
	// would be treated as identical (silent truncation collision). NIST SP 800-63B
	// §5.1.1.2 requires verifiers to support at least 64 characters; 72 matches
	// bcrypt's natural limit and avoids any silent truncation.
	maxPasswordLength = 72
)

// validateName returns true if name is non-blank. On failure it writes a 400
// error response and returns false, so callers can simply return.
func validateName(ctx context.Context, w http.ResponseWriter, name string) bool {
	if strings.TrimSpace(name) == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return false
	}
	return true
}

// validateTokenName trims whitespace from name, then validates that it is
// non-empty and does not exceed maxTokenNameLength. It returns the trimmed name
// and true on success. On failure it writes the appropriate 400 error response
// and returns "", false so callers can simply return.
func validateTokenName(ctx context.Context, w http.ResponseWriter, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		writeError(ctx, w, http.StatusBadRequest, "name is required")
		return "", false
	}
	if len(name) > maxTokenNameLength {
		writeError(ctx, w, http.StatusBadRequest, fmt.Sprintf("name must be at most %d characters", maxTokenNameLength))
		return "", false
	}
	return name, true
}

// validatePassword checks that a password meets the length requirements.
// On failure it writes a 400 error response and returns false, so callers can
// simply return.
func validatePassword(ctx context.Context, w http.ResponseWriter, password string) bool {
	if len(password) < minPasswordLength {
		writeError(ctx, w, http.StatusBadRequest, fmt.Sprintf("password must be at least %d bytes", minPasswordLength))
		return false
	}
	if len(password) > maxPasswordLength {
		writeError(ctx, w, http.StatusBadRequest, fmt.Sprintf("password must be at most %d bytes", maxPasswordLength))
		return false
	}
	return true
}
