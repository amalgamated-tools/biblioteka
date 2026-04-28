package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValidateURLForSSRF_FieldNameInErrors verifies that the field parameter
// is used in all error message prefixes.
func TestValidateURLForSSRF_FieldNameInErrors(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		field   string
		schemes []string
		want    string
	}{
		{
			name:    "wrong scheme uses field name",
			rawURL:  "ftp://example.com",
			field:   "my_url",
			schemes: []string{"https"},
			want:    "my_url",
		},
		{
			name:    "userinfo uses field name",
			rawURL:  "https://user:pass@example.com",
			field:   "my_url",
			schemes: []string{"https"},
			want:    "my_url",
		},
		{
			name:    "missing host uses field name",
			rawURL:  "https://",
			field:   "my_url",
			schemes: []string{"https"},
			want:    "my_url",
		},
		{
			name:    "IPv6 zone ID uses field name",
			rawURL:  "https://[fe80::1%25lo0]",
			field:   "my_url",
			schemes: []string{"https"},
			want:    "my_url",
		},
		{
			name:    "private IP uses field name",
			rawURL:  "https://192.168.1.1",
			field:   "my_url",
			schemes: []string{"https"},
			want:    "my_url",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateURLForSSRF(t.Context(), tc.rawURL, tc.field, tc.schemes)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestValidateURLForSSRF_SingleScheme verifies that a single-scheme list
// is accepted correctly and rejected for non-matching schemes.
func TestValidateURLForSSRF_SingleScheme(t *testing.T) {
	err := validateURLForSSRF(t.Context(), "https://example.com", "url", []string{"https"})
	require.NoError(t, err)

	err = validateURLForSSRF(t.Context(), "http://example.com", "url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "https")
}

// TestValidateURLForSSRF_MultiScheme verifies that a multi-scheme list
// accepts all listed schemes and rejects others.
func TestValidateURLForSSRF_MultiScheme(t *testing.T) {
	err := validateURLForSSRF(t.Context(), "http://example.com", "url", []string{"http", "https"})
	require.NoError(t, err)

	err = validateURLForSSRF(t.Context(), "https://example.com", "url", []string{"http", "https"})
	require.NoError(t, err)

	err = validateURLForSSRF(t.Context(), "ftp://example.com", "url", []string{"http", "https"})
	require.Error(t, err)
	// error message lists both allowed schemes
	require.Contains(t, err.Error(), "http")
	require.Contains(t, err.Error(), "https")
}

// TestValidateURLForSSRF_PrivateIPBlocked verifies common private ranges are blocked.
func TestValidateURLForSSRF_PrivateIPBlocked(t *testing.T) {
	privateURLs := []string{
		"https://10.0.0.1",
		"https://172.16.0.1",
		"https://192.168.1.1",
		"https://127.0.0.1",
		"https://169.254.169.254",
		"https://[::1]",
		"https://[fe80::1]",
	}
	for _, u := range privateURLs {
		err := validateURLForSSRF(t.Context(), u, "url", []string{"https"})
		require.Error(t, err, "expected error for private URL %s", u)
	}
}

// TestValidateURLForSSRF_PublicIPAccepted verifies that routable public IP
// addresses are accepted (no DNS lookup needed for literal IPs).
func TestValidateURLForSSRF_PublicIPAccepted(t *testing.T) {
	err := validateURLForSSRF(t.Context(), "https://8.8.8.8", "url", []string{"https"})
	require.NoError(t, err)
}

// TestValidateURLForSSRF_IPv6ZoneIDRejected verifies that IPv6 zone identifiers
// are rejected regardless of the field name.
func TestValidateURLForSSRF_IPv6ZoneIDRejected(t *testing.T) {
	err := validateURLForSSRF(t.Context(), "https://[fe80::1%25lo0]", "url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zone")
}

// TestValidateURLForSSRF_LocalhostRejectedViaDNS verifies that "localhost"
// (a DNS name that resolves to the loopback address) is blocked.
func TestValidateURLForSSRF_LocalhostRejectedViaDNS(t *testing.T) {
	err := validateURLForSSRF(t.Context(), "https://localhost", "url", []string{"https"})
	require.Error(t, err)
}
