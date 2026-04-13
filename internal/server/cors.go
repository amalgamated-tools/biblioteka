package server

import (
	"net/http"
	"strings"
)

// corsAllowedHeaders is the fixed list of request headers the browser extension
// is permitted to send in cross-origin requests.
const corsAllowedHeaders = "Authorization, Content-Type, X-Request-ID"

// corsAllowedMethods is the fixed list of HTTP methods permitted by the CORS
// preflight check.
const corsAllowedMethods = "GET, POST, OPTIONS"

// corsMaxAge is the preflight cache duration in seconds (24 hours).
const corsMaxAge = "86400"

// corsMiddleware returns middleware that adds CORS headers for the supplied set
// of allowed origins. An empty allowedOrigins set is a no-op; the middleware
// still executes but never emits Access-Control-* response headers, so browser
// cross-origin requests remain blocked by default.
//
// Only requests with an Origin header that appears verbatim in allowedOrigins
// receive permissive headers — this prevents a misconfigured wildcard from
// opening every endpoint to cross-origin access. Preflight OPTIONS requests
// from allowed origins receive a 204 No Content response; OPTIONS from
// disallowed or absent origins are passed through to the next handler so the
// underlying route can return its normal response (e.g. 405 Method Not Allowed).
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
					w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
					w.Header().Set("Access-Control-Max-Age", corsMaxAge)
					w.Header().Add("Vary", "Origin")

					// Short-circuit OPTIONS preflight for allowed origins only.
					if r.Method == http.MethodOptions {
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
