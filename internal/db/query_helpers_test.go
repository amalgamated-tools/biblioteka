package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsUniqueViolation(t *testing.T) {
	require.False(t, isUniqueViolation(nil))
	require.True(t, isUniqueViolation(errors.New("UNIQUE constraint failed: books.title")))
	require.True(t, isUniqueViolation(errors.New("duplicate key value violates unique constraint \"books_title_key\"")))
	require.False(t, isUniqueViolation(errors.New("some other error")))
}

func TestIsForeignKeyViolation(t *testing.T) {
	require.False(t, isForeignKeyViolation(nil))
	require.True(t, isForeignKeyViolation(errors.New("FOREIGN KEY constraint failed")))
	require.True(t, isForeignKeyViolation(errors.New("violates foreign key constraint \"books_author_id_fkey\"")))
	require.False(t, isForeignKeyViolation(errors.New("some other error")))
}

func TestIsColumnUniqueViolation(t *testing.T) {
	require.False(t, isColumnUniqueViolation(nil, "books.title", "idx_books_title"))
	require.False(t, isColumnUniqueViolation(errors.New("some other error"), "books.title", "idx_books_title"))

	// SQLite: matches table.column in message
	sqliteErr := errors.New("UNIQUE constraint failed: books.title")
	require.True(t, isColumnUniqueViolation(sqliteErr, "books.title", "idx_books_title"))
	require.False(t, isColumnUniqueViolation(sqliteErr, "books.isbn", "idx_books_isbn"))

	// PostgreSQL: matches index name in message
	pgErr := errors.New("duplicate key value violates unique constraint \"idx_books_title\"")
	require.True(t, isColumnUniqueViolation(pgErr, "books.title", "idx_books_title"))
	require.False(t, isColumnUniqueViolation(pgErr, "books.isbn", "idx_books_isbn"))
}

func TestBuildInClause_StartAtOne(t *testing.T) {
	inClause, args := buildInClause([]string{"book-1", "book-2", "book-3"}, 1)

	require.Equal(t, "$1,$2,$3", inClause)
	require.Equal(t, []any{"book-1", "book-2", "book-3"}, args)
}

func TestBuildInClause_WithOffset(t *testing.T) {
	inClause, args := buildInClause([]string{"book-1", "book-2"}, 2)

	require.Equal(t, "$2,$3", inClause)
	require.Equal(t, []any{"book-1", "book-2"}, args)
}

func TestBuildInClause_EmptyValues(t *testing.T) {
	inClause, args := buildInClause([]string{}, 1)

	require.Empty(t, inClause)
	require.Empty(t, args)
}
