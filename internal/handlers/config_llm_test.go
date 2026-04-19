package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// noopLLMEndpointURLValidator disables SSRF URL validation for tests that focus
// on other aspects of HandleLLMConfig (e.g., settings persistence). Tests for
// SSRF validation itself call validateLLMEndpointURL directly.
func noopLLMEndpointURLValidator(_ context.Context, _ string) error { return nil }

func setupLLMConfigTest(t *testing.T) (*ConfigHandler, string, string) {
	t.Helper()
	h, adminID, regularID := setupConfigHandler(t)
	require.NoError(t, h.DB.SetAdmin(t.Context(), adminID, true))
	return h, adminID, regularID
}

func TestHandleLLMConfig_GetDefault(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/llm", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var cfg LLMConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
	require.False(t, cfg.Enabled)
	require.Empty(t, cfg.Provider)
	require.Empty(t, cfg.Endpoint)
	require.Empty(t, cfg.Model)
}

func TestHandleLLMConfig_GetForbiddenForNonAdmin(t *testing.T) {
	h, _, regularID := setupLLMConfigTest(t)

	r := httptest.NewRequest(http.MethodGet, "/api/config/llm", nil)
	r = withUserID(r, regularID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleLLMConfig_PutValid(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)
	// Bypass URL validation: this test exercises settings persistence, not SSRF protection.
	h.LLMEndpointURLValidator = noopLLMEndpointURLValidator

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://localhost:11434",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var cfg LLMConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &cfg))
	require.True(t, cfg.Enabled)
	require.Equal(t, "ollama", cfg.Provider)
	require.Equal(t, "http://localhost:11434", cfg.Endpoint)
	require.Equal(t, "llama3", cfg.Model)
	require.True(t, cfg.RestartRequired)
}

func TestHandleLLMConfig_PutMissingEndpoint(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_PutMissingModel(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://localhost:11434",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_PutUnsupportedProvider(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "unsupported",
		Endpoint: "http://localhost:11434",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_PutWhitespaceEndpoint(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "   ",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_PutDisabledAllowsEmpty(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{Enabled: false})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleLLMConfig_MethodNotAllowed(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/config/llm", nil)
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// --- validateLLMEndpointURL ---

func TestValidateLLMEndpointURL_HttpSchemeAccepted(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://ollama.example.com:11434")
	require.NoError(t, err)
}

func TestValidateLLMEndpointURL_HttpsSchemeAccepted(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "https://ollama.example.com")
	require.NoError(t, err)
}

func TestValidateLLMEndpointURL_FileSchemeRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "file:///etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")
}

func TestValidateLLMEndpointURL_GopherSchemeRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "gopher://internal.host/")
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")
}

func TestValidateLLMEndpointURL_LoopbackIPv4Rejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://127.0.0.1:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_AWSMetadataIPRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://169.254.169.254")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_RFC1918ClassARejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://10.0.0.1:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_RFC1918ClassBRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://172.16.0.1:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_RFC1918ClassCRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://192.168.1.1:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_IPv6LoopbackRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://[::1]:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_IPv6LinkLocalRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://[fe80::1]:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_LocalhostRejectedViaDNS(t *testing.T) {
	// "localhost" is a DNS name (not a literal IP), but it resolves to the
	// loopback address; the DNS-resolution check must catch it.
	err := validateLLMEndpointURL(t.Context(), "http://localhost:11434")
	require.Error(t, err)
}

func TestValidateLLMEndpointURL_UserinfoRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://user:pass@ollama.example.com:11434")
	require.Error(t, err)
	require.Contains(t, err.Error(), "userinfo")
}

func TestValidateLLMEndpointURL_IPv6ZoneIDRejected(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://[fe80::1%25lo0]:11434")
	require.Error(t, err)
	require.Contains(t, err.Error(), "zone")
}

func TestValidateLLMEndpointURL_MissingHost(t *testing.T) {
	err := validateLLMEndpointURL(t.Context(), "http://")
	require.Error(t, err)
}

// --- Handler-level SSRF rejection tests ---

func TestHandleLLMConfig_SSRFPrivateIPRejected(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://192.168.1.1:11434",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_SSRFLoopbackIPRejected(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://127.0.0.1:11434",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_SSRFAWSMetadataIPRejected(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://169.254.169.254:80",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleLLMConfig_SSRFNotValidatedWhenDisabled(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	// When LLM is disabled, a private endpoint should be stored without validation.
	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://127.0.0.1:11434",
		Model:    "llama3",
		Enabled:  false,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleLLMConfig_ValidPublicEndpointAccepted(t *testing.T) {
	h, adminID, _ := setupLLMConfigTest(t)

	body := mustMarshal(t, LLMConfig{
		Provider: "ollama",
		Endpoint: "http://ollama.example.com:11434",
		Model:    "llama3",
		Enabled:  true,
	})
	r := httptest.NewRequest(http.MethodPut, "/api/config/llm", bytes.NewReader(body))
	r = withUserID(r, adminID)
	w := httptest.NewRecorder()

	h.HandleLLMConfig(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}
