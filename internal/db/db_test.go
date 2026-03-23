package db

import "testing"

func TestDialectOrderBy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		db        *DB
		column    string
		direction string
		want      string
	}{
		{
			name:      "sqlite plain column asc",
			db:        &DB{Dialect: DialectSQLite},
			column:    "title",
			direction: "ASC",
			want:      "ORDER BY title ASC, rowid ASC",
		},
		{
			name:      "postgres plain column desc",
			db:        &DB{Dialect: DialectPostgres},
			column:    "created_at",
			direction: "DESC",
			want:      "ORDER BY created_at DESC, id DESC",
		},
		{
			name:      "sqlite qualified column asc",
			db:        &DB{Dialect: DialectSQLite},
			column:    "b.title",
			direction: "ASC",
			want:      "ORDER BY b.title ASC, b.rowid ASC",
		},
		{
			name:      "postgres qualified column asc",
			db:        &DB{Dialect: DialectPostgres},
			column:    "bf.file_name",
			direction: "ASC",
			want:      "ORDER BY bf.file_name ASC, bf.id ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.db.dialectOrderBy(tt.column, tt.direction)
			if got != tt.want {
				t.Errorf("dialectOrderBy(%q, %q) = %q, want %q", tt.column, tt.direction, got, tt.want)
			}
		})
	}
}
