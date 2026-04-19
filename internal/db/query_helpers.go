package db

import (
	"strconv"
	"strings"
)

// isUniqueViolation checks if an error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite: "UNIQUE constraint failed: ..."
	// PostgreSQL: "duplicate key value violates unique constraint ..."
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}

// isForeignKeyViolation reports whether err is a foreign-key constraint violation.
func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// SQLite: "FOREIGN KEY constraint failed"
	// PostgreSQL: "violates foreign key constraint ..."
	return strings.Contains(msg, "FOREIGN KEY constraint failed") ||
		strings.Contains(msg, "violates foreign key constraint")
}

// isColumnUniqueViolation reports whether err is a unique constraint violation
// on the specified table column or named unique index (as reported in the error message).
func isColumnUniqueViolation(err error, tableCol, idxName string) bool {
	if err == nil || !isUniqueViolation(err) {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, tableCol) || strings.Contains(msg, idxName)
}

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
