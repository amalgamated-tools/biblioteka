// Package registry provides the default LLM provider factory map.
// It exists as a separate package to avoid an import cycle between
// internal/llm and the provider implementations (e.g., internal/llm/ollama).
package registry

import (
	"github.com/amalgamated-tools/biblioteka/internal/llm"
	"github.com/amalgamated-tools/biblioteka/internal/llm/ollama"
)

// DefaultFactories returns the built-in provider factory map.
// Both cmd/server and internal/server should use this instead of
// constructing the map inline, so new providers are added in one place.
func DefaultFactories() map[string]llm.Factory {
	return map[string]llm.Factory{
		llm.ProviderOllama: func(endpoint, model string) llm.Provider {
			return ollama.New(endpoint, model)
		},
	}
}
