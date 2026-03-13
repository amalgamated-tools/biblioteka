package db

import (
	"database/sql"
	"testing"
)

func TestCreateSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries("The Dark Tower", strPtr("dt-123"), nil, nil)
	if err != nil {
		t.Fatalf("CreateSeries() error: %v", err)
	}
	if s.ID == "" {
		t.Error("CreateSeries() returned empty ID")
	}
	if s.Name != "The Dark Tower" {
		t.Errorf("Name = %q, want %q", s.Name, "The Dark Tower")
	}
	if s.GoodreadsID == nil || *s.GoodreadsID != "dt-123" {
		t.Errorf("GoodreadsID = %v, want %q", s.GoodreadsID, "dt-123")
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestCreateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries("The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("first CreateSeries() error: %v", err)
	}

	_, err = d.CreateSeries("The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestGetSeries(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateSeries("The Dark Tower", nil, nil, nil)

	found, err := d.GetSeries(created.ID)
	if err != nil {
		t.Fatalf("GetSeries() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "The Dark Tower" {
		t.Errorf("Name = %q, want %q", found.Name, "The Dark Tower")
	}
}

func TestGetSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeries("nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeries(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries("Discworld", nil, nil, nil)
	_, _ = d.CreateSeries("The Dark Tower", nil, nil, nil)

	list, err := d.ListSeries()
	if err != nil {
		t.Fatalf("ListSeries() error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListSeries() returned %d, want 2", len(list))
	}
	if list[0].Name != "Discworld" {
		t.Errorf("first series Name = %q, want %q", list[0].Name, "Discworld")
	}
}

func TestUpdateSeries(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateSeries("Dark Tower", nil, nil, nil)

	updated, err := d.UpdateSeries(created.ID, "The Dark Tower", strPtr("dt-456"), nil, nil)
	if err != nil {
		t.Fatalf("UpdateSeries() error: %v", err)
	}
	if updated.Name != "The Dark Tower" {
		t.Errorf("Name = %q, want %q", updated.Name, "The Dark Tower")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "dt-456" {
		t.Errorf("GoodreadsID = %v, want %q", updated.GoodreadsID, "dt-456")
	}
}

func TestUpdateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries("The Dark Tower", nil, nil, nil)
	s2, _ := d.CreateSeries("Dark Tower", nil, nil, nil)

	_, err := d.UpdateSeries(s2.ID, "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestDeleteSeries(t *testing.T) {
	d := newTestDB(t)

	s, _ := d.CreateSeries("The Dark Tower", nil, nil, nil)

	err := d.DeleteSeries(s.ID)
	if err != nil {
		t.Fatalf("DeleteSeries() error: %v", err)
	}

	_, err = d.GetSeries(s.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteSeries("nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
