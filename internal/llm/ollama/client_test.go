package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// plainHTTPClient returns a standard HTTP client without SSRF restrictions,
// for use in tests that need to reach loopback test servers.
func plainHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}

func TestClient_Generate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: `{"genres":["Fiction"]}`},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	client.HTTPClient = plainHTTPClient()
	content, err := client.Generate(t.Context(), "test prompt")
	require.NoError(t, err)
	require.Equal(t, `{"genres":["Fiction"]}`, content)
}

func TestClient_Generate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	client.HTTPClient = plainHTTPClient()
	_, err := client.Generate(t.Context(), "test prompt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status 500")
}

func TestClient_Generate_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: ""},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	client.HTTPClient = plainHTTPClient()
	_, err := client.Generate(t.Context(), "test prompt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty response content")
}

// --- SSRF dialer tests ---

func TestSSRFSafeHTTPClient_BlocksLoopbackIPv4(t *testing.T) {
	c := New("http://127.0.0.1:11434", "llama3")
	_, err := c.Generate(t.Context(), "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestSSRFSafeHTTPClient_BlocksAWSMetadataIP(t *testing.T) {
	c := New("http://169.254.169.254:80", "llama3")
	_, err := c.Generate(t.Context(), "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestSSRFSafeHTTPClient_BlocksRFC1918ClassA(t *testing.T) {
	c := New("http://10.0.0.1:11434", "llama3")
	_, err := c.Generate(t.Context(), "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestSSRFSafeHTTPClient_BlocksRFC1918ClassC(t *testing.T) {
	c := New("http://192.168.1.1:11434", "llama3")
	_, err := c.Generate(t.Context(), "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "private")
}

func TestSSRFSafeHTTPClient_BlocksLocalhost(t *testing.T) {
	// "localhost" resolves to the loopback address; the dialer must block it.
	c := New("http://localhost:11434", "llama3")
	_, err := c.Generate(t.Context(), "test")
	require.Error(t, err)
}

func TestClient_Generate_ResponseBodyTooLarge(t *testing.T) {
	largeContent := strings.Repeat("x", maxOllamaResponseBytes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: largeContent},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	defer srv.Close()

	client := New(srv.URL, "llama3")
	client.HTTPClient = plainHTTPClient()
	_, err := client.Generate(t.Context(), "test prompt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode response")
}
