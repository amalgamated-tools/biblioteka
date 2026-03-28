package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}
	if lib.ID == "" {
		t.Error("CreateLibrary() returned empty ID")
	}
	if lib.Name != "Fiction" {
		t.Errorf("Name = %q, want %q", lib.Name, "Fiction")
	}
	if lib.Paths != `["/mnt/books/fiction"]` {
		t.Errorf("Paths = %q, want %q", lib.Paths, `["/mnt/books/fiction"]`)
	}
	if lib.OrganizationType != LibraryOrganizationBookPerFolder {
		t.Errorf("OrganizationType = %q, want %q", lib.OrganizationType, LibraryOrganizationBookPerFolder)
	}
	if lib.Monitored {
		t.Error("Monitored should be false")
	}
	if lib.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if lib.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestCreateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != nil {
		t.Fatalf("first CreateLibrary() error: %v", err)
	}

	_, err = d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/other"]`, LibraryOrganizationBookPerFolder, false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestGetLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, true)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}

	found, err := d.GetLibrary(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetLibrary() error: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "Fiction" {
		t.Errorf("Name = %q, want %q", found.Name, "Fiction")
	}
	if !found.Monitored {
		t.Error("Monitored should be true")
	}
}

func TestGetLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetLibrary(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListLibraries(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false); err != nil {
		t.Fatalf("CreateLibrary() for Fiction error: %v", err)
	}
	if _, err := d.CreateLibrary(context.Background(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, true); err != nil {
		t.Fatalf("CreateLibrary() for Non-Fiction error: %v", err)
	}

	libs, err := d.ListLibraries(context.Background())
	if err != nil {
		t.Fatalf("ListLibraries() error: %v", err)
	}
	if len(libs) != 2 {
		t.Fatalf("ListLibraries() returned %d, want 2", len(libs))
	}
	if libs[0].Name != "Fiction" {
		t.Errorf("first library Name = %q, want %q", libs[0].Name, "Fiction")
	}
}

func TestUpdateLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}

	updated, err := d.UpdateLibrary(context.Background(), created.ID, "Novels", `["/mnt/novels","/mnt/fiction"]`, LibraryOrganizationBookPerFolder, true)
	if err != nil {
		t.Fatalf("UpdateLibrary() error: %v", err)
	}
	if updated.Name != "Novels" {
		t.Errorf("Name = %q, want %q", updated.Name, "Novels")
	}
	if updated.Paths != `["/mnt/novels","/mnt/fiction"]` {
		t.Errorf("Paths = %q, want %q", updated.Paths, `["/mnt/novels","/mnt/fiction"]`)
	}
	if !updated.Monitored {
		t.Error("Monitored should be true")
	}
}

func TestUpdateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false); err != nil {
		t.Fatalf("CreateLibrary() for Fiction error: %v", err)
	}
	lib2, err := d.CreateLibrary(context.Background(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != nil {
		t.Fatalf("CreateLibrary() for Non-Fiction error: %v", err)
	}

	_, err = d.UpdateLibrary(context.Background(), lib2.ID, "Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestDeleteLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != nil {
		t.Fatalf("CreateLibrary() error: %v", err)
	}

	err = d.DeleteLibrary(context.Background(), lib.ID)
	if err != nil {
		t.Fatalf("DeleteLibrary() error: %v", err)
	}

	_, err = d.GetLibrary(context.Background(), lib.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteLibrary(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
