package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(context.Background(), "The Dark Tower", new("dt-123"), nil, nil)
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

func TestCreateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	tests := []struct {
		input string
		want  string
	}{
		{"  The Dark Tower  ", "The Dark Tower"},
		{"The   Wheel   of Time", "The Wheel of Time"},
		{"  A  Song   of Ice   and Fire  ", "A Song of Ice and Fire"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			s, err := d.CreateSeries(context.Background(), tt.input, nil, nil, nil)
			if err != nil {
				t.Fatalf("CreateSeries(%q) error: %v", tt.input, err)
			}
			if s.Name != tt.want {
				t.Errorf("CreateSeries(%q).Name = %q, want %q", tt.input, s.Name, tt.want)
			}
		})
	}
}

func TestCreateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("first CreateSeries() error: %v", err)
	}

	_, err = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(context.Background(), "Mistborn", nil, nil, nil)
	if err != nil {
		t.Fatalf("first CreateSeries() error: %v", err)
	}

	_, err = d.CreateSeries(context.Background(), "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateSeries(context.Background(), name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			t.Errorf("CreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestUpdateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.UpdateSeries(context.Background(), s.ID, name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			t.Errorf("UpdateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestFindOrCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateSeries(context.Background(), name)
		if err != ErrInvalidSeriesName {
			t.Errorf("FindOrCreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestGetSeries(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	found, err := d.GetSeries(context.Background(), created.ID)
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

	_, err := d.GetSeries(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeries(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "Discworld", nil, nil, nil)
	_, _ = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	list, err := d.ListSeries(context.Background())
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

	created, _ := d.CreateSeries(context.Background(), "Dark Tower", nil, nil, nil)

	updated, err := d.UpdateSeries(context.Background(), created.ID, "The Dark Tower", new("dt-456"), nil, nil)
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

func TestUpdateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(context.Background(), "Dark Tower", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateSeries() error: %v", err)
	}

	updated, err := d.UpdateSeries(context.Background(), created.ID, "  The   Dark   Tower  ", nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateSeries() error: %v", err)
	}
	if updated.Name != "The Dark Tower" {
		t.Errorf("UpdateSeries().Name = %q, want %q", updated.Name, "The Dark Tower")
	}
}

func TestUpdateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)
	s2, _ := d.CreateSeries(context.Background(), "Dark Tower", nil, nil, nil)

	_, err := d.UpdateSeries(context.Background(), s2.ID, "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestUpdateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateSeries(context.Background(), "Mistborn", nil, nil, nil)
	s2, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	_, err := d.UpdateSeries(context.Background(), s2.ID, "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestFindOrCreateSeries_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(context.Background(), "Mistborn")
	if err != nil {
		t.Fatalf("first FindOrCreateSeries() error: %v", err)
	}

	found, err := d.FindOrCreateSeries(context.Background(), "mistborn")
	if err != nil {
		t.Fatalf("second FindOrCreateSeries() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected same series ID, got %q and %q", found.ID, created.ID)
	}
}

func TestFindOrCreateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(context.Background(), "The Wheel of Time")
	if err != nil {
		t.Fatalf("first FindOrCreateSeries() error: %v", err)
	}

	found, err := d.FindOrCreateSeries(context.Background(), "  The   Wheel   of Time  ")
	if err != nil {
		t.Fatalf("second FindOrCreateSeries() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("expected same series ID, got %q and %q", found.ID, created.ID)
	}
	if found.Name != "The Wheel of Time" {
		t.Errorf("FindOrCreateSeries().Name = %q, want %q", found.Name, "The Wheel of Time")
	}
}

func TestDeleteSeries(t *testing.T) {
	d := newTestDB(t)

	s, _ := d.CreateSeries(context.Background(), "The Dark Tower", nil, nil, nil)

	err := d.DeleteSeries(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("DeleteSeries() error: %v", err)
	}

	_, err = d.GetSeries(context.Background(), s.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteSeries(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeriesPaginated(t *testing.T) {
	d := newTestDB(t)

	names := []string{"Discworld", "Dune", "Foundation", "The Dark Tower"}
	for _, name := range names {
		_, err := d.CreateSeries(context.Background(), name, nil, nil, nil)
		if err != nil {
			t.Fatalf("CreateSeries(%q) error: %v", name, err)
		}
	}

	// First page: 2 of 4 series.
	page1, total, err := d.ListSeriesPaginated(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("ListSeriesPaginated() error: %v", err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		t.Fatalf("len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Name != "Discworld" {
		t.Errorf("page1[0].Name = %q, want %q", page1[0].Name, "Discworld")
	}

	// Second page: remaining 2 series.
	page2, total2, err := d.ListSeriesPaginated(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("ListSeriesPaginated() page 2 error: %v", err)
	}
	if total2 != 4 {
		t.Errorf("page 2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		t.Fatalf("len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Name != "Foundation" {
		t.Errorf("page2[0].Name = %q, want %q", page2[0].Name, "Foundation")
	}

	// Empty table: total should be 0.
	d2 := newTestDB(t)
	empty, total3, err := d2.ListSeriesPaginated(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListSeriesPaginated() empty error: %v", err)
	}
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
