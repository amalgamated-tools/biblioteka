package llm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/stretchr/testify/require"
)

// stubSettingsReader is a simple map-based SettingsReader for tests.
type stubSettingsReader map[string]string

// stubProvider is a no-op Provider for tests.
type stubProvider struct{}

func (stubProvider) Generate(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s stubSettingsReader) GetSetting(_ context.Context, key string) (string, error) {
	v, ok := s[key]
	if !ok {
		return "", errors.New("setting not found")
	}
	return v, nil
}

var testKeys = llm.BootstrapSettings{
	Enabled:  "llm_enabled",
	Provider: "llm_provider",
	Endpoint: "llm_endpoint",
	Model:    "llm_model",
}

func TestBootstrap_Disabled(t *testing.T) {
	settings := stubSettingsReader{
		"llm_enabled": "false",
	}
	result := llm.Bootstrap(context.Background(), settings, testKeys, nil)
	require.Nil(t, result.Provider)
	require.Empty(t, result.ProviderName)
	require.Empty(t, result.ModelName)
}

func TestBootstrap_NotConfigured(t *testing.T) {
	// Enabled but no endpoint → no provider.
	settings := stubSettingsReader{
		"llm_enabled":  "true",
		"llm_provider": "ollama",
		"llm_endpoint": "",
		"llm_model":    "llama3",
	}
	result := llm.Bootstrap(context.Background(), settings, testKeys, nil)
	require.Nil(t, result.Provider)
	require.Equal(t, "ollama", result.ProviderName)
	require.Equal(t, "llama3", result.ModelName)
}

func TestBootstrap_UnknownProvider(t *testing.T) {
	settings := stubSettingsReader{
		"llm_enabled":  "true",
		"llm_provider": "nonexistent",
		"llm_endpoint": "http://localhost:11434",
		"llm_model":    "llama3",
	}
	result := llm.Bootstrap(context.Background(), settings, testKeys, map[string]llm.Factory{})
	require.Nil(t, result.Provider)
	require.Equal(t, "nonexistent", result.ProviderName)
	require.Equal(t, "llama3", result.ModelName)
}

func TestBootstrap_KnownProvider(t *testing.T) {
	settings := stubSettingsReader{
		"llm_enabled":  "true",
		"llm_provider": "ollama",
		"llm_endpoint": "http://localhost:11434",
		"llm_model":    "llama3",
	}
	var called bool
	factories := map[string]llm.Factory{
		llm.ProviderOllama: func(endpoint, model string) llm.Provider {
			called = true
			return &stubProvider{}
		},
	}
	result := llm.Bootstrap(context.Background(), settings, testKeys, factories)
	require.True(t, called)
	require.NotNil(t, result.Provider)
	require.Equal(t, llm.ProviderOllama, result.ProviderName)
	require.Equal(t, "llama3", result.ModelName)
}

func TestBootstrap_DefaultProvider(t *testing.T) {
	// When provider setting is empty, Bootstrap defaults to "ollama".
	settings := stubSettingsReader{
		"llm_enabled":  "true",
		"llm_provider": "",
		"llm_endpoint": "http://localhost:11434",
		"llm_model":    "llama3",
	}
	factories := map[string]llm.Factory{
		llm.ProviderOllama: func(endpoint, model string) llm.Provider {
			return &stubProvider{}
		},
	}
	result := llm.Bootstrap(context.Background(), settings, testKeys, factories)
	require.NotNil(t, result.Provider)
	require.Equal(t, llm.ProviderOllama, result.ProviderName)
	require.Equal(t, "llama3", result.ModelName)
}

func TestBootstrap_InvalidSettings(t *testing.T) {
	// Missing key names in BootstrapSettings should return empty result.
	settings := stubSettingsReader{
		"llm_enabled": "true",
	}
	keys := llm.BootstrapSettings{Enabled: "llm_enabled"} // Provider, Endpoint, Model are empty
	result := llm.Bootstrap(context.Background(), settings, keys, nil)
	require.Nil(t, result.Provider)
	require.Empty(t, result.ProviderName)
	require.Empty(t, result.ModelName)
}
