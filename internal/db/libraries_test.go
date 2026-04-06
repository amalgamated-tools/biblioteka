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
	require.NotEqual(t, "", lib.ID)
	require.Equal(t, "Fiction", lib.Name)
	require.Equal(t, `["/mnt/books/fiction"]`, lib.Paths)
	require.Equal(t, LibraryOrganizationBookPerFolder, lib.OrganizationType)
	require.False(t, lib.Monitored)
	require.False(t, lib.CreatedAt.IsZero())
	require.False(t, lib.UpdatedAt.IsZero())
}

func TestCreateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "first CreateLibrary() error")

	_, err = d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/other"]`, LibraryOrganizationBookPerFolder, false)
	require.ErrorIs(t, err, ErrLibraryNameExists)
}

func TestGetLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/books/fiction"]`, LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "CreateLibrary() error")

	found, err := d.GetLibrary(t.Context(), created.ID)
	require.NoError(t, err, "GetLibrary() error")
	require.Equal(t, created.ID, found.ID)
	require.Equal(t, "Fiction", found.Name)
	require.True(t, found.Monitored)
}

func TestGetLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetLibrary(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListLibraries(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() for Fiction error")
	_, err = d.CreateLibrary(t.Context(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "CreateLibrary() for Non-Fiction error")

	libs, err := d.ListLibraries(t.Context())
	require.NoError(t, err, "ListLibraries() error")
	require.Len(t, libs, 2)
	require.Equal(t, "Fiction", libs[0].Name)
}

func TestUpdateLibrary(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")

	updated, err := d.UpdateLibrary(t.Context(), created.ID, "Novels", `["/mnt/novels","/mnt/fiction"]`, LibraryOrganizationBookPerFolder, true)
	require.NoError(t, err, "UpdateLibrary() error")
	require.Equal(t, "Novels", updated.Name)
	require.Equal(t, `["/mnt/novels","/mnt/fiction"]`, updated.Paths)
	require.True(t, updated.Monitored)
}

func TestUpdateLibrary_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() for Fiction error")
	lib2, err := d.CreateLibrary(t.Context(), "Non-Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() for Non-Fiction error")

	_, err = d.UpdateLibrary(t.Context(), lib2.ID, "Fiction", `["/mnt/nonfiction"]`, LibraryOrganizationBookPerFolder, false)
	require.ErrorIs(t, err, ErrLibraryNameExists)
}

func TestDeleteLibrary(t *testing.T) {
	d := newTestDB(t)

	lib, err := d.CreateLibrary(t.Context(), "Fiction", `["/mnt/fiction"]`, LibraryOrganizationBookPerFolder, false)
	require.NoError(t, err, "CreateLibrary() error")

	err = d.DeleteLibrary(t.Context(), lib.ID)
	require.NoError(t, err, "DeleteLibrary() error")

	_, err = d.GetLibrary(t.Context(), lib.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteLibrary_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteLibrary(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}
