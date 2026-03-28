package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestSetAndGetSetting(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting(context.Background(), "theme", "dark"); err != nil {
		t.Fatalf("SetSetting() error: %v", err)
	}

	val, err := d.GetSetting(context.Background(), "theme")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "dark" {
		t.Errorf("GetSetting() = %q, want %q", val, "dark")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetSetting(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSetSetting_Upsert(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting(context.Background(), "color", "blue"); err != nil {
		t.Fatalf("SetSetting() for blue error: %v", err)
	}
	if err := d.SetSetting(context.Background(), "color", "red"); err != nil {
		t.Fatalf("SetSetting() for red error: %v", err)
	}

	val, err := d.GetSetting(context.Background(), "color")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if val != "red" {
		t.Errorf("GetSetting() = %q, want %q after upsert", val, "red")
	}
}
