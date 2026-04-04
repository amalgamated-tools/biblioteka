package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")
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

	_, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "first CreateLibrary() error")

	_, err = d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/other"]`, LibraryOrganizationBookPerFolder, false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestGetLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "CreateLibrary() error")

	found, err := d.GetLibrary(t.Context(), created.ID)
	require.NoError(t, err, "GetLibrary() error")
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

	_, err := d.GetLibrary(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListLibraries(t *testing.T) {
	d := newTestDB(t)

	if _, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false); err != nil {
		require.NoError(t, err, "CreateLibrary() for Fiction error")
	}
	if _, err := d.CreateLibrary(t.Context(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, true); err != nil {
		require.NoError(t, err, "CreateLibrary() for Non-Fiction error")
	}

	libs, err := d.ListLibraries(t.Context())
	require.NoError(t, err, "ListLibraries() error")
	if len(libs) != 2 {
		require.Failf(t, "failed", "ListLibraries() returned %d, want 2", len(libs))
	}
	if libs[0].Name != "Fiction" {
		t.Errorf("first library Name = %q, want %q", libs[0].Name, "Fiction")
	}
}

func TestUpdateLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")

	updated, err := d.UpdateLibrary(t.Context(), created.ID, "Novels", `["/mnt/novels","/mnt/fiction"]`, LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "UpdateLibrary() error")
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

	if _, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false); err != nil {
		require.NoError(t, err, "CreateLibrary() for Fiction error")
	}
	lib2, err := d.CreateLibrary(t.Context(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() for Non-Fiction error")

	_, err = d.UpdateLibrary(t.Context(), lib2.ID, "Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestDeleteLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")

	err = d.DeleteLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "DeleteLibrary() error")

	_, err = d.GetLibrary(t.Context(), lib.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteLibrary(t.Context(), "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
