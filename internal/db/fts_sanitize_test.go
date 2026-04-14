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

// ---- buildILIKESearchWhere ----

func TestBuildILIKESearchWhere_EmptyQuery(t *testing.T) {
	where, args := buildILIKESearchWhere("", 1)
	require.Empty(t, where)
	require.Nil(t, args)
}

func TestBuildILIKESearchWhere_WhitespaceOnly(t *testing.T) {
	where, args := buildILIKESearchWhere("   ", 1)
	require.Empty(t, where)
	require.Nil(t, args)
}

func TestBuildILIKESearchWhere_SingleToken(t *testing.T) {
	where, args := buildILIKESearchWhere("Foundation", 1)
	require.Equal(t, `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\')`, where)
	require.Equal(t, []any{"%Foundation%"}, args)
}

func TestBuildILIKESearchWhere_MultipleTokens_ANDSemantics(t *testing.T) {
	where, args := buildILIKESearchWhere("desert planet", 1)
	expected := `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\') AND (title ILIKE $2 ESCAPE '\' OR description ILIKE $2 ESCAPE '\')`
	require.Equal(t, expected, where)
	require.Equal(t, []any{"%desert%", "%planet%"}, args)
}

func TestBuildILIKESearchWhere_ThreeTokens(t *testing.T) {
	where, args := buildILIKESearchWhere("a b c", 1)
	expected := `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\') AND (title ILIKE $2 ESCAPE '\' OR description ILIKE $2 ESCAPE '\') AND (title ILIKE $3 ESCAPE '\' OR description ILIKE $3 ESCAPE '\')`
	require.Equal(t, expected, where)
	require.Equal(t, []any{"%a%", "%b%", "%c%"}, args)
}

func TestBuildILIKESearchWhere_SpecialCharsEscaped(t *testing.T) {
	where, args := buildILIKESearchWhere(`100%`, 1)
	require.Equal(t, `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\')`, where)
	require.Equal(t, []any{`%100\%%`}, args)
}

func TestBuildILIKESearchWhere_UnderscoreEscaped(t *testing.T) {
	where, args := buildILIKESearchWhere("hello_world", 1)
	require.Equal(t, `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\')`, where)
	require.Equal(t, []any{`%hello\_world%`}, args)
}

func TestBuildILIKESearchWhere_BackslashEscaped(t *testing.T) {
	where, args := buildILIKESearchWhere(`back\slash`, 1)
	require.Equal(t, `WHERE (title ILIKE $1 ESCAPE '\' OR description ILIKE $1 ESCAPE '\')`, where)
	require.Equal(t, []any{`%back\\slash%`}, args)
}

func TestBuildILIKESearchWhere_CustomStartIdx(t *testing.T) {
	// When called with startIdx=3, placeholders begin at $3.
	where, args := buildILIKESearchWhere("desert planet", 3)
	expected := `WHERE (title ILIKE $3 ESCAPE '\' OR description ILIKE $3 ESCAPE '\') AND (title ILIKE $4 ESCAPE '\' OR description ILIKE $4 ESCAPE '\')`
	require.Equal(t, expected, where)
	require.Equal(t, []any{"%desert%", "%planet%"}, args)
}

func TestBuildILIKESearchWhere_ExtraWhitespace(t *testing.T) {
	// Extra whitespace between tokens should produce the same result as a
	// single space.
	where1, args1 := buildILIKESearchWhere("desert planet", 1)
	where2, args2 := buildILIKESearchWhere("  desert   planet  ", 1)
	require.Equal(t, where1, where2)
	require.Equal(t, args1, args2)
}
