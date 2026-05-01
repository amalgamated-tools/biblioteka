// Package llm provides interfaces and types for LLM-based book metadata enrichment.
package llm

import (
	"context"
	"slices"
)

// ProviderOllama is the provider name for Ollama.
const ProviderOllama = "ollama"

// SupportedProviders lists the provider names the system can instantiate.
var SupportedProviders = []string{ProviderOllama}

// IsSupported reports whether the given provider name is supported.
func IsSupported(provider string) bool {
	return slices.Contains(SupportedProviders, provider)
}

// Provider generates LLM completions from a prompt.
type Provider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// EnrichmentResult is the structured JSON the prompt instructs the model to return.
type EnrichmentResult struct {
	Genres               []string `json:"genres"`
	Themes               []string `json:"themes"`
	Mood                 string   `json:"mood"`
	ReadingLevel         string   `json:"reading_level"` // children/young_adult/adult/academic
	SuggestedTags        []string `json:"suggested_tags"`
	GeneratedDescription string   `json:"generated_description"`
}
