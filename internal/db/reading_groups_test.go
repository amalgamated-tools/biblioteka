package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestUserForGroup(t *testing.T, d *DB, email string) string {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", email, "password123")
	require.NoError(t, err, "createTestUserForGroup")
	return u.ID
}

func TestCreateGroup(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NotEmpty(t, g.ID)
	require.Equal(t, ownerID, g.OwnerID)
	require.Equal(t, "Book Club", g.Name)
	require.Nil(t, g.Description)
	require.Equal(t, 1, g.MemberCount)
	require.False(t, g.CreatedAt.IsZero())
}

func TestCreateGroup_WithDescription(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	desc := "A club for book lovers"
	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", &desc)
	require.NoError(t, err)
	require.NotNil(t, g.Description)
	require.Equal(t, desc, *g.Description)
}

func TestCreateGroup_BlankName(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	for _, name := range []string{"", " ", "  \t  "} {
		_, err := d.CreateGroup(t.Context(), ownerID, name, nil)
		require.ErrorIs(t, err, ErrInvalidGroupName, "name=%q", name)
	}
}

func TestCreateGroup_DuplicateName(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	_, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.ErrorIs(t, err, ErrGroupNameExists)
}

func TestGetGroup_MemberCanAccess(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	fetched, err := d.GetGroup(t.Context(), g.ID, ownerID)
	require.NoError(t, err)
	require.Equal(t, g.ID, fetched.ID)
	require.Equal(t, "Book Club", fetched.Name)
}

func TestGetGroup_NonMemberCannotAccess(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.GetGroup(t.Context(), g.ID, otherID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGroups(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	_, err := d.CreateGroup(t.Context(), ownerID, "Club A", nil)
	require.NoError(t, err)
	_, err = d.CreateGroup(t.Context(), ownerID, "Club B", nil)
	require.NoError(t, err)

	groups, err := d.ListGroups(t.Context(), ownerID)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// Other user sees nothing.
	groups, err = d.ListGroups(t.Context(), otherID)
	require.NoError(t, err)
	require.Empty(t, groups)
}

func TestUpdateGroup_OwnerCanUpdate(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	desc := "Updated description"
	updated, err := d.UpdateGroup(t.Context(), g.ID, ownerID, "New Name", &desc)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.Name)
	require.NotNil(t, updated.Description)
	require.Equal(t, desc, *updated.Description)
}

func TestUpdateGroup_NonOwnerCannotUpdate(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.UpdateGroup(t.Context(), g.ID, otherID, "Hacked", nil)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteGroup_OwnerCanDelete(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.DeleteGroup(t.Context(), g.ID, ownerID)
	require.NoError(t, err)

	_, err = d.GetGroup(t.Context(), g.ID, ownerID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteGroup_NonOwnerCannotDelete(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.DeleteGroup(t.Context(), g.ID, otherID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestAddGroupMember(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	// Member should now see the group.
	fetched, err := d.GetGroup(t.Context(), g.ID, memberID)
	require.NoError(t, err)
	require.Equal(t, 2, fetched.MemberCount)
}

func TestAddGroupMember_NonOwnerCannotAdd(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")
	thirdID := createTestUserForGroup(t, d, "third@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.AddGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	// Member (not owner) cannot add others.
	err = d.AddGroupMember(t.Context(), g.ID, memberID, thirdID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestAddGroupMember_InvalidUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.AddGroupMember(t.Context(), g.ID, ownerID, "nonexistent-user-id")
	require.ErrorIs(t, err, ErrMemberUserNotFound)
}

func TestRemoveGroupMember_OwnerCanRemoveMember(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, memberID))

	err = d.RemoveGroupMember(t.Context(), g.ID, ownerID, memberID)
	require.NoError(t, err)

	// Member should no longer see the group.
	_, err = d.GetGroup(t.Context(), g.ID, memberID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRemoveGroupMember_MemberCanRemoveSelf(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, memberID))

	err = d.RemoveGroupMember(t.Context(), g.ID, memberID, memberID)
	require.NoError(t, err)
}

func TestRemoveGroupMember_OwnerCannotLeaveSelf(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	err = d.RemoveGroupMember(t.Context(), g.ID, ownerID, ownerID)
	require.ErrorIs(t, err, ErrOwnerCannotLeaveGroup)
}

func TestRemoveGroupMember_MemberCannotRemoveOther(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	member1ID := createTestUserForGroup(t, d, "member1@example.com")
	member2ID := createTestUserForGroup(t, d, "member2@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, member1ID))
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, member2ID))

	// Non-owner member cannot remove another member.
	err = d.RemoveGroupMember(t.Context(), g.ID, member1ID, member2ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGroupMembers(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, memberID))

	members, err := d.ListGroupMembers(t.Context(), g.ID, ownerID)
	require.NoError(t, err)
	require.Len(t, members, 2)
}

func TestListGroupMembers_NonMemberCannotList(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	_, err = d.ListGroupMembers(t.Context(), g.ID, otherID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestListGroupMemberProgress_IncludesMembersWithoutReadingState(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	memberID := createTestUserForGroup(t, d, "member@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)
	require.NoError(t, d.AddGroupMember(t.Context(), g.ID, ownerID, memberID))

	b, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Neither member has a reading state. With LEFT JOIN, both should appear.
	progress, err := d.ListGroupMemberProgress(t.Context(), g.ID, b.ID, ownerID)
	require.NoError(t, err)
	require.Len(t, progress, 2, "all members should appear even without reading states")
	for _, p := range progress {
		require.Equal(t, float64(0), p.Percentage)
		require.Nil(t, p.UpdatedAt)
	}
}

func TestIsMember(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForGroup(t, d, "owner@example.com")
	otherID := createTestUserForGroup(t, d, "other@example.com")

	g, err := d.CreateGroup(t.Context(), ownerID, "Book Club", nil)
	require.NoError(t, err)

	ok, err := d.IsMember(t.Context(), g.ID, ownerID)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = d.IsMember(t.Context(), g.ID, otherID)
	require.NoError(t, err)
	require.False(t, ok)
}
