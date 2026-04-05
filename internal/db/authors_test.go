package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAuthor(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateAuthor(t.Context(), "Stephen King", new("123"), nil, nil, new("http://example.com/king.jpg"))
	require.NoError(t, err, "CreateAuthor() error")
	require.NotEqual(t, "", a.ID)
	require.Equal(t, "Stephen King", a.Name)
	require.NotNil(t, a.GoodreadsID)
	require.Equal(t, "123", *a.GoodreadsID)
	require.Nil(t, a.HardcoverID)
	require.False(t, a.CreatedAt.IsZero())
}

func TestCreateAuthor_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	tests := []struct {
		input string
		want  string
	}{
		{"  Jane Austen  ", "Jane Austen"},
		{"Stephen   King", "Stephen King"},
		{"  Anne  McCaffrey  ", "Anne McCaffrey"},
	}

	for _, tt := range tests {
		a, err := d.CreateAuthor(t.Context(), tt.input, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q) error", tt.input)
		require.Equal(t, tt.want, a.Name)
	}
}

func TestCreateAuthor_PreservesCapitalization(t *testing.T) {
	d := newTestDB(t)

	tests := []struct {
		input string
		want  string
	}{
		{"Anne McCaffrey", "Anne McCaffrey"},
		{"Melissa de la Cruz", "Melissa de la Cruz"},
		{"J.R.R. Tolkien", "J.R.R. Tolkien"},
	}

	for _, tt := range tests {
		a, err := d.CreateAuthor(t.Context(), tt.input, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q) error", tt.input)
		require.Equal(t, tt.want, a.Name)
	}
}

func TestCreateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "first CreateAuthor() error")

	_, err = d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.Equal(t, ErrAuthorNameExists, err)
}

func TestCreateAuthor_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Jane Austen", nil, nil, nil, nil)
	require.NoError(t, err, "first CreateAuthor() error")

	cases := []string{"jane austen", "JANE AUSTEN", "Jane AUSTEN", "jAnE aUsTeN"}
	for _, name := range cases {
		_, err = d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.Equal(t, ErrAuthorNameExists, err)
	}
}

func TestGetAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	found, err := d.GetAuthor(t.Context(), created.ID)
	require.NoError(t, err, "GetAuthor() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "Stephen King", found.Name)
}

func TestGetAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthor(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetAuthorByName(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Anne McCaffrey", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	// All case variants should find the same author.
	for _, name := range []string{"Anne McCaffrey", "anne mccaffrey", "ANNE MCCAFFREY", "anne McCaffrey"} {
		found, err := d.GetAuthorByName(t.Context(), name)
		require.NoError(t, err, "GetAuthorByName(%q) error", name)
		require.Equal(t, created.ID, found.ID)
		require.Equal(t, "Anne McCaffrey", found.Name)
	}
}

func TestGetAuthorByName_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthorByName(t.Context(), "Nonexistent Author")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListAuthors(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Brandon Sanderson", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")
	_, err = d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	authors, err := d.ListAuthors(t.Context())
	require.NoError(t, err, "ListAuthors() error")
	require.Len(t, authors, 2)
	require.Equal(t, "Brandon Sanderson", authors[0].Name)
}

func TestListAuthorsEmptyTable(t *testing.T) {
	d := newTestDB(t)

	authors, err := d.ListAuthors(t.Context())
	require.NoError(t, err, "ListAuthors() error")
	require.Len(t, authors, 0)
	require.NotNil(t, authors)
}

func TestUpdateAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "S. King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	updated, err := d.UpdateAuthor(t.Context(), created.ID, "Stephen King", new("456"), nil, nil, nil)
	require.NoError(t, err, "UpdateAuthor() error")
	require.Equal(t, "Stephen King", updated.Name)
	require.NotNil(t, updated.GoodreadsID)
	require.Equal(t, "456", *updated.GoodreadsID)
}

func TestUpdateAuthor_PreservesCapitalization(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Jane Austen", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	updated, err := d.UpdateAuthor(t.Context(), created.ID, "  Anne  McCaffrey  ", nil, nil, nil, nil)
	require.NoError(t, err, "UpdateAuthor() error")
	require.Equal(t, "Anne McCaffrey", updated.Name)
}

func TestUpdateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")
	a2, err := d.CreateAuthor(t.Context(), "S. King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	_, err = d.UpdateAuthor(t.Context(), a2.ID, "Stephen King", nil, nil, nil, nil)
	require.Equal(t, ErrAuthorNameExists, err)
}

func TestDeleteAuthor(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	err = d.DeleteAuthor(t.Context(), a.ID)
	require.NoError(t, err, "DeleteAuthor() error")

	_, err = d.GetAuthor(t.Context(), a.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteAuthor(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListAuthorsPaginated(t *testing.T) {
	d := newTestDB(t)

	names := []string{"Brandon Sanderson", "Isaac Asimov", "Stephen King", "Ursula K. Le Guin"}
	for _, name := range names {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q) error", name)
	}

	// First page: 2 of 4 authors.
	page1, total, err := d.ListAuthorsPaginated(t.Context(), 2, 0)
	require.NoError(t, err, "ListAuthorsPaginated() error")
	require.Equal(t, 4, total)
	require.Len(t, page1, 2)
	require.Equal(t, "Brandon Sanderson", page1[0].Name)
	require.Equal(t, "Isaac Asimov", page1[1].Name)

	// Second page: remaining 2 authors.
	page2, total2, err := d.ListAuthorsPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListAuthorsPaginated() page 2 error")
	require.Equal(t, 4, total2)
	require.Len(t, page2, 2)
	require.Equal(t, "Stephen King", page2[0].Name)
	require.Equal(t, "Ursula K. Le Guin", page2[1].Name)

	// Empty table: total should be 0.
	d2 := newTestDB(t)
	empty, total3, err := d2.ListAuthorsPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListAuthorsPaginated() empty error")
	require.Equal(t, 0, total3)
	require.Len(t, empty, 0)
	require.NotNil(t, empty)
}

func TestListAuthorsPaginatedZeroLimit(t *testing.T) {
	d := newTestDB(t)

	names := []string{"Brandon Sanderson", "Isaac Asimov"}
	for _, name := range names {
		_, err := d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		require.NoError(t, err, "CreateAuthor(%q) error", name)
	}

	// limit=0 should return the real total with an empty items slice.
	items, total, err := d.ListAuthorsPaginated(t.Context(), 0, 0)
	require.NoError(t, err, "ListAuthorsPaginated(limit=0) error")
	require.Equal(t, 2, total)
	require.Len(t, items, 0)
	require.NotNil(t, items)

	items2, total2, err := d.ListAuthorsPaginated(t.Context(), -1, 0)
	require.NoError(t, err, "ListAuthorsPaginated(limit=-1) error")
	require.Equal(t, 2, total2)
	require.Len(t, items2, 0)
	require.NotNil(t, items2)
}

func TestFindOrCreateAuthor_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateAuthor(t.Context(), name)
		require.Equal(t, ErrInvalidAuthorName, err)
	}
}

func TestFindOrCreateAuthor_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateAuthor(t.Context(), "Stephen King")
	require.NoError(t, err, "first FindOrCreateAuthor() error")

	found, err := d.FindOrCreateAuthor(t.Context(), "stephen king")
	require.NoError(t, err, "second FindOrCreateAuthor() error")
	require.Equal(t, created.ID, found.ID)
}

func TestFindOrCreateAuthor_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateAuthor(t.Context(), "Brandon Sanderson")
	require.NoError(t, err, "first FindOrCreateAuthor() error")

	found, err := d.FindOrCreateAuthor(t.Context(), "  Brandon   Sanderson  ")
	require.NoError(t, err, "second FindOrCreateAuthor() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "Brandon Sanderson", found.Name)
}
