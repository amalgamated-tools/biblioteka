package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(context.Background(), "The Dark Tower", strPtr("dt-123"), nil, nil)
	if err != nil {
		failNowf(t, "CreateSeries() error: %v", err)
	}
	if s.ID == "" {
		fail(t, "CreateSeries() returned empty ID")
	}
	if s.Name != "The Dark Tower" {
		failf(t, "Name = %q, want %q", s.Name, "The Dark Tower")
	}
	if s.GoodreadsID == nil || *s.GoodreadsID != "dt-123" {
		failf(t, "GoodreadsID = %v, want %q", s.GoodreadsID, "dt-123")
	}
	if s.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
	}
}

func TestCreateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		failNowf(t, "first CreateSeries() error: %v", err)
	}

	_, err = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		failf(t, "expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(context.Background(), "Mistborn", nil, nil, nil)
	if err != nil {
		failNowf(t, "first CreateSeries() error: %v", err)
	}

	_, err = d.CreateSeries(context.Background(), "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		failf(t, "expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateSeries(context.Background(), name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			failf(t, "CreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestUpdateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.UpdateSeries(context.Background(), s.ID, name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			failf(t, "UpdateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestFindOrCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateSeries(context.Background(), name)
		if err != ErrInvalidSeriesName {
			failf(t, "FindOrCreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestGetSeries(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	found, err := d.GetSeries(context.Background(), created.ID)
	if err != nil {
		failNowf(t, "GetSeries() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "The Dark Tower" {
		failf(t, "Name = %q, want %q", found.Name, "The Dark Tower")
	}
}

func TestGetSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeries(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeries(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "Discworld", nil, nil, nil)
	_, _ = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	list, err := d.ListSeries(context.Background())
	if err != nil {
		failNowf(t, "ListSeries() error: %v", err)
	}
	if len(list) != 2 {
		failNowf(t, "ListSeries() returned %d, want 2", len(list))
	}
	if list[0].Name != "Discworld" {
		failf(t, "first series Name = %q, want %q", list[0].Name, "Discworld")
	}
}

func TestUpdateSeries(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateSeries(context.Background(), "Dark Tower", nil, nil, nil)

	updated, err := d.UpdateSeries(context.Background(), created.ID, "The Dark Tower", strPtr("dt-456"), nil, nil)
	if err != nil {
		failNowf(t, "UpdateSeries() error: %v", err)
	}
	if updated.Name != "The Dark Tower" {
		failf(t, "Name = %q, want %q", updated.Name, "The Dark Tower")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "dt-456" {
		failf(t, "GoodreadsID = %v, want %q", updated.GoodreadsID, "dt-456")
	}
}

func TestUpdateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	s2, _ := d.CreateSeries(context.Background(), "Dark Tower", nil, nil, nil)

	_, err := d.UpdateSeries(context.Background(), s2.ID, "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		failf(t, "expected ErrSeriesNameExists, got %v", err)
	}
}

func TestUpdateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "Mistborn", nil, nil, nil)
	s2, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	_, err := d.UpdateSeries(context.Background(), s2.ID, "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		failf(t, "expected ErrSeriesNameExists, got %v", err)
	}
}

func TestFindOrCreateSeries_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(context.Background(), "Mistborn")
	if err != nil {
		failNowf(t, "first FindOrCreateSeries() error: %v", err)
	}

	found, err := d.FindOrCreateSeries(context.Background(), "mistborn")
	if err != nil {
		failNowf(t, "second FindOrCreateSeries() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "expected same series ID, got %q and %q", found.ID, created.ID)
	}
}

func TestDeleteSeries(t *testing.T) {
	d := newTestDB(t)

	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	err := d.DeleteSeries(context.Background(), s.ID)
	if err != nil {
		failNowf(t, "DeleteSeries() error: %v", err)
	}

	_, err = d.GetSeries(context.Background(), s.ID)
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteSeries(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}
