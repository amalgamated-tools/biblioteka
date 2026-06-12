package middleware

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactRequestURL(t *testing.T) {
	t.Parallel()

	raw := "https://example.com/kobo/secret-token/v1/library/sync?x=1"
	parsed, err := url.Parse(raw)
	require.NoError(t, err)

	got := redactRequestURL(parsed)
	want := "https://example.com/kobo/REDACTED/v1/library/sync?x=1"
	require.Equal(t, want, got)
}
