package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ParseEnrichmentResult extracts a JSON EnrichmentResult from a raw LLM
// response. It handles models that wrap the JSON in markdown code fences.
func ParseEnrichmentResult(raw string) (*EnrichmentResult, error) {
	s := strings.TrimSpace(raw)
	// Strip optional markdown code fence (```json ... ``` or ``` ... ```)
	if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		s = strings.TrimPrefix(s, "json")
		if end := strings.LastIndex(s, "```"); end != -1 {
			s = s[:end]
		}
	}
	s = strings.TrimSpace(s)
	// Find the first '{' and last '}' to extract just the JSON object.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || start >= end {
		return nil, errors.New("no JSON object found in response")
	}
	s = s[start : end+1]
	var result EnrichmentResult
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, fmt.Errorf("unmarshal enrichment result: %w", err)
	}
	return &result, nil
}
