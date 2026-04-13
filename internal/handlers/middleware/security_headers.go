package middleware

import "net/http"

// themeBootstrapScriptCSPHash is the CSP hash-source token (including the
// "sha256-" algorithm prefix) for the inline theme-bootstrap script in
// frontend/index.html — the bare <script> block that reads biblioteka_theme
// from localStorage and toggles the "dark" class before the bundle loads.
//
// NOTE: The regex below matches a bare <script> tag (no attributes). If the
// bootstrap tag gains attributes, update the pattern accordingly.
//
// To regenerate after editing the script (run from the repository root):
//
//	python3 -c "import hashlib,base64,re; c=open('frontend/index.html').read(); s=re.search(r'<script>(.*?)</script>',c,re.DOTALL).group(1); print('sha256-'+base64.b64encode(hashlib.sha256(s.encode()).digest()).decode())"
const themeBootstrapScriptCSPHash = "sha256-fH8pmaGT8bEGA0OitMqoXdy+W8xbN89w8ghrDCdlrwA="

// globalCSP is the Content-Security-Policy header value applied to all responses.
// Individual route handlers (such as the Swagger UI) may override this with
// more permissive or restrictive values for their particular use case.
// It permits the embedded frontend's inline theme bootstrap script (via its
// SHA-256 hash) and Google Fonts stylesheet/font resources so the SPA continues
// to render as intended when this middleware is applied globally.
//
// No 'unsafe-inline' is present in either script-src or style-src:
//   - script-src uses the theme bootstrap script's SHA-256 hash instead.
//   - style-src omits 'unsafe-inline' because the initial frontend/index.html
//     response does not include inline <style> blocks or style= HTML
//     attributes; dynamic element styles set later by Svelte-compiled
//     JavaScript (e.g. dom.style.cssText) are CSSOM operations and are not
//     governed by style-src.
const globalCSP = "default-src 'self'; script-src 'self' '" + themeBootstrapScriptCSPHash + "'; style-src 'self' https://fonts.googleapis.com; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data: https://fonts.gstatic.com;"

// hsts is the Strict-Transport-Security header value used when secure cookies
// are enabled. Two years max-age with includeSubDomains protects all
// subdomains against protocol-downgrade attacks.
const hsts = "max-age=63072000; includeSubDomains"

// SecurityHeadersConfig holds configuration for the security headers middleware.
type SecurityHeadersConfig struct {
	// SecureCookies controls whether HTTPS-only headers such as
	// Strict-Transport-Security are emitted. Set to true when the
	// application is served over HTTPS externally, including when TLS
	// is terminated by an upstream reverse proxy or load balancer
	// (i.e. SECURE_COOKIES=true).
	SecureCookies bool
}

// NewSecurityHeadersMiddleware returns a middleware that adds baseline HTTP
// security headers to every response. When cfg.SecureCookies is true,
// a Strict-Transport-Security (HSTS) header is also set to protect against
// protocol-downgrade attacks for responses served over HTTPS externally.
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
