package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// helpers reuse createTestUserForGroup from reading_groups_test.go (same package).

func setupGroupWithMember(t *testing.T, d *DB) (ownerID, memberID, groupID string) {
	t.Helper()
	ownerID = createTestUserForGroup(t, d, "owner@example.com")
	memberID = createTestUserForGroup(t, d, "member@example.com")
	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	groupID = g.ID
	_, err = d.AddGroupMember(t.Context(), groupID, ownerID, memberID)
	require.NoError(t, err)
	return ownerID, memberID, groupID
}

// --- ShareListWithGroup ---

func TestShareListWithGroup_Basic(t *testing.T) {
	d := newTestDB(t)
	ownerID, _, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
	require.True(t, shared, "first share should return true (newly shared)")
}

func TestShareListWithGroup_Idempotent(t *testing.T) {
	d := newTestDB(t)
	ownerID, _, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
	require.False(t, shared, "second share of same list should return false (already shared)")
}

func TestShareListWithGroup_NonOwnerCannotShare(t *testing.T) {
	d := newTestDB(t)
	ownerID, memberID, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareListWithGroup_NonMemberCannotShare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	rl, err := d.CreateReadingList(t.Context(), outsiderID, "Outsider List", nil)
	require.NoError(t, err)

	// outsiderID owns the list but is NOT a group member
	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareListWithGroup_NonExistentList(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, "nonexistent-list-id", ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- UnshareListFromGroup ---

func TestUnshareListFromGroup_Basic(t *testing.T) {
	d := newTestDB(t)
	ownerID, memberID, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	// List should no longer appear in group's lists
	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

func TestUnshareListFromGroup_NonOwnerCannotUnshare(t *testing.T) {
	d := newTestDB(t)
	ownerID, memberID, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	// memberID did not create the list; cannot unshare it
	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUnshareListFromGroup_NotShared(t *testing.T) {
	d := newTestDB(t)
	ownerID, _, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	// List was never shared; unsharing should return ErrNoRows
	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- ListGroupReadingLists ---

func TestListGroupReadingLists_Empty(t *testing.T) {
	d := newTestDB(t)
	_, memberID, groupID := setupGroupWithMember(t, d)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

func TestListGroupReadingLists_ReturnSharedLists(t *testing.T) {
	d := newTestDB(t)
	ownerID, memberID, groupID := setupGroupWithMember(t, d)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	require.Equal(t, rl.ID, lists[0].ID)
	require.Equal(t, "My List", lists[0].Name)
}

func TestListGroupReadingLists_NonMemberCannotList(t *testing.T) {
	d := newTestDB(t)
	ownerID, _, groupID := setupGroupWithMember(t, d)
	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")

	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	_, err = d.ListGroupReadingLists(t.Context(), groupID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGroupReadingLists_MultipleLists(t *testing.T) {
	d := newTestDB(t)
	ownerID, memberID, groupID := setupGroupWithMember(t, d)

	for _, name := range []string{"Alpha List", "Beta List", "Gamma List"} {
		rl, err := d.CreateReadingList(t.Context(), ownerID, name, nil)
		require.NoError(t, err)
		_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
		require.NoError(t, err)
	}

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 3)
	// Results are ordered by name ASC
	require.Equal(t, "Alpha List", lists[0].Name)
	require.Equal(t, "Beta List", lists[1].Name)
	require.Equal(t, "Gamma List", lists[2].Name)
}
