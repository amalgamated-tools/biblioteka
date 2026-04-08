package auth

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter implements a per-IP token-bucket rate limiter.
type RateLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*visitor
	nextCleanup time.Time

	rate           float64 // tokens added per second
	burst          int     // max tokens (bucket size)
	cleanup        time.Duration
	trustedProxies []*net.IPNet
}

// NewRateLimiter creates a rate limiter that allows `rate` requests per second
// with a maximum burst size. Stale entries are cleaned up periodically.
// X-Forwarded-For is ignored; use NewRateLimiterWithTrustedProxies to enable it.
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return newRateLimiter(rate, burst, nil)
}

// NewRateLimiterWithTrustedProxies creates a rate limiter that trusts the
// X-Forwarded-For header only when the direct peer (RemoteAddr) is within one
// of the given CIDR ranges. When trusted, the rightmost non-trusted IP in the
// chain is used as the client IP. This prevents spoofing by untrusted clients.
func NewRateLimiterWithTrustedProxies(rate float64, burst int, trustedProxies []*net.IPNet) *RateLimiter {
	return newRateLimiter(rate, burst, trustedProxies)
}

func newRateLimiter(rate float64, burst int, trustedProxies []*net.IPNet) *RateLimiter {
	cleanup := 5 * time.Minute
	return &RateLimiter{
		visitors:       make(map[string]*visitor),
		nextCleanup:    time.Now().Add(cleanup),
		rate:           rate,
		burst:          burst,
		cleanup:        cleanup,
		trustedProxies: trustedProxies,
	}
}

func (rl *RateLimiter) cleanupVisitors(now time.Time) {
	staleBefore := now.Add(-rl.cleanup)
	for key, v := range rl.visitors {
		if v.lastSeen.Before(staleBefore) {
			delete(rl.visitors, key)
		}
	}
}

// allow checks whether the given key (IP) is allowed to proceed.
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if !now.Before(rl.nextCleanup) {
		rl.cleanupVisitors(now)
		rl.nextCleanup = now.Add(rl.cleanup)
	}

	v, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitor{
			tokens:   float64(rl.burst) - 1,
			lastSeen: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * rl.rate
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}

	v.tokens--
	return true
}

// ipFromRequest extracts the client IP from RemoteAddr. X-Forwarded-For is
// ignored; use ipFromRequestTrusted when the server is behind trusted proxies.
func ipFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// isTrusted reports whether ip falls within any of the given CIDR ranges.
func isTrusted(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// ipFromRequestTrusted extracts the client IP by walking the X-Forwarded-For
// chain from right to left, skipping IPs that fall within trustedProxies. It
// only inspects X-Forwarded-For when the direct peer (RemoteAddr) is itself
// trusted. Returns the first non-trusted IP, or RemoteAddr if none is found.
func ipFromRequestTrusted(r *http.Request, trustedProxies []*net.IPNet) string {
	remoteHost := ipFromRequest(r)
	remoteIP := net.ParseIP(remoteHost)

	// If RemoteAddr is not a trusted proxy, X-Forwarded-For cannot be trusted.
	if remoteIP == nil || !isTrusted(remoteIP, trustedProxies) {
		return remoteHost
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remoteHost
	}

	// Split X-Forwarded-For into individual IPs, walk from right to left.
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if candidate == "" {
			continue
		}
		ip := net.ParseIP(candidate)
		if ip == nil {
			// Unparseable entry; treat as untrusted client IP.
			return candidate
		}
		if !isTrusted(ip, trustedProxies) {
			return candidate
		}
	}

	// Every IP in the chain was trusted — fall back to RemoteAddr.
	return remoteHost
}

// ParseTrustedProxyCIDRs parses a comma-separated list of CIDR strings into
// a slice of *net.IPNet. It returns an error for any invalid CIDR notation.
func ParseTrustedProxyCIDRs(raw string) ([]*net.IPNet, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	cidrs := make([]*net.IPNet, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(p)
		if err != nil {
			return nil, err
		}
		cidrs = append(cidrs, cidr)
	}
	return cidrs, nil
}

// Limit wraps an http.HandlerFunc with per-IP rate limiting.
func (rl *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ip string
		if len(rl.trustedProxies) > 0 {
			ip = ipFromRequestTrusted(r, rl.trustedProxies)
		} else {
			ip = ipFromRequest(r)
		}
		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			slog.InfoContext(r.Context(), "rate limit exceeded", slog.String(otelkeys.IP, ip))
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, please try again later"})
			return
		}
		next(w, r)
	}
}
