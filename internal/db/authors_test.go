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
	if a.ID == "" {
		t.Error("CreateAuthor() returned empty ID")
	}
	if a.Name != "Stephen King" {
		t.Errorf("Name = %q, want %q", a.Name, "Stephen King")
	}
	if a.GoodreadsID == nil || *a.GoodreadsID != "123" {
		t.Errorf("GoodreadsID = %v, want %q", a.GoodreadsID, "123")
	}
	if a.HardcoverID != nil {
		t.Errorf("HardcoverID = %v, want nil", a.HardcoverID)
	}
	if a.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
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
		if a.Name != tt.want {
			t.Errorf("CreateAuthor(%q).Name = %q, want %q", tt.input, a.Name, tt.want)
		}
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
		if a.Name != tt.want {
			t.Errorf("CreateAuthor(%q).Name = %q, want %q", tt.input, a.Name, tt.want)
		}
	}
}

func TestCreateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "first CreateAuthor() error")

	_, err = d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		t.Errorf("expected ErrAuthorNameExists, got %v", err)
	}
}

func TestCreateAuthor_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Jane Austen", nil, nil, nil, nil)
	require.NoError(t, err, "first CreateAuthor() error")

	cases := []string{"jane austen", "JANE AUSTEN", "Jane AUSTEN", "jAnE aUsTeN"}
	for _, name := range cases {
		_, err = d.CreateAuthor(t.Context(), name, nil, nil, nil, nil)
		if err != ErrAuthorNameExists {
			t.Errorf("CreateAuthor(%q): expected ErrAuthorNameExists, got %v", name, err)
		}
	}
}

func TestGetAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	found, err := d.GetAuthor(t.Context(), created.ID)
	require.NoError(t, err, "GetAuthor() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "Stephen King" {
		t.Errorf("Name = %q, want %q", found.Name, "Stephen King")
	}
}

func TestGetAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthor(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetAuthorByName(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Anne McCaffrey", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	// All case variants should find the same author.
	for _, name := range []string{"Anne McCaffrey", "anne mccaffrey", "ANNE MCCAFFREY", "anne McCaffrey"} {
		found, err := d.GetAuthorByName(t.Context(), name)
		require.NoError(t, err, "GetAuthorByName(%q) error", name)
		if found.ID != created.ID {
			t.Errorf("GetAuthorByName(%q) ID = %q, want %q", name, found.ID, created.ID)
		}
		if found.Name != "Anne McCaffrey" {
			t.Errorf("GetAuthorByName(%q) Name = %q, want %q", name, found.Name, "Anne McCaffrey")
		}
	}
}

func TestGetAuthorByName_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthorByName(t.Context(), "Nonexistent Author")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListAuthors(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Brandon Sanderson", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")
	_, err = d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	authors, err := d.ListAuthors(t.Context())
	require.NoError(t, err, "ListAuthors() error")
	if len(authors) != 2 {
		require.Failf(t, "failed", "ListAuthors() returned %d, want 2", len(authors))
	}
	if authors[0].Name != "Brandon Sanderson" {
		t.Errorf("first author Name = %q, want %q", authors[0].Name, "Brandon Sanderson")
	}
}

func TestListAuthorsEmptyTable(t *testing.T) {
	d := newTestDB(t)

	authors, err := d.ListAuthors(t.Context())
	require.NoError(t, err, "ListAuthors() error")
	if len(authors) != 0 {
		t.Errorf("len(authors) = %d, want 0", len(authors))
	}
	if authors == nil {
		t.Error("authors = nil, want empty slice")
	}
}

func TestUpdateAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "S. King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	updated, err := d.UpdateAuthor(t.Context(), created.ID, "Stephen King", new("456"), nil, nil, nil)
	require.NoError(t, err, "UpdateAuthor() error")
	if updated.Name != "Stephen King" {
		t.Errorf("Name = %q, want %q", updated.Name, "Stephen King")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "456" {
		t.Errorf("GoodreadsID = %v, want %q", updated.GoodreadsID, "456")
	}
}

func TestUpdateAuthor_PreservesCapitalization(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(t.Context(), "Jane Austen", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	updated, err := d.UpdateAuthor(t.Context(), created.ID, "  Anne  McCaffrey  ", nil, nil, nil, nil)
	require.NoError(t, err, "UpdateAuthor() error")
	if updated.Name != "Anne McCaffrey" {
		t.Errorf("Name = %q, want %q", updated.Name, "Anne McCaffrey")
	}
}

func TestUpdateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")
	a2, err := d.CreateAuthor(t.Context(), "S. King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	_, err = d.UpdateAuthor(t.Context(), a2.ID, "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		t.Errorf("expected ErrAuthorNameExists, got %v", err)
	}
}

func TestDeleteAuthor(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateAuthor(t.Context(), "Stephen King", nil, nil, nil, nil)
	require.NoError(t, err, "CreateAuthor() error")

	err = d.DeleteAuthor(t.Context(), a.ID)
	require.NoError(t, err, "DeleteAuthor() error")

	_, err = d.GetAuthor(t.Context(), a.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteAuthor(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
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
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Name != "Brandon Sanderson" {
		t.Errorf("page1[0].Name = %q, want %q", page1[0].Name, "Brandon Sanderson")
	}
	if page1[1].Name != "Isaac Asimov" {
		t.Errorf("page1[1].Name = %q, want %q", page1[1].Name, "Isaac Asimov")
	}

	// Second page: remaining 2 authors.
	page2, total2, err := d.ListAuthorsPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListAuthorsPaginated() page 2 error")
	if total2 != 4 {
		t.Errorf("page 2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Name != "Stephen King" {
		t.Errorf("page2[0].Name = %q, want %q", page2[0].Name, "Stephen King")
	}
	if page2[1].Name != "Ursula K. Le Guin" {
		t.Errorf("page2[1].Name = %q, want %q", page2[1].Name, "Ursula K. Le Guin")
	}

	// Empty table: total should be 0.
	d2 := newTestDB(t)
	empty, total3, err := d2.ListAuthorsPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListAuthorsPaginated() empty error")
	if total3 != 0 {
		t.Errorf("empty total = %d, want 0", total3)
	}
	if len(empty) != 0 {
		t.Errorf("empty result len = %d, want 0", len(empty))
	}
	if empty == nil {
		t.Error("empty result = nil, want empty slice")
	}
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
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
	if items == nil {
		t.Error("items = nil, want empty slice")
	}

	items2, total2, err := d.ListAuthorsPaginated(t.Context(), -1, 0)
	require.NoError(t, err, "ListAuthorsPaginated(limit=-1) error")
	if total2 != 2 {
		t.Errorf("total = %d, want 2", total2)
	}
	if len(items2) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items2))
	}
	if items2 == nil {
		t.Error("items2 = nil, want empty slice")
	}
}

func TestFindOrCreateAuthor_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateAuthor(t.Context(), name)
		if err != ErrInvalidAuthorName {
			t.Errorf("FindOrCreateAuthor(%q) = %v, want ErrInvalidAuthorName", name, err)
		}
	}
}

func TestFindOrCreateAuthor_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateAuthor(t.Context(), "Stephen King")
	require.NoError(t, err, "first FindOrCreateAuthor() error")

	found, err := d.FindOrCreateAuthor(t.Context(), "stephen king")
	require.NoError(t, err, "second FindOrCreateAuthor() error")
	if found.ID != created.ID {
		t.Errorf("expected same author ID, got %q and %q", found.ID, created.ID)
	}
}

func TestFindOrCreateAuthor_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateAuthor(t.Context(), "Brandon Sanderson")
	require.NoError(t, err, "first FindOrCreateAuthor() error")

	found, err := d.FindOrCreateAuthor(t.Context(), "  Brandon   Sanderson  ")
	require.NoError(t, err, "second FindOrCreateAuthor() error")
	if found.ID != created.ID {
		t.Errorf("expected same author ID, got %q and %q", found.ID, created.ID)
	}
	if found.Name != "Brandon Sanderson" {
		t.Errorf("FindOrCreateAuthor().Name = %q, want %q", found.Name, "Brandon Sanderson")
	}
}
