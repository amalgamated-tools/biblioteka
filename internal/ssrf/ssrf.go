// Package ssrf provides shared SSRF (Server-Side Request Forgery) protection
// utilities used across the application.
package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// dnsLookupTimeout is the maximum time allowed for a DNS lookup during URL
// validation.
const dnsLookupTimeout = 2 * time.Second

// privateIPNets lists IP networks that must never be targeted by an outbound
// request (SSRF protection). This is the single authoritative list shared
// across all packages; callers use IsPrivateIP instead of accessing it directly.
var privateIPNets = func() []*net.IPNet {
	var nets []*net.IPNet
	for _, cidr := range []string{
		"10.0.0.0/8",     // RFC-1918 class A private
		"172.16.0.0/12",  // RFC-1918 class B private
		"192.168.0.0/16", // RFC-1918 class C private
		"127.0.0.0/8",    // IPv4 loopback
		"169.254.0.0/16", // IPv4 link-local (incl. AWS IMDS 169.254.169.254)
		"0.0.0.0/8",      // "this" network
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique-local (fc00:: / fd00::)
		"fe80::/10",      // IPv6 link-local
		"::/128",         // IPv6 unspecified
		"100.64.0.0/10",  // Shared address space (RFC 6598)
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("ssrf: invalid CIDR %q: %v", cidr, err))
		}
		nets = append(nets, n)
	}
	return nets
}()

// IsPrivateIP reports whether ip falls within any private, loopback, or
// link-local range (RFC-1918, RFC-6598, and similar).
func IsPrivateIP(ip net.IP) bool {
	for _, n := range privateIPNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// SafeHTTPClient returns an *http.Client whose dialer validates that the
// resolved IP address is not in a private/loopback/link-local range. This
// prevents DNS rebinding attacks where a hostname passes the initial
// validation but resolves to a private address at connect time.
//
// Pass a non-zero timeout to set a deadline on the whole HTTP transaction
// (e.g. 5*time.Minute for the Ollama client). Pass 0 for no client-level
// timeout (the OIDC client relies on per-request context deadlines instead).
func SafeHTTPClient(timeout time.Duration) *http.Client {
	baseDialer := &net.Dialer{}
	safeDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		var safeIP string
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if IsPrivateIP(ip) {
				return nil, fmt.Errorf("refusing to connect to private address %s", ipStr)
			}
			if safeIP == "" {
				safeIP = ipStr
			}
		}
		if safeIP == "" {
			return nil, fmt.Errorf("no valid addresses for host %s", host)
		}
		// Connect directly to the validated IP — never re-resolve the hostname.
		return baseDialer.DialContext(ctx, network, net.JoinHostPort(safeIP, port))
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{DialContext: safeDialContext},
	}
}

// ValidateURL validates rawURL for SSRF risks. fieldName is used in error
// messages (e.g. "issuer_url", "endpoint"). schemes is the list of permitted
// URL schemes (e.g. []string{"https"} or []string{"http", "https"}).
//
// The following checks are applied:
//  1. URL must parse without error.
//  2. Scheme must be one of the permitted schemes.
//  3. Userinfo (user:password) is rejected to prevent credential leakage.
//  4. Host must be non-empty.
//  5. IPv6 literals with zone identifiers are rejected.
//  6. Literal private/loopback/link-local IP addresses are blocked.
//  7. If the host is a DNS name, it is resolved (with a bounded timeout) and
//     any private/loopback/link-local address in the result is also blocked.
//
// DNS errors are intentionally swallowed (fail-open): the SSRF-safe dialer in
// SafeHTTPClient provides a second layer of defense at connect time.
func ValidateURL(ctx context.Context, rawURL, fieldName string, schemes []string) error {
	if len(schemes) == 0 {
		return fmt.Errorf("%s: at least one permitted scheme must be specified", fieldName)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", fieldName, err)
	}

	validScheme := false
	for _, s := range schemes {
		if u.Scheme == s {
			validScheme = true
			break
		}
	}
	if !validScheme {
		schemeList := strings.Join(schemes, " or ")
		return fmt.Errorf("%s must use the %s scheme", fieldName, schemeList)
	}

	if u.User != nil {
		return errors.New(fieldName + " must not contain userinfo (credentials)")
	}

	host := u.Hostname()
	if host == "" {
		return errors.New(fieldName + " must include a host")
	}

	// Reject IPv6 literals with zone identifiers (e.g. "fe80::1%lo0") which
	// can bypass net.ParseIP and fall through to DNS resolution.
	if strings.Contains(host, "%") {
		return errors.New(fieldName + " must not contain an IPv6 zone identifier")
	}

	// Block literal private/loopback/link-local IP addresses directly in the URL.
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateIP(ip) {
			return errors.New(fieldName + " must not point to a private, loopback, or link-local address")
		}
		return nil // a routable literal IP is accepted; no DNS lookup needed
	}

	// Resolve the hostname and block any private/loopback/link-local result.
	// Use a short timeout so a slow/hanging DNS server cannot block the
	// request indefinitely.
	//
	// DNS errors (timeout, NXDOMAIN, etc.) are intentionally swallowed here
	// (fail-open). The SSRF-safe dialer in SafeHTTPClient provides a second
	// layer of defense at connect time.
	dnsCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(dnsCtx, host)
	if err == nil {
		for _, addr := range addrs {
			if ip := net.ParseIP(addr); ip != nil && IsPrivateIP(ip) {
				return errors.New(fieldName + " must not resolve to a private, loopback, or link-local address")
			}
		}
	}
	return nil
}
