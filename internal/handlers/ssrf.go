package handlers

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// privateIPNets lists IP networks that must never be targeted by an outbound
// request from Biblioteka (SSRF protection).
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
			panic(fmt.Sprintf("handlers: invalid CIDR %q: %v", cidr, err))
		}
		nets = append(nets, n)
	}
	return nets
}()

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateIPNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// dnsLookupTimeout is the maximum time allowed for a DNS lookup during URL
// validation.
const dnsLookupTimeout = 2 * time.Second

// validateURLForSSRF checks rawURL against SSRF attack vectors.
// field is used in error messages (e.g. "endpoint" or "issuer_url").
// allowedSchemes lists the permitted URL schemes (e.g. []string{"https"} or []string{"http", "https"}).
//
// The following checks are performed:
//   - only schemes in allowedSchemes are permitted
//   - userinfo (user:password) in the URL is rejected to prevent credential leakage
//   - literal private/loopback/link-local IP addresses in the host are blocked
//   - IPv6 literals with zone identifiers are rejected
//   - if the host is a DNS name, it is resolved (with a bounded timeout) and any
//     private/loopback/link-local address in the result is also blocked
func validateURLForSSRF(ctx context.Context, rawURL, field string, allowedSchemes []string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", field, err)
	}
	schemeOK := false
	for _, s := range allowedSchemes {
		if u.Scheme == s {
			schemeOK = true
			break
		}
	}
	if !schemeOK {
		return fmt.Errorf("%s must use the %s scheme", field, strings.Join(allowedSchemes, " or "))
	}
	if u.User != nil {
		return fmt.Errorf("%s must not contain userinfo (credentials)", field)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%s must include a host", field)
	}

	// Reject IPv6 literals with zone identifiers (e.g. "fe80::1%lo0") which
	// can bypass net.ParseIP and fall through to DNS resolution.
	if strings.Contains(host, "%") {
		return fmt.Errorf("%s must not contain an IPv6 zone identifier", field)
	}

	// Block literal private/loopback/link-local IP addresses directly in the URL.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("%s must not point to a private, loopback, or link-local address", field)
		}
		return nil // a routable literal IP is accepted; no DNS lookup needed
	}

	// Resolve the hostname and block any private/loopback/link-local result.
	// Use a short timeout so a slow/hanging DNS server cannot block the
	// request indefinitely.
	//
	// DNS errors (timeout, NXDOMAIN, etc.) are intentionally swallowed here
	// (fail-open). This preserves availability: a connectivity problem will be
	// surfaced when the actual request is made, and the SSRF-safe dialer
	// provides a second layer of defense at connect time.
	dnsCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(dnsCtx, host)
	if err == nil {
		for _, addr := range addrs {
			if ip := net.ParseIP(addr); ip != nil && isPrivateIP(ip) {
				return fmt.Errorf("%s must not resolve to a private, loopback, or link-local address", field)
			}
		}
	}
	return nil
}
