package handlers

import (
	"net/http"
	"strconv"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

// parseLimitOffset extracts pagination parameters from the request query string.
// Invalid or out-of-range values silently fall back to safe defaults.
func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 1 {
			if n > maxLimit {
				n = maxLimit
			}
			limit = n
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
