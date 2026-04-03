package goodreads

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewClient verifies that NewClient returns a fully initialized Client with
// non-empty token, host, and non-nil HTTP and GraphQL clients.
func TestNewClient(t *testing.T) {
	c := NewClient()
	require.NotNil(t, c)
	require.NotEmpty(t, c.token)
	require.NotEmpty(t, c.host)
	require.NotNil(t, c.client)
	require.NotNil(t, c.httpClient)

	t.Run("host is an HTTPS URL", func(t *testing.T) {
		require.Contains(t, c.host, "https://")
	})
	t.Run("token has non-trivial length", func(t *testing.T) {
		require.Greater(t, len(c.token), 10)
	})
}
