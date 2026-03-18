package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowsInitialRequests(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	t.Cleanup(rl.Stop)

	for i := range 5 {
		if !rl.allow("127.0.0.1") {
			failf(t, "request %d should have been allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksWhenBucketEmpty(t *testing.T) {
	rl := NewRateLimiter(0, 2) // rate=0 (no refill), burst=2
	t.Cleanup(rl.Stop)

	// First call: new visitor, starts with burst-1=1 token, allowed.
	rl.allow("127.0.0.1")
	// Second call: tokens=1, consumes 1, tokens=0, allowed.
	rl.allow("127.0.0.1")
	// Third call: tokens=0 < 1, blocked.
	if rl.allow("127.0.0.1") {
		fail(t, "request should have been blocked after bucket is empty")
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	rl := NewRateLimiter(0, 1) // burst=1, no refill
	t.Cleanup(rl.Stop)

	// First call for each IP uses burst, so first call is always allowed
	if !rl.allow("1.1.1.1") {
		fail(t, "first request from 1.1.1.1 should be allowed")
	}
	if !rl.allow("2.2.2.2") {
		fail(t, "first request from 2.2.2.2 should be allowed")
	}

	// Second calls should be blocked (tokens=0 after first call)
	if rl.allow("1.1.1.1") {
		fail(t, "second request from 1.1.1.1 should be blocked")
	}
	if rl.allow("2.2.2.2") {
		fail(t, "second request from 2.2.2.2 should be blocked")
	}
}

func TestIpFromRequest_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"

	ip := ipFromRequest(r)
	if ip != "192.168.1.1" {
		failf(t, "ipFromRequest() = %q, want %q", ip, "192.168.1.1")
	}
}

func TestIpFromRequest_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 172.16.0.1")

	ip := ipFromRequest(r)
	if ip != "10.0.0.1" {
		failf(t, "ipFromRequest() = %q, want %q", ip, "10.0.0.1")
	}
}

func TestIpFromRequest_XForwardedFor_Single(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.2")

	ip := ipFromRequest(r)
	if ip != "10.0.0.2" {
		failf(t, "ipFromRequest() = %q, want %q", ip, "10.0.0.2")
	}
}

func TestRateLimiter_Limit_BlockedRequest(t *testing.T) {
	rl := NewRateLimiter(0, 1) // burst=1, no refill
	t.Cleanup(rl.Stop)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	limited := rl.Limit(handler)

	// First request: allowed
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "1.2.3.4:9999"
	w1 := httptest.NewRecorder()
	limited(w1, r1)
	if w1.Code != http.StatusOK {
		failf(t, "first request: expected 200, got %d", w1.Code)
	}

	// Second request: blocked
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "1.2.3.4:9999"
	w2 := httptest.NewRecorder()
	limited(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		failf(t, "second request: expected 429, got %d", w2.Code)
	}
}
