package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestGroupAndMember creates a group owned by ownerID and adds memberID as a member.
// It returns the group ID.
func createTestGroupAndMember(t *testing.T, d *DB, ownerID, memberID string) string {
	t.Helper()
	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	_, err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)
	return g.ID
}

// TestShareListWithGroup_OwnerCanShare verifies that a list owner who is also
// a group member can share their list with the group.
func TestShareListWithGroup_OwnerCanShare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
	require.True(t, shared, "first share should return true")
}

// TestShareListWithGroup_Idempotent verifies that sharing the same list twice
// is idempotent and returns false on the second call.
func TestShareListWithGroup_Idempotent(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
	require.True(t, shared)

	shared, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
	require.False(t, shared, "second share should return false (idempotent)")
}

// TestShareListWithGroup_NonOwnerRejected verifies that a group member who
// does not own the list cannot share it.
func TestShareListWithGroup_NonOwnerRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows, "non-owner should not be able to share the list")
}

// TestShareListWithGroup_NonMemberRejected verifies that a list owner who is
// not a group member cannot share the list with that group.
func TestShareListWithGroup_NonMemberRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	rl, err := d.CreateReadingList(t.Context(), outsiderID, "Favorites", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows, "non-member should not be able to share into the group")
}

// TestShareListWithGroup_NonExistentList verifies that sharing a non-existent
// list returns an error.
func TestShareListWithGroup_NonExistentList(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, "nonexistent-list-id", ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestUnshareListFromGroup_OwnerCanUnshare verifies that a list owner can
// unshare their list from a group.
func TestUnshareListFromGroup_OwnerCanUnshare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)
}

// TestUnshareListFromGroup_NonOwnerRejected verifies that a group member who
// does not own the list cannot unshare it.
func TestUnshareListFromGroup_NonOwnerRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows, "non-owner should not be able to unshare the list")
}

// TestUnshareListFromGroup_NeverShared verifies that unsharing a list that was
// never shared with the group returns sql.ErrNoRows.
func TestUnshareListFromGroup_NeverShared(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), g.ID, rl.ID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestListGroupReadingLists_MemberSeesSharedLists verifies that a group member
// can see lists shared with the group, returned in alphabetical order.
func TestListGroupReadingLists_MemberSeesSharedLists(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl1, err := d.CreateReadingList(t.Context(), ownerID, "Zebra", nil)
	require.NoError(t, err)
	rl2, err := d.CreateReadingList(t.Context(), ownerID, "Alpha", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, rl1.ID, ownerID)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl2.ID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 2)
	require.Equal(t, "Alpha", lists[0].Name, "lists should be ordered alphabetically")
	require.Equal(t, "Zebra", lists[1].Name)
}

// TestListGroupReadingLists_EmptyWhenNothingShared verifies that the list is
// empty when no reading lists have been shared with the group.
func TestListGroupReadingLists_EmptyWhenNothingShared(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), g.ID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

// TestListGroupReadingLists_NonMemberRejected verifies that a non-member
// cannot list the group's reading lists.
func TestListGroupReadingLists_NonMemberRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.ListGroupReadingLists(t.Context(), g.ID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestListGroupReadingLists_UnsharedListDisappears verifies that a list
// disappears from the group after it is unshared.
func TestListGroupReadingLists_UnsharedListDisappears(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	groupID := createTestGroupAndMember(t, d, ownerID, memberID)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 1)

	err = d.UnshareListFromGroup(t.Context(), groupID, rl.ID, ownerID)
	require.NoError(t, err)

	lists, err = d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Empty(t, lists, "list should disappear after unsharing")
}
