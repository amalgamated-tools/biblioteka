package db

import (
	"reflect"
	"strings"
	"testing"
)

func TestRemoveInlineComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple inline comment",
			input:    "SELECT * FROM users; -- get all users",
			expected: "SELECT * FROM users; ",
		},
		{
			name:     "inline comment with semicolon",
			input:    "CREATE TABLE foo; -- comment with ; here",
			expected: "CREATE TABLE foo; ",
		},
		{
			name:     "inline comment with newline",
			input:    "SELECT * FROM users; -- get all users\n",
			expected: "SELECT * FROM users; \n",
		},
		{
			name:     "multiple statements with comments",
			input:    "SELECT * FROM a; -- comment\nSELECT * FROM b; -- another comment",
			expected: "SELECT * FROM a; \nSELECT * FROM b; ",
		},
		{
			name:     "comment with string containing --",
			input:    "INSERT INTO x VALUES ('value -- not a comment'); -- real comment",
			expected: "INSERT INTO x VALUES ('value -- not a comment'); ",
		},
		{
			name:     "no comments",
			input:    "SELECT * FROM users;",
			expected: "SELECT * FROM users;",
		},
		{
			name:     "double dash in string",
			input:    "SELECT '-- this is not a comment' FROM users;",
			expected: "SELECT '-- this is not a comment' FROM users;",
		},
		{
			name:     "escaped quotes with comment",
			input:    "SELECT 'it''s a test' FROM x; -- comment here",
			expected: "SELECT 'it''s a test' FROM x; ",
		},
		{
			name:     "double quotes with comment",
			input:    `SELECT "column -- name" FROM t; -- comment`,
			expected: `SELECT "column -- name" FROM t; `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeInlineComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeInlineComments() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "simple statement",
			input: "SELECT * FROM users",
			expected: []string{
				"SELECT * FROM users",
			},
		},
		{
			name:  "multiple statements",
			input: "SELECT * FROM users; SELECT * FROM posts;",
			expected: []string{
				"SELECT * FROM users",
				" SELECT * FROM posts",
			},
		},
		{
			name:  "statement with inline comment",
			input: "CREATE TABLE foo; -- comment with ; here",
			expected: []string{
				"CREATE TABLE foo",
				" ",
			},
		},
		{
			name:  "semicolon in string",
			input: "INSERT INTO x VALUES ('a;b'); INSERT INTO y VALUES ('c');",
			expected: []string{
				"INSERT INTO x VALUES ('a;b')",
				" INSERT INTO y VALUES ('c')",
			},
		},
		{
			name:  "inline comment with semicolon and newline",
			input: "SELECT * FROM a; -- comment; with semicolon\nSELECT * FROM b;",
			expected: []string{
				"SELECT * FROM a",
				" \nSELECT * FROM b",
			},
		},
		{
			name:  "comment preserved in string",
			input: "INSERT INTO x VALUES ('-- not a comment; really'); SELECT 1;",
			expected: []string{
				"INSERT INTO x VALUES ('-- not a comment; really')",
				" SELECT 1",
			},
		},
		{
			name:  "escaped quotes",
			input: "INSERT INTO x VALUES ('it''s ok'); SELECT 1;",
			expected: []string{
				"INSERT INTO x VALUES ('it''s ok')",
				" SELECT 1",
			},
		},
		{
			name: "sqlite trigger statement",
			input: `CREATE TRIGGER IF NOT EXISTS goqite_updated_timestamp AFTER UPDATE ON goqite BEGIN
	UPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = old.id;
END;

CREATE INDEX IF NOT EXISTS goqite_queue_priority_created_idx ON goqite (queue, priority DESC, created);`,
			expected: []string{
				"CREATE TRIGGER IF NOT EXISTS goqite_updated_timestamp AFTER UPDATE ON goqite BEGIN\n\tUPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = old.id;\nEND",
				"\n\nCREATE INDEX IF NOT EXISTS goqite_queue_priority_created_idx ON goqite (queue, priority DESC, created)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitStatements(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitStatements() = %v, want %v", result, tt.expected)
				for i := range result {
					if i < len(tt.expected) {
						if result[i] != tt.expected[i] {
							t.Errorf("  [%d] got %q, want %q", i, result[i], tt.expected[i])
						}
					} else {
						t.Errorf("  [%d] got %q (extra)", i, result[i])
					}
				}
			}
		})
	}
}

func TestExtractUpSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic migration",
			input: `-- migrate:up
CREATE TABLE users (
    id INTEGER PRIMARY KEY
);

-- migrate:down
DROP TABLE users;`,
			expected: "CREATE TABLE users (\n    id INTEGER PRIMARY KEY\n);",
		},
		{
			name: "migration with comments",
			input: `-- This is a comment
-- migrate:up
-- Another comment
CREATE TABLE users (
    id INTEGER PRIMARY KEY
);

-- migrate:down
DROP TABLE users;`,
			expected: "CREATE TABLE users (\n    id INTEGER PRIMARY KEY\n);",
		},
		{
			name:     "no up section",
			input:    "-- migrate:down\nDROP TABLE users;",
			expected: "",
		},
		{
			name: "multiple statements",
			input: `-- migrate:up
CREATE TABLE users (id INTEGER);
CREATE TABLE posts (id INTEGER);

-- migrate:down
DROP TABLE posts;
DROP TABLE users;`,
			expected: "CREATE TABLE users (id INTEGER);\nCREATE TABLE posts (id INTEGER);",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUpSQL(tt.input)
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)
			if result != expected {
				t.Errorf("extractUpSQL() = %q, want %q", result, expected)
			}
		})
	}
}
