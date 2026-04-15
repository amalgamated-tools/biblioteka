package llm

import (
	"fmt"
	"html"
	"strings"
)

// BuildEnrichPrompt builds a prompt that instructs an LLM to return a JSON
// EnrichmentResult for the given book.
func BuildEnrichPrompt(title string, authors []string, existingDescription string) string {
	authorStr := "Unknown"
	if len(authors) > 0 {
		authorStr = strings.Join(authors, ", ")
	}
	desc := existingDescription
	if desc == "" {
		desc = "(none)"
	}
	return fmt.Sprintf(`You are a literary metadata assistant. Analyze the book described below and return a JSON object with the following fields:
- "genres": array of genre strings (e.g. ["Fiction", "Mystery"])
- "themes": array of thematic strings (e.g. ["Coming of Age", "Redemption"])
- "mood": a single mood descriptor (e.g. "Dark and suspenseful")
- "reading_level": one of "children", "young_adult", "adult", or "academic"
- "suggested_tags": array of concise tags suitable for library cataloging (5–10 tags)
- "generated_description": a 2–3 sentence description of the book suitable for a library catalog entry

Return ONLY valid JSON with no additional text, explanation, or markdown.

<book>
<title>%s</title>
<authors>%s</authors>
<description>%s</description>
</book>`, html.EscapeString(title), html.EscapeString(authorStr), html.EscapeString(desc))
}
