package db

import (
	"context"
	"database/sql"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestCreateAuthor(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateAuthor(context.Background(), "Stephen King", strPtr("123"), nil, nil, strPtr("http://example.com/king.jpg"))
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}
	if a.ID == "" {
		fail(t, "CreateAuthor() returned empty ID")
	}
	if a.Name != "Stephen King" {
		failf(t, "Name = %q, want %q", a.Name, "Stephen King")
	}
	if a.GoodreadsID == nil || *a.GoodreadsID != "123" {
		failf(t, "GoodreadsID = %v, want %q", a.GoodreadsID, "123")
	}
	if a.HardcoverID != nil {
		failf(t, "HardcoverID = %v, want nil", a.HardcoverID)
	}
	if a.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
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
		a, err := d.CreateAuthor(context.Background(), tt.input, nil, nil, nil, nil)
		if err != nil {
			failNowf(t, "CreateAuthor(%q) error: %v", tt.input, err)
		}
		if a.Name != tt.want {
			failf(t, "CreateAuthor(%q).Name = %q, want %q", tt.input, a.Name, tt.want)
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
		a, err := d.CreateAuthor(context.Background(), tt.input, nil, nil, nil, nil)
		if err != nil {
			failNowf(t, "CreateAuthor(%q) error: %v", tt.input, err)
		}
		if a.Name != tt.want {
			failf(t, "CreateAuthor(%q).Name = %q, want %q", tt.input, a.Name, tt.want)
		}
	}
}

func TestCreateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "first CreateAuthor() error: %v", err)
	}

	_, err = d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		failf(t, "expected ErrAuthorNameExists, got %v", err)
	}
}

func TestCreateAuthor_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(context.Background(), "Jane Austen", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "first CreateAuthor() error: %v", err)
	}

	cases := []string{"jane austen", "JANE AUSTEN", "Jane AUSTEN", "jAnE aUsTeN"}
	for _, name := range cases {
		_, err = d.CreateAuthor(context.Background(), name, nil, nil, nil, nil)
		if err != ErrAuthorNameExists {
			failf(t, "CreateAuthor(%q): expected ErrAuthorNameExists, got %v", name, err)
		}
	}
}

func TestGetAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	found, err := d.GetAuthor(context.Background(), created.ID)
	if err != nil {
		failNowf(t, "GetAuthor() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "Stephen King" {
		failf(t, "Name = %q, want %q", found.Name, "Stephen King")
	}
}

func TestGetAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthor(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetAuthorByName(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(context.Background(), "Anne McCaffrey", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	// All case variants should find the same author.
	for _, name := range []string{"Anne McCaffrey", "anne mccaffrey", "ANNE MCCAFFREY", "anne McCaffrey"} {
		found, err := d.GetAuthorByName(context.Background(), name)
		if err != nil {
			failNowf(t, "GetAuthorByName(%q) error: %v", name, err)
		}
		if found.ID != created.ID {
			failf(t, "GetAuthorByName(%q) ID = %q, want %q", name, found.ID, created.ID)
		}
		if found.Name != "Anne McCaffrey" {
			failf(t, "GetAuthorByName(%q) Name = %q, want %q", name, found.Name, "Anne McCaffrey")
		}
	}
}

func TestGetAuthorByName_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthorByName(context.Background(), "Nonexistent Author")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestListAuthors(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(context.Background(), "Brandon Sanderson", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}
	_, err = d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	authors, err := d.ListAuthors(context.Background())
	if err != nil {
		failNowf(t, "ListAuthors() error: %v", err)
	}
	if len(authors) != 2 {
		failNowf(t, "ListAuthors() returned %d, want 2", len(authors))
	}
	if authors[0].Name != "Brandon Sanderson" {
		failf(t, "first author Name = %q, want %q", authors[0].Name, "Brandon Sanderson")
	}
}

func TestUpdateAuthor(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(context.Background(), "S. King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	updated, err := d.UpdateAuthor(context.Background(), created.ID, "Stephen King", strPtr("456"), nil, nil, nil)
	if err != nil {
		failNowf(t, "UpdateAuthor() error: %v", err)
	}
	if updated.Name != "Stephen King" {
		failf(t, "Name = %q, want %q", updated.Name, "Stephen King")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "456" {
		failf(t, "GoodreadsID = %v, want %q", updated.GoodreadsID, "456")
	}
}

func TestUpdateAuthor_PreservesCapitalization(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateAuthor(context.Background(), "Jane Austen", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	updated, err := d.UpdateAuthor(context.Background(), created.ID, "  Anne  McCaffrey  ", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "UpdateAuthor() error: %v", err)
	}
	if updated.Name != "Anne McCaffrey" {
		failf(t, "Name = %q, want %q", updated.Name, "Anne McCaffrey")
	}
}

func TestUpdateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}
	a2, err := d.CreateAuthor(context.Background(), "S. King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	_, err = d.UpdateAuthor(context.Background(), a2.ID, "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		failf(t, "expected ErrAuthorNameExists, got %v", err)
	}
}

func TestDeleteAuthor(t *testing.T) {
	d := newTestDB(t)

	a, err := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		failNowf(t, "CreateAuthor() error: %v", err)
	}

	err = d.DeleteAuthor(context.Background(), a.ID)
	if err != nil {
		failNowf(t, "DeleteAuthor() error: %v", err)
	}

	_, err = d.GetAuthor(context.Background(), a.ID)
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteAuthor(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}
