package goodreads

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

var (
	// DefaultToken is the default token used for authentication with the Goodreads API. It is decoded from a hex string and should be kept secret.
	DefaultToken = []byte{100, 161, 45, 120, 112, 103, 115, 100, 121, 107, 98, 114, 101, 103, 106, 104, 112, 114, 54, 101, 122, 113, 104, 117, 119}
	// DefaultHost is the default host URL for the Goodreads API. It is decoded from a hex string and should be kept secret.
	DefaultHost = []byte{104, 116, 116, 112, 115, 58, 47, 47, 107, 120, 98, 119, 109, 113, 111, 118, 54, 106, 103, 115, 51, 100, 97, 97, 97, 109, 98, 55, 52, 52, 121, 99, 117, 52, 46, 97, 112, 112, 115, 121, 110, 99, 45, 97, 112, 105, 46, 117, 115, 45, 101, 97, 115, 116, 45, 49, 46, 97, 109, 97, 122, 111, 110, 97, 119, 115, 46, 99, 111, 109, 47, 103, 114, 97, 112, 104, 113, 108}
)

type Client struct {
	Token  string
	Host   string
	client graphql.Client
}

func NewClient() *Client {
	return &Client{
		Token: string(DefaultToken),
		Host:  string(DefaultHost),
		client: graphql.NewClient(
			string(DefaultHost),
			&http.Client{
				Transport: &AuthTransport{
					Token:            DefaultToken,
					WrappedTransport: http.DefaultTransport,
				},
			},
		),
	}
}
