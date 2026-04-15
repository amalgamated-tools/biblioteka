package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEnrichmentResult_CleanJSON(t *testing.T) {
	raw := `{"genres":["Fiction"],"themes":["Redemption"],"mood":"Dark","reading_level":"adult","suggested_tags":["classic","fiction"],"generated_description":"A great book."}`
	result, err := ParseEnrichmentResult(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"Fiction"}, result.Genres)
	require.Equal(t, []string{"Redemption"}, result.Themes)
	require.Equal(t, "Dark", result.Mood)
	require.Equal(t, "adult", result.ReadingLevel)
	require.Equal(t, []string{"classic", "fiction"}, result.SuggestedTags)
	require.Equal(t, "A great book.", result.GeneratedDescription)
}

func TestParseEnrichmentResult_MarkdownFence(t *testing.T) {
	raw := "```json\n{\"genres\":[\"Mystery\"],\"themes\":[],\"mood\":\"Tense\",\"reading_level\":\"young_adult\",\"suggested_tags\":[\"mystery\"],\"generated_description\":\"A mystery.\"}\n```"
	result, err := ParseEnrichmentResult(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"Mystery"}, result.Genres)
	require.Equal(t, "Tense", result.Mood)
}

func TestParseEnrichmentResult_PlainFence(t *testing.T) {
	raw := "```\n{\"genres\":[],\"themes\":[],\"mood\":\"\",\"reading_level\":\"children\",\"suggested_tags\":[],\"generated_description\":\"\"}\n```"
	result, err := ParseEnrichmentResult(raw)
	require.NoError(t, err)
	require.Equal(t, "children", result.ReadingLevel)
}

func TestParseEnrichmentResult_InvalidJSON(t *testing.T) {
	_, err := ParseEnrichmentResult("not json at all")
	require.Error(t, err)
}

func TestParseEnrichmentResult_EmptyResponse(t *testing.T) {
	_, err := ParseEnrichmentResult("")
	require.Error(t, err)
}
