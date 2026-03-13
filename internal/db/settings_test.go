package db

import (
	"database/sql"
	"testing"
)

func TestSetAndGetSetting(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("SetSetting() error: %v", err)
	}

	val, err := d.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "dark" {
		t.Errorf("GetSetting() = %q, want %q", val, "dark")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSetting("nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSetSetting_Upsert(t *testing.T) {
	d := newTestDB(t)

	_ = d.SetSetting("color", "blue")
	_ = d.SetSetting("color", "red")

	val, err := d.GetSetting("color")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "red" {
		t.Errorf("GetSetting() = %q, want %q after upsert", val, "red")
	}
}
