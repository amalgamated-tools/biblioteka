package auth

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowsInitialRequests(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	for range 5 {
		require.True(t, rl.allow("127.0.0.1"))
	}
}

func TestRateLimiter_BlocksWhenBucketEmpty(t *testing.T) {
	rl := NewRateLimiter(0, 2) // rate=0 (no refill), burst=2

	// First call: new visitor, starts with burst-1=1 token, allowed.
	rl.allow("127.0.0.1")
	// Second call: tokens=1, consumes 1, tokens=0, allowed.
	rl.allow("127.0.0.1")
	// Third call: tokens=0 < 1, blocked.
	require.False(t, rl.allow("127.0.0.1"), "request should have been blocked after bucket is empty")
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	rl := NewRateLimiter(0, 1) // burst=1, no refill

	// First call for each IP uses burst, so first call is always allowed
	require.True(t, rl.allow("1.1.1.1"))
	require.True(t, rl.allow("2.2.2.2"))

	// Second calls should be blocked (tokens=0 after first call)
	require.False(t, rl.allow("1.1.1.1"), "second request from 1.1.1.1 should be blocked")
	require.False(t, rl.allow("2.2.2.2"), "second request from 2.2.2.2 should be blocked")
}

func TestIpFromRequest_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"

	ip := ipFromRequest(r)
	require.Equal(t, "192.168.1.1", ip)
}

func TestIpFromRequest_IgnoresXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"
	r.Header.Set("X-Forwarded-For", "10.0.0.1, 172.16.0.1")

	ip := ipFromRequest(r)
	require.Equal(t, "192.168.1.1", ip, "ipFromRequest must ignore X-Forwarded-For")
}

func mustParseCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, cidr, err := net.ParseCIDR(s)
	require.NoError(t, err)
	return cidr
}

func TestIpFromRequestTrusted_RemoteNotTrusted(t *testing.T) {
	trusted := []*net.IPNet{mustParseCIDR(t, "10.0.0.0/8")}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.5:9999"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")

	ip := ipFromRequestTrusted(r, trusted)
	require.Equal(t, "203.0.113.5", ip, "should ignore XFF when RemoteAddr is not trusted")
}

func TestIpFromRequestTrusted_SingleProxy(t *testing.T) {
	trusted := []*net.IPNet{mustParseCIDR(t, "10.0.0.0/8")}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:9999"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")

	ip := ipFromRequestTrusted(r, trusted)
	require.Equal(t, "203.0.113.50", ip)
}

func TestIpFromRequestTrusted_MultipleProxies(t *testing.T) {
	trusted := []*net.IPNet{
		mustParseCIDR(t, "10.0.0.0/8"),
		mustParseCIDR(t, "172.16.0.0/12"),
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:9999"
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 172.16.0.5, 10.0.0.2")

	ip := ipFromRequestTrusted(r, trusted)
	require.Equal(t, "203.0.113.50", ip, "should return the rightmost non-trusted IP")
}

func TestIpFromRequestTrusted_AllTrusted(t *testing.T) {
	trusted := []*net.IPNet{mustParseCIDR(t, "0.0.0.0/0")}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:9999"
	r.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.3")

	ip := ipFromRequestTrusted(r, trusted)
	require.Equal(t, "10.0.0.1", ip, "should fall back to RemoteAddr when all XFF IPs are trusted")
}

func TestIpFromRequestTrusted_NoXFF(t *testing.T) {
	trusted := []*net.IPNet{mustParseCIDR(t, "10.0.0.0/8")}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:9999"

	ip := ipFromRequestTrusted(r, trusted)
	require.Equal(t, "10.0.0.1", ip, "should return RemoteAddr when XFF is absent")
}

func TestIpFromRequestTrusted_SpoofedXFF_IgnoredWithoutTrust(t *testing.T) {
	// No trusted proxies — even if someone sets XFF, it's ignored.
	rl := NewRateLimiter(0, 1) // burst=1, no refill

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	limited := rl.Limit(handler)

	// First request from 1.2.3.4 — allowed.
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "1.2.3.4:9999"
	r1.Header.Set("X-Forwarded-For", "99.99.99.1")
	w1 := httptest.NewRecorder()
	limited(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	// Second request from same RemoteAddr but different XFF — should still be blocked.
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "1.2.3.4:9999"
	r2.Header.Set("X-Forwarded-For", "99.99.99.2")
	w2 := httptest.NewRecorder()
	limited(w2, r2)
	require.Equal(t, http.StatusTooManyRequests, w2.Code, "spoofed XFF should not bypass rate limit")
}

func TestParseTrustedProxyCIDRs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		count   int
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"single", "10.0.0.0/8", 1, false},
		{"multiple", "10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16", 3, false},
		{"with spaces", " 10.0.0.0/8 , 172.16.0.0/12 ", 2, false},
		{"invalid", "not-a-cidr", 0, true},
		{"partial invalid", "10.0.0.0/8, bad", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cidrs, err := ParseTrustedProxyCIDRs(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, cidrs, tt.count)
		})
	}
}

func TestRateLimiter_Limit_BlockedRequest(t *testing.T) {
	rl := NewRateLimiter(0, 1) // burst=1, no refill

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	limited := rl.Limit(handler)

	// First request: allowed
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "1.2.3.4:9999"
	w1 := httptest.NewRecorder()
	limited(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	// Second request: blocked
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "1.2.3.4:9999"
	w2 := httptest.NewRecorder()
	limited(w2, r2)
	require.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestRateLimiter_Limit_WithTrustedProxies(t *testing.T) {
	trusted := []*net.IPNet{mustParseCIDR(t, "10.0.0.0/8")}
	rl := NewRateLimiterWithTrustedProxies(0, 1, trusted) // burst=1, no refill

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	limited := rl.Limit(handler)

	// First request from client 203.0.113.1 via proxy 10.0.0.1 — allowed.
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "10.0.0.1:9999"
	r1.Header.Set("X-Forwarded-For", "203.0.113.1")
	w1 := httptest.NewRecorder()
	limited(w1, r1)
	require.Equal(t, http.StatusOK, w1.Code)

	// Second request from same client — blocked.
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "10.0.0.1:9999"
	r2.Header.Set("X-Forwarded-For", "203.0.113.1")
	w2 := httptest.NewRecorder()
	limited(w2, r2)
	require.Equal(t, http.StatusTooManyRequests, w2.Code)
}
