package db

import (
	"strconv"
	"strings"
)

// dollarN returns a PostgreSQL-style positional placeholder ($1, $2, ...).
// SQLite also accepts dollar-sign placeholders.
func dollarN(n int) string {
	return "$" + strconv.Itoa(n)
}

// buildInClause builds positional SQL placeholders and args for an IN clause.
// startAt is the 1-based index of the first placeholder.
// Callers must ensure values is non-empty; an empty slice returns ("", nil),
// which would produce invalid SQL if embedded in an IN clause.
func buildInClause[T any](values []T, startAt int) (string, []any) {
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = dollarN(startAt + i)
		args[i] = v
	}
	return strings.Join(placeholders, ","), args
}
