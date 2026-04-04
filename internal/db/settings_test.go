package db

import (
	"database/sql"
	"testing"
)

func TestSetAndGetSetting(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting(t.Context(), "theme", "dark"); err != nil {
		t.Fatalf("SetSetting() error: %v", err)
	}

	val, err := d.GetSetting(t.Context(), "theme")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "dark" {
		t.Errorf("GetSetting() = %q, want %q", val, "dark")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSetting(t.Context(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSetSetting_Upsert(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting(t.Context(), "color", "blue"); err != nil {
		t.Fatalf("SetSetting() for blue error: %v", err)
	}
	if err := d.SetSetting(t.Context(), "color", "red"); err != nil {
		t.Fatalf("SetSetting() for red error: %v", err)
	}

	val, err := d.GetSetting(t.Context(), "color")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "red" {
		t.Errorf("GetSetting() = %q, want %q after upsert", val, "red")
	}
}
