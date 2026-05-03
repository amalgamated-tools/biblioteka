package handlers

import (
	"net/http"
	"strconv"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

// parseLimit extracts only the limit pagination parameter from the request query
// string. Use this for endpoints that do not support offset-based pagination.
// Invalid or out-of-range values silently fall back to safe defaults.
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 1 {
			if n > maxLimit {
				n = maxLimit
			}
			limit = n
		}
	}
	return limit
}

// parseLimitOffset extracts pagination parameters from the request query string.
// Invalid or out-of-range values silently fall back to safe defaults.
func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (int, int) {
	limit := parseLimit(r, defaultLimit, maxLimit)
	offset := 0

	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
