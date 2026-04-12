package db

import (
	"strings"
	"unicode"
)

// sanitizeFTS5Query converts a user-supplied search string into a safe FTS5
// MATCH expression for SQLite's books_fts virtual table.
//
// Each whitespace-separated token is wrapped in FTS5 phrase syntax (double
// quotes) with a trailing * for prefix matching, so that a query for "found"
// also returns "Foundation". Tokens that contain no letter or digit are
// skipped: the unicode61 tokenizer would produce an empty phrase from them,
// which SQLite FTS5 rejects as a syntax error.
//
// Returns "" when no valid tokens remain; callers must treat this as "no
// results" and skip the FTS query entirely.
func sanitizeFTS5Query(query string) string {
	words := strings.Fields(query)
	terms := make([]string, 0, len(words))
	for _, w := range words {
		if !containsWordChar(w) {
			continue
		}
		// Escape embedded double-quotes by doubling them (FTS5 phrase-quoting rule).
		w = strings.ReplaceAll(w, `"`, `""`)
		terms = append(terms, `"`+w+`"*`)
	}
	return strings.Join(terms, " ")
}

// containsWordChar reports whether s contains at least one letter or digit.
// FTS5's unicode61 tokenizer indexes only letter and digit characters; words
// made entirely of other characters (e.g. "%", "\") produce empty phrases that
// cause FTS5 syntax errors.
func containsWordChar(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
