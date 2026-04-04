// Package goodreads provides a client for the Goodreads unpublished GraphQL
// API, supporting book searches by title/query, ISBN lookups, and direct book
// lookups by Goodreads ID, legacy ID, or ASIN.
package goodreads

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Khan/genqlient/graphql"
)

// These credentials are public and easily obtainable.
// They are stored as raw bytes only to hide them from search results.
var (
	defaultToken = []byte{
		0x64, 0x61, 0x32, 0x2d, 0x78, 0x70, 0x67, 0x73, 0x64, 0x79,
		0x64, 0x6b, 0x62, 0x72, 0x65, 0x67, 0x6a, 0x68, 0x70, 0x72,
		0x36, 0x65, 0x6a, 0x7a, 0x71, 0x64, 0x68, 0x75, 0x77, 0x79,
	}
	defaultHost = []byte{
		0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x6b, 0x78,
		0x62, 0x77, 0x6d, 0x71, 0x6f, 0x76, 0x36, 0x6a, 0x67, 0x67,
		0x33, 0x64, 0x61, 0x61, 0x61, 0x6d, 0x62, 0x37, 0x34,
		0x34, 0x79, 0x63, 0x75, 0x34, 0x2e, 0x61, 0x70, 0x70, 0x73,
		0x79, 0x6e, 0x63, 0x2d, 0x61, 0x70, 0x69, 0x2e, 0x75, 0x73,
		0x2d, 0x65, 0x61, 0x73, 0x74, 0x2d, 0x31, 0x2e, 0x61, 0x6d,
		0x61, 0x7a, 0x6f, 0x6e, 0x61, 0x77, 0x73, 0x2e, 0x63, 0x6f,
		0x6d, 0x2f, 0x67, 0x72, 0x61, 0x70, 0x68, 0x71, 0x6c,
	}
)

// HttpClient is the interface used by Client to make HTTP requests, allowing
// the real *http.Client to be replaced by a test double.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a Goodreads API client. Create one with NewClient.
type Client struct {
	token      string
	host       string
	client     graphql.Client
	httpClient HttpClient
}

// NewClient creates a Goodreads API client using the default public credentials
// and a 3-second HTTP timeout. The auth transport restricts the API key header
// to the Goodreads GraphQL endpoint to avoid leaking it to third-party domains.
func NewClient() *Client {
	host := string(defaultHost)

	// Extract the hostname so the auth transport only sends the API key
	// to the GraphQL endpoint and not to third-party domains (e.g. goodreads.com).
	var allowedHost string
	if parsed, err := url.Parse(host); err == nil {
		allowedHost = parsed.Host
	}

	httpClient := &http.Client{
		// 3 second timeout is more than enough for the Goodreads API, which is usually very fast.
		Timeout: 3 * time.Second,
		Transport: &GoodReadsAuthTransport{
			Token:            defaultToken,
			AllowedHost:      allowedHost,
			WrappedTransport: http.DefaultTransport,
		},
	}
	return &Client{
		token: string(defaultToken),
		host:  host,
		client: graphql.NewClient(
			host,
			httpClient,
		),
		httpClient: httpClient,
	}
}
