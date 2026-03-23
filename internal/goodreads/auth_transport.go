package goodreads

import "net/http"

// GoodReadsAuthTransport implements http.RoundTripper for custom authentication.
// It only injects the X-Api-Key header for requests whose host matches
// AllowedHost, preventing the API key from leaking to third-party domains
// (e.g. the public goodreads.com autocomplete endpoint).
type GoodReadsAuthTransport struct {
	Token []byte
	// AllowedHost is the hostname (host:port or just host) that should receive
	// the API key header. Requests to any other host are forwarded without
	// authentication headers.
	AllowedHost string
	// WrappedTransport is the next http.RoundTripper in the chain
	WrappedTransport http.RoundTripper
}

func (t *GoodReadsAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only inject the API key for requests targeting the allowed host.
	if t.AllowedHost == "" || req.URL.Host == t.AllowedHost {
		req = req.Clone(req.Context())
		req.Header.Set("X-Api-Key", string(t.Token))
	}

	// Execute the request using the wrapped transport
	return t.WrappedTransport.RoundTrip(req)
}
