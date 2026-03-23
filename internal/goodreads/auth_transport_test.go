package goodreads

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingTransport struct {
	lastRequest *http.Request
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lastRequest = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func TestAuthTransport_AddsKeyForAllowedHost(t *testing.T) {
	inner := &recordingTransport{}
	transport := &GoodReadsAuthTransport{
		Token:            []byte("secret-key"),
		AllowedHost:      "api.example.com",
		WrappedTransport: inner,
	}

	req, err := http.NewRequest("GET", "https://api.example.com/graphql", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, "secret-key", inner.lastRequest.Header.Get("X-Api-Key"))
}

func TestAuthTransport_OmitsKeyForDifferentHost(t *testing.T) {
	inner := &recordingTransport{}
	transport := &GoodReadsAuthTransport{
		Token:            []byte("secret-key"),
		AllowedHost:      "api.example.com",
		WrappedTransport: inner,
	}

	req, err := http.NewRequest("GET", "https://goodreads.com/book/auto_complete", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	require.Empty(t, inner.lastRequest.Header.Get("X-Api-Key"))
}

func TestAuthTransport_EmptyAllowedHostSendsKeyEverywhere(t *testing.T) {
	inner := &recordingTransport{}
	transport := &GoodReadsAuthTransport{
		Token:            []byte("secret-key"),
		AllowedHost:      "",
		WrappedTransport: inner,
	}

	req, err := http.NewRequest("GET", "https://any-host.example.com/path", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, "secret-key", inner.lastRequest.Header.Get("X-Api-Key"))
}

func TestAuthTransport_DoesNotMutateOriginalRequest(t *testing.T) {
	inner := &recordingTransport{}
	transport := &GoodReadsAuthTransport{
		Token:            []byte("secret-key"),
		AllowedHost:      "api.example.com",
		WrappedTransport: inner,
	}

	req, err := http.NewRequest("GET", "https://api.example.com/graphql", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req)
	require.NoError(t, err)

	// Original request should not have the header set
	require.Empty(t, req.Header.Get("X-Api-Key"))
	// But the forwarded request should
	require.Equal(t, "secret-key", inner.lastRequest.Header.Get("X-Api-Key"))
}
