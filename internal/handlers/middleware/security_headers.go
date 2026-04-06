package middleware

import "net/http"

// globalCSP is the Content-Security-Policy header value applied to all responses.
// Individual route handlers (such as the Swagger UI) may override this with
// more permissive or restrictive values for their particular use case.
// It permits the embedded frontend's inline theme bootstrap script and its
// Google Fonts stylesheet/font resources so the SPA continues to render as
// intended when this middleware is applied globally.
const globalCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data: https://fonts.gstatic.com;"

// SecurityHeadersMiddleware adds baseline HTTP security headers to every response.
// Individual route handlers (such as the Swagger UI) may override specific headers
// with more permissive or restrictive values for their particular use case.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", globalCSP)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
