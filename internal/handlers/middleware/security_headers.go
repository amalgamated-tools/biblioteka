package middleware

import "net/http"

// globalCSP is the Content-Security-Policy header value applied to all responses.
// Individual route handlers (such as the Swagger UI) may override this with
// more permissive or restrictive values for their particular use case.
// It permits the embedded frontend's inline theme bootstrap script and its
// Google Fonts stylesheet/font resources so the SPA continues to render as
// intended when this middleware is applied globally.
const globalCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data: https://fonts.gstatic.com;"

// hsts is the Strict-Transport-Security header value used when secure cookies
// are enabled. Two years max-age with includeSubDomains protects all
// subdomains against protocol-downgrade attacks.
const hsts = "max-age=63072000; includeSubDomains"

// SecurityHeadersConfig holds configuration for the security headers middleware.
type SecurityHeadersConfig struct {
	// SecureCookies controls whether HTTPS-only headers such as
	// Strict-Transport-Security are emitted. Set to true for any
	// deployment that terminates TLS (i.e. SECURE_COOKIES=true).
	SecureCookies bool
}

// NewSecurityHeadersMiddleware returns a middleware that adds baseline HTTP
// security headers to every response. When cfg.SecureCookies is true,
// a Strict-Transport-Security (HSTS) header is also set to protect against
// protocol-downgrade attacks on HTTPS deployments.
// Individual route handlers (such as the Swagger UI) may override specific
// headers with more permissive or restrictive values for their use case.
func NewSecurityHeadersMiddleware(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", globalCSP)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if cfg.SecureCookies {
				w.Header().Set("Strict-Transport-Security", hsts)
			}
			next.ServeHTTP(w, r)
		})
	}
}
