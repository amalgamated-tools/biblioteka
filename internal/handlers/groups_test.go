package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/stretchr/testify/require"
)

func setupGroupHandler(t *testing.T) (*GroupHandler, string) {
	t.Helper()
	d := newTestDB(t)
	h := &GroupHandler{DB: d}
	user, err := d.CreateUser(t.Context(), "Test Owner", "owner@example.com", "password1")
	require.NoError(t, err, "create user")
	return h, user.ID
}

func createGroup(t *testing.T, h *GroupHandler, userID, name string) groupDTO {
	t.Helper()
	body := mustMarshal(t, groupRequest{Name: name})
	r := httptest.NewRequest(http.MethodPost, "/api/groups", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroups(w, r)
	require.Equal(t, http.StatusCreated, w.Code)
	var dto groupDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	return dto
}

func TestCreateGroup_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	dto := createGroup(t, h, userID, "Book Club")
	require.Equal(t, "Book Club", dto.Name)
	require.Equal(t, userID, dto.OwnerID)
	require.Equal(t, 1, dto.MemberCount)
}

func TestCreateGroup_MissingName(t *testing.T) {
	h, userID := setupGroupHandler(t)

	body := mustMarshal(t, groupRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/groups", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroups(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateGroup_DuplicateName(t *testing.T) {
	h, userID := setupGroupHandler(t)

	createGroup(t, h, userID, "Book Club")

	body := mustMarshal(t, groupRequest{Name: "Book Club"})
	r := httptest.NewRequest(http.MethodPost, "/api/groups", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroups(w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestListGroups_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	createGroup(t, h, userID, "Club A")
	createGroup(t, h, userID, "Club B")

	r := httptest.NewRequest(http.MethodGet, "/api/groups", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroups(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []groupDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2)
}

func TestGetGroup_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetGroup_NonMember404(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	other, err := h.DB.CreateUser(t.Context(), "Other", "other@example.com", "pw")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID, nil)
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateGroup_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	body := mustMarshal(t, groupRequest{Name: "Updated Club"})
	r := httptest.NewRequest(http.MethodPut, "/api/groups/"+g.ID, bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	var dto groupDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
	require.Equal(t, "Updated Club", dto.Name)
}

func TestUpdateGroup_NonOwner404(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	other, err := h.DB.CreateUser(t.Context(), "Other", "other@example.com", "pw")
	require.NoError(t, err)
	require.NoError(t, h.DB.AddGroupMember(t.Context(), g.ID, userID, other.ID))

	body := mustMarshal(t, groupRequest{Name: "Hacked"})
	r := httptest.NewRequest(http.MethodPut, "/api/groups/"+g.ID, bytes.NewReader(body))
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteGroup_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	r := httptest.NewRequest(http.MethodDelete, "/api/groups/"+g.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteGroup_NonOwner404(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	other, err := h.DB.CreateUser(t.Context(), "Other", "other@example.com", "pw")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/groups/"+g.ID, nil)
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddGroupMember_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	member, err := h.DB.CreateUser(t.Context(), "Member", "member@example.com", "pw")
	require.NoError(t, err)

	body := mustMarshal(t, addGroupMemberRequest{UserID: member.ID})
	r := httptest.NewRequest(http.MethodPost, "/api/groups/"+g.ID+"/members", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestAddGroupMember_InvalidUser(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	body := mustMarshal(t, addGroupMemberRequest{UserID: "nonexistent-user"})
	r := httptest.NewRequest(http.MethodPost, "/api/groups/"+g.ID+"/members", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddGroupMember_MissingUserID(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	body := mustMarshal(t, addGroupMemberRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/groups/"+g.ID+"/members", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveGroupMember_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	member, err := h.DB.CreateUser(t.Context(), "Member", "member@example.com", "pw")
	require.NoError(t, err)
	require.NoError(t, h.DB.AddGroupMember(t.Context(), g.ID, userID, member.ID))

	r := httptest.NewRequest(http.MethodDelete, "/api/groups/"+g.ID+"/members/"+member.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestRemoveGroupMember_OwnerCannotLeaveSelf(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	r := httptest.NewRequest(http.MethodDelete, "/api/groups/"+g.ID+"/members/"+userID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListGroupMembers_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	member, err := h.DB.CreateUser(t.Context(), "Member", "member@example.com", "pw")
	require.NoError(t, err)
	require.NoError(t, h.DB.AddGroupMember(t.Context(), g.ID, userID, member.ID))

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID+"/members", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []groupMemberDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 2)
}

func TestListGroupMembers_NonMember404(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	other, err := h.DB.CreateUser(t.Context(), "Other", "other@example.com", "pw")
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID+"/members", nil)
	r = withUserID(r, other.ID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestShareList_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.NoError(t, err)

	body := mustMarshal(t, shareListRequest{ListID: rl.ID})
	r := httptest.NewRequest(http.MethodPost, "/api/groups/"+g.ID+"/lists", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestShareList_MissingListID(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	body := mustMarshal(t, shareListRequest{})
	r := httptest.NewRequest(http.MethodPost, "/api/groups/"+g.ID+"/lists", bytes.NewReader(body))
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnshareList_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.NoError(t, err)
	_, err = h.DB.ShareListWithGroup(t.Context(), g.ID, rl.ID, userID)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodDelete, "/api/groups/"+g.ID+"/lists/"+rl.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestListGroupLists_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	rl, err := h.DB.CreateReadingList(t.Context(), userID, "Favorites", nil)
	require.NoError(t, err)
	_, err = h.DB.ShareListWithGroup(t.Context(), g.ID, rl.ID, userID)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID+"/lists", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGroupProgress_MissingBookID(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID+"/progress", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGroupProgress_Handler(t *testing.T) {
	h, userID := setupGroupHandler(t)

	g := createGroup(t, h, userID, "Book Club")

	b, err := h.DB.CreateBook(t.Context(), db.BookInput{Title: "Test Book"})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/api/groups/"+g.ID+"/progress?book_id="+b.ID, nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroupRoutes(w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var dtos []groupMemberProgressDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
	require.Len(t, dtos, 1) // Only the owner is a member.
}

func TestHandleGroups_MethodNotAllowed(t *testing.T) {
	h, userID := setupGroupHandler(t)

	r := httptest.NewRequest(http.MethodDelete, "/api/groups", nil)
	r = withUserID(r, userID)
	w := httptest.NewRecorder()
	h.HandleGroups(w, r)

	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
