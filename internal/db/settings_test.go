package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAndGetSetting(t *testing.T) {
	d := newTestDB(t)

	require.NoError(t, d.SetSetting(t.Context(), "theme", "dark"), "SetSetting() error")

	val, err := d.GetSetting(t.Context(), "theme")
	require.NoError(t, err, "GetSetting() error")
	require.Equal(t, "dark", val)
}

func TestGetSetting_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSetting(t.Context(), "nonexistent")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestSetSetting_Upsert(t *testing.T) {
	d := newTestDB(t)

	require.NoError(t, d.SetSetting(t.Context(), "color", "blue"), "SetSetting() for blue error")
	require.NoError(t, d.SetSetting(t.Context(), "color", "red"), "SetSetting() for red error")

	val, err := d.GetSetting(t.Context(), "color")
	require.NoError(t, err, "GetSetting() error")
	require.Equal(t, "red", val)
}
