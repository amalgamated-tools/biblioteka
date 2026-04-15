package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

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
