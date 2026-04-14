// Package llm provides interfaces and types for LLM-based book metadata enrichment.
package llm

import "context"

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
