package llm_test

import (
	"context"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/stretchr/testify/require"
)

// ── IsSupported ───────────────────────────────────────────────────────────────

func TestIsSupported_KnownProvider(t *testing.T) {
	require.True(t, llm.IsSupported(llm.ProviderOllama))
}

func TestIsSupported_UnknownProvider(t *testing.T) {
	require.False(t, llm.IsSupported("openai"))
}

func TestIsSupported_EmptyString(t *testing.T) {
	require.False(t, llm.IsSupported(""))
}

func TestIsSupported_PartialName(t *testing.T) {
	// "ollam" is not a valid provider name even though it is a prefix of "ollama".
	require.False(t, llm.IsSupported("ollam"))
}

// ── NewProvider ───────────────────────────────────────────────────────────────

// noopProvider is a minimal Provider implementation used only in tests.
type noopProvider struct{}

func (noopProvider) Generate(_ context.Context, _ string) (string, error) { return "", nil }

func TestNewProvider_KnownProvider(t *testing.T) {
	var calledEndpoint, calledModel string
	factories := map[string]llm.Factory{
		llm.ProviderOllama: func(endpoint, model string) llm.Provider {
			calledEndpoint = endpoint
			calledModel = model
			return noopProvider{}
		},
	}

	p, name, err := llm.NewProvider(llm.ProviderOllama, "http://localhost:11434", "llama3", factories)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, llm.ProviderOllama, name)
	require.Equal(t, "http://localhost:11434", calledEndpoint)
	require.Equal(t, "llama3", calledModel)
}

func TestNewProvider_UnknownProvider_ReturnsError(t *testing.T) {
	factories := map[string]llm.Factory{}

	p, name, err := llm.NewProvider("nonexistent", "http://localhost", "model", factories)
	require.Error(t, err)
	require.Nil(t, p)
	require.Equal(t, "nonexistent", name, "resolved name should still be returned on error")
	require.Contains(t, err.Error(), "unsupported LLM provider: nonexistent")
}

func TestNewProvider_EmptyName_DefaultsToOllama(t *testing.T) {
	factories := map[string]llm.Factory{
		llm.ProviderOllama: func(_, _ string) llm.Provider { return noopProvider{} },
	}

	p, name, err := llm.NewProvider("", "http://localhost:11434", "llama3", factories)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, llm.ProviderOllama, name, "empty providerName should resolve to ProviderOllama")
}

func TestNewProvider_NilFactories_ReturnsError(t *testing.T) {
	p, _, err := llm.NewProvider(llm.ProviderOllama, "http://localhost", "model", nil)
	require.Error(t, err)
	require.Nil(t, p)
}
