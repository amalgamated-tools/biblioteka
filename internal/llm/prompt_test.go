package llm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildEnrichPrompt_BasicPromptContents(t *testing.T) {
	prompt := BuildEnrichPrompt("Dune", []string{"Frank Herbert"}, "A science fiction epic.")
	require.Contains(t, prompt, "Dune")
	require.Contains(t, prompt, "Frank Herbert")
	require.Contains(t, prompt, "A science fiction epic.")
	// Prompt must ask for the required JSON fields.
	require.Contains(t, prompt, "suggested_tags")
	require.Contains(t, prompt, "generated_description")
	require.Contains(t, prompt, "reading_level")
}

func TestBuildEnrichPrompt_NoAuthors(t *testing.T) {
	prompt := BuildEnrichPrompt("Unknown Origin", []string{}, "")
	require.Contains(t, prompt, "Unknown")
	// Empty slice → fallback to "Unknown"
	require.NotContains(t, prompt, "[]")
}

func TestBuildEnrichPrompt_MultipleAuthors(t *testing.T) {
	prompt := BuildEnrichPrompt("Good Omens", []string{"Terry Pratchett", "Neil Gaiman"}, "")
	require.Contains(t, prompt, "Terry Pratchett, Neil Gaiman")
}

func TestBuildEnrichPrompt_EmptyDescription(t *testing.T) {
	prompt := BuildEnrichPrompt("Some Book", []string{"An Author"}, "")
	// Empty description → placeholder shown to the model so it doesn't see a blank field.
	require.Contains(t, prompt, "(none)")
}

func TestBuildEnrichPrompt_NonEmptyDescription(t *testing.T) {
	prompt := BuildEnrichPrompt("Some Book", []string{"An Author"}, "A wonderful story.")
	require.Contains(t, prompt, "A wonderful story.")
	require.NotContains(t, prompt, "(none)")
}

// TestBuildEnrichPrompt_HTMLEscaping verifies that special HTML characters in
// book metadata are escaped before being embedded in the XML-like prompt
// template. Without escaping, characters like '<' and '>' would break the
// <book> element structure and could confuse the model.
func TestBuildEnrichPrompt_HTMLEscaping(t *testing.T) {
	prompt := BuildEnrichPrompt(
		"<Script> & 'Quotes'",
		[]string{"Author <One>"},
		"Description with <b>bold</b> & \"quotes\"",
	)

	// Raw HTML characters must not appear unescaped.
	require.NotContains(t, prompt, "<Script>")
	require.NotContains(t, prompt, "<One>")
	require.NotContains(t, prompt, "<b>bold</b>")

	// HTML entities must be present.
	require.Contains(t, prompt, "&lt;Script&gt;")
	require.Contains(t, prompt, "&lt;One&gt;")
	require.Contains(t, prompt, "&amp;")
}

func TestBuildEnrichPrompt_StructurePreserved(t *testing.T) {
	prompt := BuildEnrichPrompt("Title", []string{"Author"}, "Desc")
	// The <book> XML wrapper must remain intact after escaping.
	require.True(t, strings.Contains(prompt, "<book>"), "expected <book> opening tag")
	require.True(t, strings.Contains(prompt, "</book>"), "expected </book> closing tag")
	require.True(t, strings.Contains(prompt, "<title>"), "expected <title> tag")
	require.True(t, strings.Contains(prompt, "<authors>"), "expected <authors> tag")
	require.True(t, strings.Contains(prompt, "<description>"), "expected <description> tag")
}
