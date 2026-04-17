package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
