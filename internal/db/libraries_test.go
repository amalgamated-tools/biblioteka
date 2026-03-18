package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCreateLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
	if err != nil {
		failNowf(t, "CreateLibrary() error: %v", err)
	}
	if lib.ID == "" {
		fail(t, "CreateLibrary() returned empty ID")
	}
	if lib.Name != "Fiction" {
		failf(t, "Name = %q, want %q", lib.Name, "Fiction")
	}
	if lib.Paths != `["/mnt/books/fiction"]` {
		failf(t, "Paths = %q, want %q", lib.Paths, `["/mnt/books/fiction"]`)
	}
	if lib.OrganizationType != "book_per_folder" {
		failf(t, "OrganizationType = %q, want %q", lib.OrganizationType, "book_per_folder")
	}
	if lib.Monitored {
		fail(t, "Monitored should be false")
	}
	if lib.CreatedAt.IsZero() {
		fail(t, "CreatedAt is zero")
	}
	if lib.UpdatedAt.IsZero() {
		fail(t, "UpdatedAt is zero")
	}
}

func TestCreateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
	if err != nil {
		failNowf(t, "first CreateLibrary() error: %v", err)
	}

	_, err = d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/other"]`, "book_per_folder", false)
	if err != ErrLibraryNameExists {
		failf(t, "expected ErrLibraryNameExists, got %v", err)
	}
}

func TestGetLibrary(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", true)

	found, err := d.GetLibrary(context.Background(), created.ID)
	if err != nil {
		failNowf(t, "GetLibrary() error: %v", err)
	}
	if found.ID != created.ID {
		failf(t, "ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "Fiction" {
		failf(t, "Name = %q, want %q", found.Name, "Fiction")
	}
	if !found.Monitored {
		fail(t, "Monitored should be true")
	}
}

func TestGetLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetLibrary(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}

func TestListLibraries(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	_, _ = d.CreateLibrary(context.Background(), "Non-Fiction", `["/mnt/nonfiction"]`, "book_per_folder", true)

	libs, err := d.ListLibraries(context.Background())
	if err != nil {
		failNowf(t, "ListLibraries() error: %v", err)
	}
	if len(libs) != 2 {
		failNowf(t, "ListLibraries() returned %d, want 2", len(libs))
	}
	if libs[0].Name != "Fiction" {
		failf(t, "first library Name = %q, want %q", libs[0].Name, "Fiction")
	}
}

func TestUpdateLibrary(t *testing.T) {
	d := newTestDB(t)

	created, _ := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	updated, err := d.UpdateLibrary(context.Background(), created.ID, "Novels", `["/mnt/novels","/mnt/fiction"]`, "book_per_folder", true)
	if err != nil {
		failNowf(t, "UpdateLibrary() error: %v", err)
	}
	if updated.Name != "Novels" {
		failf(t, "Name = %q, want %q", updated.Name, "Novels")
	}
	if updated.Paths != `["/mnt/novels","/mnt/fiction"]` {
		failf(t, "Paths = %q, want %q", updated.Paths, `["/mnt/novels","/mnt/fiction"]`)
	}
	if !updated.Monitored {
		fail(t, "Monitored should be true")
	}
}

func TestUpdateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, _ = d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	lib2, _ := d.CreateLibrary(context.Background(), "Non-Fiction", `["/mnt/nonfiction"]`, "book_per_folder", false)

	_, err := d.UpdateLibrary(context.Background(), lib2.ID, "Fiction", `["/mnt/nonfiction"]`, "book_per_folder", false)
	if err != ErrLibraryNameExists {
		failf(t, "expected ErrLibraryNameExists, got %v", err)
	}
}

func TestDeleteLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, _ := d.CreateLibrary(context.Background(), "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	err := d.DeleteLibrary(context.Background(), lib.ID)
	if err != nil {
		failNowf(t, "DeleteLibrary() error: %v", err)
	}

	_, err = d.GetLibrary(context.Background(), lib.ID)
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteLibrary(context.Background(), "nonexistent-id")
	if err != sql.ErrNoRows {
		failf(t, "expected sql.ErrNoRows, got %v", err)
	}
}
