package db

import (
	"database/sql"
	"testing"
)

func TestCreateLibrary(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	lib, err := d.CreateLibrary(user.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
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
	if lib.OrganizationType != "book_per_folder" {
		t.Errorf("OrganizationType = %q, want %q", lib.OrganizationType, "book_per_folder")
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
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	_, err := d.CreateLibrary(user.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
	if err != nil {
		t.Fatalf("first CreateLibrary() error: %v", err)
	}

	_, err = d.CreateLibrary(user.ID, "Fiction", `["/mnt/books/other"]`, "book_per_folder", false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestCreateLibrary_SameNameDifferentUsers(t *testing.T) {
	d := newTestDB(t)
	u1, _ := d.CreateUser("Alice", "alice@example.com", "pw")
	u2, _ := d.CreateUser("Bob", "bob@example.com", "pw")

	_, err := d.CreateLibrary(u1.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
	if err != nil {
		t.Fatalf("CreateLibrary for user1 error: %v", err)
	}

	_, err = d.CreateLibrary(u2.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)
	if err != nil {
		t.Fatalf("CreateLibrary for user2 should succeed, got: %v", err)
	}
}

func TestGetLibrary(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	created, _ := d.CreateLibrary(user.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", true)

	found, err := d.GetLibrary(user.ID, created.ID)
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

func TestGetLibrary_WrongUser(t *testing.T) {
	d := newTestDB(t)
	u1, _ := d.CreateUser("Alice", "alice@example.com", "pw")
	u2, _ := d.CreateUser("Bob", "bob@example.com", "pw")

	lib, _ := d.CreateLibrary(u1.ID, "Fiction", `["/mnt/books/fiction"]`, "book_per_folder", false)

	_, err := d.GetLibrary(u2.ID, lib.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	_, err := d.GetLibrary(user.ID, "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListLibraries(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	_, _ = d.CreateLibrary(user.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	_, _ = d.CreateLibrary(user.ID, "Non-Fiction", `["/mnt/nonfiction"]`, "book_per_folder", true)

	libs, err := d.ListLibraries(user.ID)
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

func TestListLibraries_UserIsolation(t *testing.T) {
	d := newTestDB(t)
	u1, _ := d.CreateUser("Alice", "alice@example.com", "pw")
	u2, _ := d.CreateUser("Bob", "bob@example.com", "pw")

	_, _ = d.CreateLibrary(u1.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	_, _ = d.CreateLibrary(u2.ID, "Sci-Fi", `["/mnt/scifi"]`, "book_per_folder", false)

	libs, err := d.ListLibraries(u1.ID)
	if err != nil {
		t.Fatalf("ListLibraries() error: %v", err)
	}
	if len(libs) != 1 {
		t.Fatalf("ListLibraries() returned %d, want 1", len(libs))
	}
	if libs[0].Name != "Fiction" {
		t.Errorf("Name = %q, want %q", libs[0].Name, "Fiction")
	}
}

func TestUpdateLibrary(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	created, _ := d.CreateLibrary(user.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	updated, err := d.UpdateLibrary(user.ID, created.ID, "Novels", `["/mnt/novels","/mnt/fiction"]`, "book_per_folder", true)
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

func TestUpdateLibrary_WrongUser(t *testing.T) {
	d := newTestDB(t)
	u1, _ := d.CreateUser("Alice", "alice@example.com", "pw")
	u2, _ := d.CreateUser("Bob", "bob@example.com", "pw")

	lib, _ := d.CreateLibrary(u1.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	_, err := d.UpdateLibrary(u2.ID, lib.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	_, _ = d.CreateLibrary(user.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)
	lib2, _ := d.CreateLibrary(user.ID, "Non-Fiction", `["/mnt/nonfiction"]`, "book_per_folder", false)

	_, err := d.UpdateLibrary(user.ID, lib2.ID, "Fiction", `["/mnt/nonfiction"]`, "book_per_folder", false)
	if err != ErrLibraryNameExists {
		t.Errorf("expected ErrLibraryNameExists, got %v", err)
	}
}

func TestDeleteLibrary(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	lib, _ := d.CreateLibrary(user.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	err := d.DeleteLibrary(user.ID, lib.ID)
	if err != nil {
		t.Fatalf("DeleteLibrary() error: %v", err)
	}

	_, err = d.GetLibrary(user.ID, lib.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteLibrary_WrongUser(t *testing.T) {
	d := newTestDB(t)
	u1, _ := d.CreateUser("Alice", "alice@example.com", "pw")
	u2, _ := d.CreateUser("Bob", "bob@example.com", "pw")

	lib, _ := d.CreateLibrary(u1.ID, "Fiction", `["/mnt/fiction"]`, "book_per_folder", false)

	err := d.DeleteLibrary(u2.ID, lib.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)
	user, _ := d.CreateUser("Alice", "alice@example.com", "pw")

	err := d.DeleteLibrary(user.ID, "nonexistent-id")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
