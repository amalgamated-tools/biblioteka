package goodreads

import "net/http"

// GoodReadsAuthTransport implements http.RoundTripper for custom authentication
type GoodReadsAuthTransport struct {
	Token []byte
	// WrappedTransport is the next http.RoundTripper in the chain
	WrappedTransport http.RoundTripper
}

func (t *GoodReadsAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())
	req.Header.Set("X-Api-Key", string(t.Token))

	// Execute the request using the wrapped transport
	return t.WrappedTransport.RoundTrip(req)
}
