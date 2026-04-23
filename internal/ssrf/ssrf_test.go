package ssrf_test

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/ssrf"
	"github.com/stretchr/testify/require"
)

// --- IsPrivateIP ---

func TestIsPrivateIP_LoopbackIPv4(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("127.0.0.1")))
}

func TestIsPrivateIP_RFC1918ClassA(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("10.0.0.1")))
}

func TestIsPrivateIP_RFC1918ClassB(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("172.16.0.1")))
}

func TestIsPrivateIP_RFC1918ClassC(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("192.168.1.1")))
}

func TestIsPrivateIP_AWSMetadata(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("169.254.169.254")))
}

func TestIsPrivateIP_SharedAddressSpace(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("100.64.0.1")))
}

func TestIsPrivateIP_IPv6Loopback(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("::1")))
}

func TestIsPrivateIP_IPv6LinkLocal(t *testing.T) {
	require.True(t, ssrf.IsPrivateIP(net.ParseIP("fe80::1")))
}

func TestIsPrivateIP_PublicIPv4(t *testing.T) {
	require.False(t, ssrf.IsPrivateIP(net.ParseIP("8.8.8.8")))
}

func TestIsPrivateIP_PublicIPv6(t *testing.T) {
	require.False(t, ssrf.IsPrivateIP(net.ParseIP("2001:4860:4860::8888")))
}

// --- ValidateURL ---

func TestValidateURL_ValidHTTPS(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://auth.example.com", "issuer_url", []string{"https"})
	require.NoError(t, err)
}

func TestValidateURL_ValidHTTP(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "http://ollama.example.com:11434", "endpoint", []string{"http", "https"})
	require.NoError(t, err)
}

func TestValidateURL_SchemeRejectedSingleAllowed(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "http://example.com", "issuer_url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "https")
}

func TestValidateURL_SchemeRejectedMultipleAllowed(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "gopher://example.com", "endpoint", []string{"http", "https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")
}

func TestValidateURL_UserinfoRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://user:pass@example.com", "issuer_url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "userinfo")
}

func TestValidateURL_UserinfoUsernameOnlyRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://admin@example.com", "issuer_url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "userinfo")
}

func TestValidateURL_MissingHost(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_IPv6ZoneIDRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://[fe80::1%25lo0]", "issuer_url", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "zone")
}

func TestValidateURL_LoopbackIPv4Rejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://127.0.0.1", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_AWSMetadataIPRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://169.254.169.254", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_RFC1918ClassARejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://10.0.0.1", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_RFC1918ClassBRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://172.16.0.1", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_RFC1918ClassCRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://192.168.1.1", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_IPv6LoopbackRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://[::1]", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_IPv6LinkLocalRejected(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://[fe80::1]", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_LocalhostRejectedViaDNS(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "https://localhost", "issuer_url", []string{"https"})
	require.Error(t, err)
}

func TestValidateURL_FieldNameAppearsInError(t *testing.T) {
	err := ssrf.ValidateURL(t.Context(), "ftp://example.com", "my_field", []string{"https"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "my_field")
}

// --- SafeHTTPClient ---

func TestSafeHTTPClient_NoTimeout(t *testing.T) {
	c := ssrf.SafeHTTPClient(0)
	require.NotNil(t, c)
	require.Equal(t, time.Duration(0), c.Timeout)
}

func TestSafeHTTPClient_WithTimeout(t *testing.T) {
	c := ssrf.SafeHTTPClient(5 * time.Minute)
	require.NotNil(t, c)
	require.Equal(t, 5*time.Minute, c.Timeout)
}

func TestSafeHTTPClient_TransportSet(t *testing.T) {
	c := ssrf.SafeHTTPClient(0)
	_, ok := c.Transport.(*http.Transport)
	require.True(t, ok, "transport must be *http.Transport")
}

func TestSafeHTTPClient_BlocksPrivateIP(t *testing.T) {
	c := ssrf.SafeHTTPClient(5 * time.Second)
	req, err := c.Get("http://127.0.0.1:11434/")
	if err == nil {
		req.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestSafeHTTPClient_BlocksAWSMetadata(t *testing.T) {
	c := ssrf.SafeHTTPClient(5 * time.Second)
	req, err := c.Get("http://169.254.169.254/")
	if err == nil {
		req.Body.Close()
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}
