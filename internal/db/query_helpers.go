package db

import "strings"

// buildInClause builds positional SQL placeholders and args for an IN clause.
// startAt is the 1-based index of the first placeholder.
func buildInClause[T any](values []T, startAt int) (string, []any) {
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = dollarN(startAt + i)
		args[i] = v
	}
	return strings.Join(placeholders, ","), args
}
