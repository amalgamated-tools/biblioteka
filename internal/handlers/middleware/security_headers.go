package middleware

import "net/http"

// globalCSP is the Content-Security-Policy header value applied to all responses.
// Individual route handlers (such as the Swagger UI) may override this with
// more permissive or restrictive values for their particular use case.
const globalCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data:;"

// SecurityHeadersMiddleware adds baseline HTTP security headers to every response.
// Individual route handlers (such as the Swagger UI) may override specific headers
// with more restrictive values for their particular use case.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", globalCSP)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
