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
		t.Fatalf("CreateAuthor() error: %v", err)
	}
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

func TestCreateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("first CreateAuthor() error: %v", err)
	}

	_, err = d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		t.Errorf("expected ErrAuthorNameExists, got %v", err)
	}
}

func TestGetAuthor(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)

	found, err := d.GetAuthor(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetAuthor() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "Stephen King" {
		t.Errorf("Name = %q, want %q", found.Name, "Stephen King")
	}
}

func TestGetAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetAuthor(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListAuthors(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateAuthor(context.Background(), "Brandon Sanderson", nil, nil, nil, nil)
	_, _ = d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)

	authors, err := d.ListAuthors(context.Background())
	if err != nil {
		t.Fatalf("ListAuthors() error: %v", err)
	}
	if len(authors) != 2 {
		t.Fatalf("ListAuthors() returned %d, want 2", len(authors))
	}
	if authors[0].Name != "Brandon Sanderson" {
		t.Errorf("first author Name = %q, want %q", authors[0].Name, "Brandon Sanderson")
	}
}

func TestUpdateAuthor(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateAuthor(context.Background(), "S. King", nil, nil, nil, nil)

	updated, err := d.UpdateAuthor(context.Background(), created.ID, "Stephen King", strPtr("456"), nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateAuthor() error: %v", err)
	}
	if updated.Name != "Stephen King" {
		t.Errorf("Name = %q, want %q", updated.Name, "Stephen King")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "456" {
		t.Errorf("GoodreadsID = %v, want %q", updated.GoodreadsID, "456")
	}
}

func TestUpdateAuthor_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)
	a2, _ := d.CreateAuthor(context.Background(), "S. King", nil, nil, nil, nil)

	_, err := d.UpdateAuthor(context.Background(), a2.ID, "Stephen King", nil, nil, nil, nil)
	if err != ErrAuthorNameExists {
		t.Errorf("expected ErrAuthorNameExists, got %v", err)
	}
}

func TestDeleteAuthor(t *testing.T) {
	d := newTestDB(t)

	a, _ := d.CreateAuthor(context.Background(), "Stephen King", nil, nil, nil, nil)

	err := d.DeleteAuthor(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("DeleteAuthor() error: %v", err)
	}

	_, err = d.GetAuthor(context.Background(), a.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteAuthor_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteAuthor(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
