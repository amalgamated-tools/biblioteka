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
}

// TestNewClient_HostContainsExpectedDomain verifies that the default host URL is
// non-empty and looks like an HTTPS URL (basic sanity check without hardcoding the
// obfuscated value).
func TestNewClient_HostContainsExpectedDomain(t *testing.T) {
	c := NewClient()
	require.Contains(t, c.host, "https://")
	// Token should be a non-trivial length string.
	require.Greater(t, len(c.token), 10)
}
