package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestUserForGroupList creates a user for use in reading group list tests.
func createTestUserForGroupList(t *testing.T, d *DB, email string) string {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", email, "password123")
	require.NoError(t, err, "createTestUserForGroupList")
	return u.ID
}

// makeGroupAndList is a helper that creates a group (with owner as member) and
// a reading list owned by ownerID. Returns (groupID, listID).
func makeGroupAndList(t *testing.T, d *DB, ownerID string) (string, string) {
	t.Helper()
	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)
	rl, err := d.CreateReadingList(t.Context(), ownerID, "Test List", nil)
	require.NoError(t, err)
	return g.ID, rl.ID
}

// TestShareListWithGroup_OwnerCanShare verifies that a list owner who is also
// a group member can share their list with the group.
func TestShareListWithGroup_OwnerCanShare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	groupID, listID := makeGroupAndList(t, d, ownerID)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)
	require.True(t, shared, "first share should return true")
}

// TestShareListWithGroup_Idempotent verifies that sharing the same list with
// the same group a second time is a no-op and returns (false, nil).
func TestShareListWithGroup_Idempotent(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	groupID, listID := makeGroupAndList(t, d, ownerID)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)
	require.True(t, shared)

	shared, err = d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)
	require.False(t, shared, "second share of same list+group should be a no-op")
}

// TestShareListWithGroup_NonOwnerCannotShare verifies that a group member who
// does not own the list cannot share it.
func TestShareListWithGroup_NonOwnerCannotShare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	memberID := createTestUserForGroupList(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)
	_, err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Test List", nil)
	require.NoError(t, err)

	// Member does not own the list — must be rejected.
	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestShareListWithGroup_NonMemberCannotShare verifies that even a list owner
// cannot share with a group they are not a member of.
func TestShareListWithGroup_NonMemberCannotShare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	strangerID := createTestUserForGroupList(t, d, "stranger@example.com")

	// Group owned by stranger — ownerID is not a member.
	g, err := d.CreateGroup(t.Context(), strangerID, "Stranger Group", nil)
	require.NoError(t, err)
	rl, err := d.CreateReadingList(t.Context(), ownerID, "My List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestShareListWithGroup_NonExistentList verifies that sharing a non-existent
// list returns an error (no rows from the ownership check).
func TestShareListWithGroup_NonExistentList(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, "nonexistent-list-id", ownerID)
	require.Error(t, err)
}

// TestUnshareListFromGroup_OwnerCanUnshare verifies that a list owner can
// remove their list from a group it was previously shared with.
func TestUnshareListFromGroup_OwnerCanUnshare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	groupID, listID := makeGroupAndList(t, d, ownerID)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	// After unsharing, the list should no longer appear in the group's lists.
	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

// TestUnshareListFromGroup_NonOwnerCannotUnshare verifies that a group member
// who does not own the list cannot unshare it.
func TestUnshareListFromGroup_NonOwnerCannotUnshare(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	memberID := createTestUserForGroupList(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)
	_, err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Test List", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, ownerID)
	require.NoError(t, err)

	// Member does not own the list — unshare must be rejected.
	err = d.UnshareListFromGroup(t.Context(), g.ID, rl.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestUnshareListFromGroup_NotSharedReturnsNoRows verifies that unsharing a
// list that was never shared returns sql.ErrNoRows.
func TestUnshareListFromGroup_NotSharedReturnsNoRows(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	groupID, listID := makeGroupAndList(t, d, ownerID)

	// Never shared — unshare should return sql.ErrNoRows.
	err := d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestListGroupReadingLists_MemberSeesSharedLists verifies that a group member
// sees all reading lists that have been shared with their group.
func TestListGroupReadingLists_MemberSeesSharedLists(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	memberID := createTestUserForGroupList(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)
	_, err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	rl1, err := d.CreateReadingList(t.Context(), ownerID, "Alpha List", nil)
	require.NoError(t, err)
	rl2, err := d.CreateReadingList(t.Context(), ownerID, "Beta List", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl1.ID, ownerID)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl2.ID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), g.ID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 2)

	// Results are ordered by name ascending.
	require.Equal(t, "Alpha List", lists[0].Name)
	require.Equal(t, "Beta List", lists[1].Name)
}

// TestListGroupReadingLists_EmptyWhenNoneShared verifies that the list is
// empty when no lists have been shared with the group.
func TestListGroupReadingLists_EmptyWhenNoneShared(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), g.ID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

// TestListGroupReadingLists_NonMemberCannotList verifies that a user who is
// not a member of the group receives sql.ErrNoRows.
func TestListGroupReadingLists_NonMemberCannotList(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	outsiderID := createTestUserForGroupList(t, d, "outsider@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Test Group", nil)
	require.NoError(t, err)
	rl, err := d.CreateReadingList(t.Context(), ownerID, "Test List", nil)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), g.ID, rl.ID, ownerID)
	require.NoError(t, err)

	_, err = d.ListGroupReadingLists(t.Context(), g.ID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestListGroupReadingLists_UnsharedListDisappears verifies that a list no
// longer appears in the group after it is unshared.
func TestListGroupReadingLists_UnsharedListDisappears(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroupList(t, d, "owner@example.com")
	groupID, listID := makeGroupAndList(t, d, ownerID)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Len(t, lists, 1)

	err = d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	lists, err = d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}
