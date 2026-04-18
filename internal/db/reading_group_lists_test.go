package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupGroupAndList creates a group with an owner who also has a reading list.
// It returns the owner's user ID, the group ID, and the reading list ID.
func setupGroupAndList(t *testing.T, d *DB) (ownerID, groupID, listID string) {
	t.Helper()
	ownerID = createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err, "CreateGroup")
	groupID = g.ID

	rl, err := d.CreateReadingList(t.Context(), ownerID, "Favorites", nil)
	require.NoError(t, err, "CreateReadingList")
	listID = rl.ID
	return
}

// --- ShareListWithGroup ---

func TestShareListWithGroup_OwnerMember(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	shared, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)
	require.True(t, shared, "first share should return true (newly shared)")
}

func TestShareListWithGroup_Idempotent(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	// Second call should succeed but indicate no new row was inserted.
	shared, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)
	require.False(t, shared, "second share should return false (already shared)")
}

func TestShareListWithGroup_NonOwnerOfListIsRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, _ := setupGroupAndList(t, d)

	// Create a second member who joins the group but does not own the list.
	memberID := createTestUserForGroup(t, d, "member@example.com")
	_, err := d.AddGroupMember(t.Context(), groupID, ownerID, memberID)
	require.NoError(t, err)

	// Create a separate reading list owned by memberID.
	memberList, err := d.CreateReadingList(t.Context(), memberID, "Member List", nil)
	require.NoError(t, err)

	// ownerID does not own memberList → should be rejected.
	_, err = d.ShareListWithGroup(t.Context(), groupID, memberList.ID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareListWithGroup_NonMemberIsRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, _ := setupGroupAndList(t, d)

	// Create a user who is NOT a member of the group but owns a reading list.
	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")
	outsiderList, err := d.CreateReadingList(t.Context(), outsiderID, "Outsider List", nil)
	require.NoError(t, err)

	// ownerID owns the group; outsiderID owns the list but is not in the group.
	_ = ownerID
	_, err = d.ShareListWithGroup(t.Context(), groupID, outsiderList.ID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestShareListWithGroup_NonexistentList(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, _ := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, "nonexistent-list-id", ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- UnshareListFromGroup ---

func TestUnshareListFromGroup(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	// List should no longer appear in group lists.
	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

func TestUnshareListFromGroup_NotShared(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	// Unsharing a list that was never shared should return sql.ErrNoRows.
	err := d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUnshareListFromGroup_NonOwnerIsRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	// A second member should not be able to unshare the owner's list.
	memberID := createTestUserForGroup(t, d, "member@example.com")
	_, err = d.AddGroupMember(t.Context(), groupID, ownerID, memberID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, listID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- ListGroupReadingLists ---

func TestListGroupReadingLists_MemberSeesSharedList(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	require.Equal(t, listID, lists[0].ID)
}

func TestListGroupReadingLists_EmptyWhenNoneShared(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, _ := setupGroupAndList(t, d)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}

func TestListGroupReadingLists_NonMemberIsRejected(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	outsiderID := createTestUserForGroup(t, d, "outsider@example.com")
	_, err = d.ListGroupReadingLists(t.Context(), groupID, outsiderID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGroupReadingLists_SecondMemberSeesSharedList(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	memberID := createTestUserForGroup(t, d, "member@example.com")
	_, err := d.AddGroupMember(t.Context(), groupID, ownerID, memberID)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, memberID)
	require.NoError(t, err)
	require.Len(t, lists, 1)
	require.Equal(t, listID, lists[0].ID)
}

func TestListGroupReadingLists_MultipleListsOrderedByName(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, _ := setupGroupAndList(t, d)

	// Create two additional lists with predictable names.
	alpha, err := d.CreateReadingList(t.Context(), ownerID, "Alpha", nil)
	require.NoError(t, err)
	zeta, err := d.CreateReadingList(t.Context(), ownerID, "Zeta", nil)
	require.NoError(t, err)

	_, err = d.ShareListWithGroup(t.Context(), groupID, zeta.ID, ownerID)
	require.NoError(t, err)
	_, err = d.ShareListWithGroup(t.Context(), groupID, alpha.ID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Len(t, lists, 2)
	require.Equal(t, "Alpha", lists[0].Name)
	require.Equal(t, "Zeta", lists[1].Name)
}

func TestListGroupReadingLists_UnsharedListIsHidden(t *testing.T) {
	d := newTestDB(t)
	ownerID, groupID, listID := setupGroupAndList(t, d)

	_, err := d.ShareListWithGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	err = d.UnshareListFromGroup(t.Context(), groupID, listID, ownerID)
	require.NoError(t, err)

	lists, err := d.ListGroupReadingLists(t.Context(), groupID, ownerID)
	require.NoError(t, err)
	require.Empty(t, lists)
}
