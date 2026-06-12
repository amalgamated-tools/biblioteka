package otel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizePathForTelemetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "non-kobo path unchanged", in: "/api/books/123", want: "/api/books/123"},
		{name: "kobo token path redacted", in: "/kobo/secret-token/v1/library/sync", want: "/kobo/REDACTED/v1/library/sync"},
		{name: "kobo token only redacted", in: "/kobo/secret-token", want: "/kobo/REDACTED"},
		{name: "empty kobo token with remainder", in: "/kobo//v1/library/sync", want: "/kobo/REDACTED/v1/library/sync"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizePathForTelemetry(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
