package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", new("dt-123"), nil, nil)
	require.NoError(t, err, "CreateSeries() error")
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
			s, err := d.CreateSeries(t.Context(), tt.input, nil, nil, nil)
			require.NoError(t, err, "CreateSeries(%q) error", tt.input)
			if s.Name != tt.want {
				t.Errorf("CreateSeries(%q).Name = %q, want %q", tt.input, s.Name, tt.want)
			}
		})
	}
}

func TestCreateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "first CreateSeries() error")

	_, err = d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "Mistborn", nil, nil, nil)
	require.NoError(t, err, "first CreateSeries() error")

	_, err = d.CreateSeries(t.Context(), "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateSeries(t.Context(), name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			t.Errorf("CreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestUpdateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.UpdateSeries(t.Context(), s.ID, name, nil, nil, nil)
		if err != ErrInvalidSeriesName {
			t.Errorf("UpdateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestFindOrCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateSeries(t.Context(), name)
		if err != ErrInvalidSeriesName {
			t.Errorf("FindOrCreateSeries(%q) = %v, want ErrInvalidSeriesName", name, err)
		}
	}
}

func TestGetSeries(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	found, err := d.GetSeries(t.Context(), created.ID)
	require.NoError(t, err, "GetSeries() error")
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "The Dark Tower" {
		t.Errorf("Name = %q, want %q", found.Name, "The Dark Tower")
	}
}

func TestGetSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeries(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetSeriesByName(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	// All case variants should find the same series.
	for _, name := range []string{"The Dark Tower", "the dark tower", "THE DARK TOWER", "the Dark Tower"} {
		found, err := d.GetSeriesByName(t.Context(), name)
		require.NoError(t, err, "GetSeriesByName(%q) error", name)
		if found.ID != created.ID {
			t.Errorf("GetSeriesByName(%q) ID = %q, want %q", name, found.ID, created.ID)
		}
		if found.Name != "The Dark Tower" {
			t.Errorf("GetSeriesByName(%q) Name = %q, want %q", name, found.Name, "The Dark Tower")
		}
	}
}

func TestGetSeriesByName_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeriesByName(t.Context(), "Nonexistent Series")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeries(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateSeries(t.Context(), "Discworld", nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateSeries() for Discworld error")
	}
	if _, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateSeries() for The Dark Tower error")
	}

	list, err := d.ListSeries(t.Context())
	require.NoError(t, err, "ListSeries() error")
	if len(list) != 2 {
		require.Failf(t, "failed", "ListSeries() returned %d, want 2", len(list))
	}
	if list[0].Name != "Discworld" {
		t.Errorf("first series Name = %q, want %q", list[0].Name, "Discworld")
	}
}

func TestUpdateSeries(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	updated, err := d.UpdateSeries(t.Context(), created.ID, "The Dark Tower", new("dt-456"), nil, nil)
	require.NoError(t, err, "UpdateSeries() error")
	if updated.Name != "The Dark Tower" {
		t.Errorf("Name = %q, want %q", updated.Name, "The Dark Tower")
	}
	if updated.GoodreadsID == nil || *updated.GoodreadsID != "dt-456" {
		t.Errorf("GoodreadsID = %v, want %q", updated.GoodreadsID, "dt-456")
	}
}

func TestUpdateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	updated, err := d.UpdateSeries(t.Context(), created.ID, "  The   Dark   Tower  ", nil, nil, nil)
	require.NoError(t, err, "UpdateSeries() error")
	if updated.Name != "The Dark Tower" {
		t.Errorf("UpdateSeries().Name = %q, want %q", updated.Name, "The Dark Tower")
	}
}

func TestUpdateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateSeries() for The Dark Tower error")
	}
	s2, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for Dark Tower error")

	_, err = d.UpdateSeries(t.Context(), s2.ID, "The Dark Tower", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestUpdateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateSeries(t.Context(), "Mistborn", nil, nil, nil); err != nil {
		require.NoError(t, err, "CreateSeries() for Mistborn error")
	}
	s2, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for The Dark Tower error")

	_, err = d.UpdateSeries(t.Context(), s2.ID, "mistborn", nil, nil, nil)
	if err != ErrSeriesNameExists {
		t.Errorf("expected ErrSeriesNameExists, got %v", err)
	}
}

func TestFindOrCreateSeries_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(t.Context(), "Mistborn")
	require.NoError(t, err, "first FindOrCreateSeries() error")

	found, err := d.FindOrCreateSeries(t.Context(), "mistborn")
	require.NoError(t, err, "second FindOrCreateSeries() error")
	if found.ID != created.ID {
		t.Errorf("expected same series ID, got %q and %q", found.ID, created.ID)
	}
}

func TestFindOrCreateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(t.Context(), "The Wheel of Time")
	require.NoError(t, err, "first FindOrCreateSeries() error")

	found, err := d.FindOrCreateSeries(t.Context(), "  The   Wheel   of Time  ")
	require.NoError(t, err, "second FindOrCreateSeries() error")
	if found.ID != created.ID {
		t.Errorf("expected same series ID, got %q and %q", found.ID, created.ID)
	}
	if found.Name != "The Wheel of Time" {
		t.Errorf("FindOrCreateSeries().Name = %q, want %q", found.Name, "The Wheel of Time")
	}
}

func TestDeleteSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	err = d.DeleteSeries(t.Context(), s.ID)
	require.NoError(t, err, "DeleteSeries() error")

	_, err = d.GetSeries(t.Context(), s.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteSeries(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListSeriesPaginated(t *testing.T) {
	d := newTestDB(t)

	names := []string{"Discworld", "Dune", "Foundation", "The Dark Tower"}
	for _, name := range names {
		_, err := d.CreateSeries(t.Context(), name, nil, nil, nil)
		require.NoError(t, err, "CreateSeries(%q) error", name)
	}

	// First page: 2 of 4 series.
	page1, total, err := d.ListSeriesPaginated(t.Context(), 2, 0)
	require.NoError(t, err, "ListSeriesPaginated() error")
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(page1) != 2 {
		require.Failf(t, "failed", "len(page1) = %d, want 2", len(page1))
	}
	if page1[0].Name != "Discworld" {
		t.Errorf("page1[0].Name = %q, want %q", page1[0].Name, "Discworld")
	}
	if page1[1].Name != "Dune" {
		t.Errorf("page1[1].Name = %q, want %q", page1[1].Name, "Dune")
	}

	// Second page: remaining 2 series.
	page2, total2, err := d.ListSeriesPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListSeriesPaginated() page 2 error")
	if total2 != 4 {
		t.Errorf("page 2 total = %d, want 4", total2)
	}
	if len(page2) != 2 {
		require.Failf(t, "failed", "len(page2) = %d, want 2", len(page2))
	}
	if page2[0].Name != "Foundation" {
		t.Errorf("page2[0].Name = %q, want %q", page2[0].Name, "Foundation")
	}
	if page2[1].Name != "The Dark Tower" {
		t.Errorf("page2[1].Name = %q, want %q", page2[1].Name, "The Dark Tower")
	}

	// Empty table: total should be 0.
	d2 := newTestDB(t)
	empty, total3, err := d2.ListSeriesPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListSeriesPaginated() empty error")
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
