package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Timestamp is a custom type that scans both string (SQLite) and time.Time
// (PostgreSQL) values and marshals to a consistent JSON string.
// Strings without timezone info are assumed to be UTC, which matches
// SQLite's datetime('now') and PostgreSQL's NOW() AT TIME ZONE 'UTC'.
type Timestamp struct {
	time.Time
}

// Layouts with explicit timezone info — parsed with time.Parse.
var tzLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999-07:00",
}

// Layouts without timezone info — parsed with time.ParseInLocation as UTC.
var utcLayouts = []string{
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

// Scan implements sql.Scanner, accepting string or time.Time.
func (t *Timestamp) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		for _, layout := range tzLayouts {
			if parsed, err := time.Parse(layout, v); err == nil {
				t.Time = parsed
				return nil
			}
		}
		for _, layout := range utcLayouts {
			if parsed, err := time.ParseInLocation(layout, v, time.UTC); err == nil {
				t.Time = parsed
				return nil
			}
		}
		return fmt.Errorf("Timestamp.Scan: unable to parse %q", v)
	case nil:
		t.Time = time.Time{}
		return nil
	default:
		return fmt.Errorf("Timestamp.Scan: unsupported type %T", value)
	}
}

// MarshalJSON outputs the timestamp as a JSON string in RFC3339 format.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + t.Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON parses a JSON string in RFC3339 format into a Timestamp.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == `""` || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	// Strip quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("Timestamp.UnmarshalJSON: %w", err)
	}
	t.Time = parsed
	return nil
}

// Dialect identifies which database backend is in use.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// DB wraps the sql.DB connection pool with dialect awareness.
type DB struct {
	*sql.DB
	Dialect Dialect
}

// execAffected executes a statement and returns the driver error from RowsAffected
// if one occurs, or sql.ErrNoRows when it affects no rows.
func (d *DB) execAffected(ctx context.Context, query string, args ...any) error {
	res, err := d.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, raErr := res.RowsAffected()
	if raErr != nil {
		return raErr
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// now returns the SQL expression for the current timestamp in the active dialect.
func (d *DB) now() string {
	if d.Dialect == DialectPostgres {
		return "NOW()"
	}
	return "datetime('now')"
}

// dialectOrderBy returns an ORDER BY clause with a dialect-appropriate
// row identifier tiebreaker. SQLite uses rowid while PostgreSQL uses id.
// column must be a hardcoded SQL identifier — never pass user-supplied input.
func (d *DB) dialectOrderBy(column, direction string) string {
	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC"
	}
	tiebreaker := "rowid"
	if d.Dialect == DialectPostgres {
		tiebreaker = "id"
	}
	if idx := strings.LastIndexByte(column, '.'); idx >= 0 {
		tiebreaker = column[:idx+1] + tiebreaker
	}
	return fmt.Sprintf("ORDER BY %s %s, %s %s", column, direction, tiebreaker, direction)
}
