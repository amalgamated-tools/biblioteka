package handlers

import (
	"net/http"
	"strconv"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

// parseBoundedQueryInt extracts a single integer query parameter by key,
// clamps it to [minVal, maxVal], and falls back to defaultVal on any error.
func parseBoundedQueryInt(r *http.Request, key string, defaultVal, minVal, maxVal int) int {
	v := defaultVal
	if s := r.URL.Query().Get(key); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil && n >= minVal {
			if n > maxVal {
				n = maxVal
			}
			v = n
		}
	}
	return v
}

// parseLimit extracts only the limit pagination parameter from the request query
// string. Use this for endpoints that do not support offset-based pagination
// (e.g. scored recommendation feeds). Invalid or out-of-range values silently
// fall back to safe defaults.
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	return parseBoundedQueryInt(r, "limit", defaultLimit, 1, maxLimit)
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
