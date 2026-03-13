package db

import (
	"database/sql"
	"testing"
)

func TestCreateAndListArrServices(t *testing.T) {
	d := newTestDB(t)

	svc, err := d.CreateArrService("My Radarr", ArrServiceTypeRadarr, "http://radarr:7878", "apikey123")
	if err != nil {
		t.Fatalf("CreateArrService() error: %v", err)
	}
	if svc.ID == "" {
		t.Error("CreateArrService() returned empty ID")
	}
	if svc.Name != "My Radarr" {
		t.Errorf("Name = %q, want %q", svc.Name, "My Radarr")
	}
	if svc.Type != ArrServiceTypeRadarr {
		t.Errorf("Type = %q, want %q", svc.Type, ArrServiceTypeRadarr)
	}

	services, err := d.ListArrServices()
	if err != nil {
		t.Fatalf("ListArrServices() error: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service, got %d", len(services))
	}
	if services[0].ID != svc.ID {
		t.Errorf("ID mismatch: got %q, want %q", services[0].ID, svc.ID)
	}
}

func TestListArrServices_Empty(t *testing.T) {
	d := newTestDB(t)

	services, err := d.ListArrServices()
	if err != nil {
		t.Fatalf("ListArrServices() error: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}

func TestGetArrService(t *testing.T) {
	d := newTestDB(t)

	svc, _ := d.CreateArrService("Sonarr", ArrServiceTypeSonarr, "http://sonarr:8989", "key")

	found, err := d.GetArrService(svc.ID)
	if err != nil {
		t.Fatalf("GetArrService() error: %v", err)
	}
	if found.ID != svc.ID {
		t.Errorf("ID = %q, want %q", found.ID, svc.ID)
	}
}

func TestUpdateArrService(t *testing.T) {
	d := newTestDB(t)

	svc, _ := d.CreateArrService("Old Name", ArrServiceTypeRadarr, "http://old", "oldkey")

	updated, err := d.UpdateArrService(svc.ID, "New Name", ArrServiceTypeSonarr, "http://new", "newkey")
	if err != nil {
		t.Fatalf("UpdateArrService() error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Type != ArrServiceTypeSonarr {
		t.Errorf("Type = %q, want %q", updated.Type, ArrServiceTypeSonarr)
	}
	if updated.URL != "http://new" {
		t.Errorf("URL = %q, want %q", updated.URL, "http://new")
	}
}

func TestDeleteArrService(t *testing.T) {
	d := newTestDB(t)

	svc, _ := d.CreateArrService("Seerr", ArrServiceTypeSeerr, "http://seerr", "key")

	if err := d.DeleteArrService(svc.ID); err != nil {
		t.Fatalf("DeleteArrService() error: %v", err)
	}

	_, err := d.GetArrService(svc.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteArrService_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteArrService("nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
