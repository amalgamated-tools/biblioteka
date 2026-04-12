package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeFTS5Query_SingleWord(t *testing.T) {
	require.Equal(t, `"Foundation"*`, sanitizeFTS5Query("Foundation"))
}

func TestSanitizeFTS5Query_MultipleWords(t *testing.T) {
	require.Equal(t, `"desert"* "planet"*`, sanitizeFTS5Query("desert planet"))
}

func TestSanitizeFTS5Query_ExtraWhitespace(t *testing.T) {
	require.Equal(t, `"foo"* "bar"*`, sanitizeFTS5Query("  foo   bar  "))
}

func TestSanitizeFTS5Query_CasePreserved(t *testing.T) {
	// The FTS5 unicode61 tokenizer handles lowercasing; we preserve case here.
	require.Equal(t, `"FOUNDATION"*`, sanitizeFTS5Query("FOUNDATION"))
}

func TestSanitizeFTS5Query_EmbeddedDoubleQuotes(t *testing.T) {
	// Embedded " must be doubled inside an FTS5 phrase.
	require.Equal(t, `"say"* """hello"""*`, sanitizeFTS5Query(`say "hello"`))
}

func TestSanitizeFTS5Query_SpecialCharsOnlyReturnsEmpty(t *testing.T) {
	// Characters with no letter or digit produce no valid FTS5 terms.
	for _, q := range []string{"%", `\`, "*", "-", "^", "", "   "} {
		require.Equal(t, "", sanitizeFTS5Query(q), "query: %q", q)
	}
}

func TestSanitizeFTS5Query_MixedWordAndSpecialChars(t *testing.T) {
	// Words that contain at least one letter/digit are included; the tokenizer
	// strips the non-word characters at query time.
	require.Equal(t, `"hello%"*`, sanitizeFTS5Query("hello%"))
	require.Equal(t, `"100%"*`, sanitizeFTS5Query("100%"))
}

func TestSanitizeFTS5Query_EmptyString(t *testing.T) {
	require.Equal(t, "", sanitizeFTS5Query(""))
}

func TestContainsWordChar(t *testing.T) {
	require.True(t, containsWordChar("hello"))
	require.True(t, containsWordChar("123"))
	require.True(t, containsWordChar("hello_world")) // contains letters
	require.True(t, containsWordChar("100%"))        // contains digits
	require.False(t, containsWordChar("%"))
	require.False(t, containsWordChar(`\`))
	require.False(t, containsWordChar("*"))
	require.False(t, containsWordChar("-"))
	require.False(t, containsWordChar(""))
}
