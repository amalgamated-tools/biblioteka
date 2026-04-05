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
	require.NotEqual(t, "", s.ID)
	require.Equal(t, "The Dark Tower", s.Name)
	require.NotNil(t, s.GoodreadsID)
	require.Equal(t, "dt-123", *s.GoodreadsID)
	require.False(t, s.CreatedAt.IsZero())
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
			require.Equal(t, tt.want, s.Name)
		})
	}
}

func TestCreateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "first CreateSeries() error")

	_, err = d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.Equal(t, ErrSeriesNameExists, err)
}

func TestCreateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "Mistborn", nil, nil, nil)
	require.NoError(t, err, "first CreateSeries() error")

	_, err = d.CreateSeries(t.Context(), "mistborn", nil, nil, nil)
	require.Equal(t, ErrSeriesNameExists, err)
}

func TestCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateSeries(t.Context(), name, nil, nil, nil)
		require.Equal(t, ErrInvalidSeriesName, err)
	}
}

func TestUpdateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	for _, name := range []string{"", " ", "  \t  "} {
		_, err = d.UpdateSeries(t.Context(), s.ID, name, nil, nil, nil)
		require.Equal(t, ErrInvalidSeriesName, err)
	}
}

func TestFindOrCreateSeries_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.FindOrCreateSeries(t.Context(), name)
		require.Equal(t, ErrInvalidSeriesName, err)
	}
}

func TestGetSeries(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	found, err := d.GetSeries(t.Context(), created.ID)
	require.NoError(t, err, "GetSeries() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "The Dark Tower", found.Name)
}

func TestGetSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeries(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetSeriesByName(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	// All case variants should find the same series.
	for _, name := range []string{"The Dark Tower", "the dark tower", "THE DARK TOWER", "the Dark Tower"} {
		found, err := d.GetSeriesByName(t.Context(), name)
		require.NoError(t, err, "GetSeriesByName(%q) error", name)
		require.Equal(t, created.ID, found.ID)
		require.Equal(t, "The Dark Tower", found.Name)
	}
}

func TestGetSeriesByName_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSeriesByName(t.Context(), "Nonexistent Series")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListSeries(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "Discworld", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for Discworld error")
	_, err = d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for The Dark Tower error")

	list, err := d.ListSeries(t.Context())
	require.NoError(t, err, "ListSeries() error")
	require.Len(t, list, 2)
	require.Equal(t, "Discworld", list[0].Name)
}

func TestUpdateSeries(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	updated, err := d.UpdateSeries(t.Context(), created.ID, "The Dark Tower", new("dt-456"), nil, nil)
	require.NoError(t, err, "UpdateSeries() error")
	require.Equal(t, "The Dark Tower", updated.Name)
	require.NotNil(t, updated.GoodreadsID)
	require.Equal(t, "dt-456", *updated.GoodreadsID)
}

func TestUpdateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	updated, err := d.UpdateSeries(t.Context(), created.ID, "  The   Dark   Tower  ", nil, nil, nil)
	require.NoError(t, err, "UpdateSeries() error")
	require.Equal(t, "The Dark Tower", updated.Name)
}

func TestUpdateSeries_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for The Dark Tower error")
	s2, err := d.CreateSeries(t.Context(), "Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for Dark Tower error")

	_, err = d.UpdateSeries(t.Context(), s2.ID, "The Dark Tower", nil, nil, nil)
	require.Equal(t, ErrSeriesNameExists, err)
}

func TestUpdateSeries_DuplicateNameCaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateSeries(t.Context(), "Mistborn", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for Mistborn error")
	s2, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() for The Dark Tower error")

	_, err = d.UpdateSeries(t.Context(), s2.ID, "mistborn", nil, nil, nil)
	require.Equal(t, ErrSeriesNameExists, err)
}

func TestFindOrCreateSeries_CaseInsensitive(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(t.Context(), "Mistborn")
	require.NoError(t, err, "first FindOrCreateSeries() error")

	found, err := d.FindOrCreateSeries(t.Context(), "mistborn")
	require.NoError(t, err, "second FindOrCreateSeries() error")
	require.Equal(t, created.ID, found.ID)
}

func TestFindOrCreateSeries_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	created, err := d.FindOrCreateSeries(t.Context(), "The Wheel of Time")
	require.NoError(t, err, "first FindOrCreateSeries() error")

	found, err := d.FindOrCreateSeries(t.Context(), "  The   Wheel   of Time  ")
	require.NoError(t, err, "second FindOrCreateSeries() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "The Wheel of Time", found.Name)
}

func TestDeleteSeries(t *testing.T) {
	d := newTestDB(t)

	s, err := d.CreateSeries(t.Context(), "The Dark Tower", nil, nil, nil)
	require.NoError(t, err, "CreateSeries() error")

	err = d.DeleteSeries(t.Context(), s.ID)
	require.NoError(t, err, "DeleteSeries() error")

	_, err = d.GetSeries(t.Context(), s.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteSeries(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
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
	require.Equal(t, 4, total)
	require.Len(t, page1, 2)
	require.Equal(t, "Discworld", page1[0].Name)
	require.Equal(t, "Dune", page1[1].Name)

	// Second page: remaining 2 series.
	page2, total2, err := d.ListSeriesPaginated(t.Context(), 2, 2)
	require.NoError(t, err, "ListSeriesPaginated() page 2 error")
	require.Equal(t, 4, total2)
	require.Len(t, page2, 2)
	require.Equal(t, "Foundation", page2[0].Name)
	require.Equal(t, "The Dark Tower", page2[1].Name)

	// Empty table: total should be 0.
	d2 := newTestDB(t)
	empty, total3, err := d2.ListSeriesPaginated(t.Context(), 10, 0)
	require.NoError(t, err, "ListSeriesPaginated() empty error")
	require.Equal(t, 0, total3)
	require.Len(t, empty, 0)
	require.NotNil(t, empty)
}
