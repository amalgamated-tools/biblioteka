package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTestUserForEnrichment creates a user to own AI enrichments in tests.
func createTestUserForEnrichment(t *testing.T, d *DB, email string) string {
	t.Helper()
	u, err := d.CreateUser(t.Context(), "Test User", email, "password123")
	require.NoError(t, err)
	return u.ID
}

// makeAIEnrichment is a convenience wrapper that creates an enrichment with
// sensible defaults; individual test cases override only what they care about.
func makeAIEnrichment(t *testing.T, d *DB, userID string, bookID *string) *AIEnrichment {
	t.Helper()
	e, err := d.CreateAIEnrichment(
		t.Context(),
		userID,
		bookID,
		"openai", "gpt-4",
		[]string{"fiction", "adventure"},
		new("young adult"),
		new("A thrilling adventure."),
		`{"raw":"data"}`,
	)
	require.NoError(t, err)
	return e
}

// --- CreateAIEnrichment ---

func TestCreateAIEnrichment(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	bookID := book.ID
	e, err := d.CreateAIEnrichment(
		t.Context(),
		userID,
		&bookID,
		"openai", "gpt-4",
		[]string{"fiction"},
		new("adult"),
		new("A great read."),
		`{"choices":[]}`,
	)
	require.NoError(t, err)
	require.NotEmpty(t, e.ID)
	require.Equal(t, userID, e.UserID)
	require.NotNil(t, e.BookID)
	require.Equal(t, bookID, *e.BookID)
	require.Equal(t, AIEnrichmentStatusPending, e.Status)
	require.Equal(t, "openai", e.Provider)
	require.Equal(t, "gpt-4", e.Model)
	require.Equal(t, []string{"fiction"}, e.SuggestedTags)
	require.Equal(t, "adult", *e.ReadingLevel)
	require.Equal(t, "A great read.", *e.GeneratedDescription)
	require.Equal(t, `{"choices":[]}`, e.RawResponse)
	require.False(t, e.CreatedAt.IsZero())
}

func TestCreateAIEnrichment_NilBookID(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e, err := d.CreateAIEnrichment(t.Context(), userID, nil, "openai", "gpt-4", []string{}, nil, nil, "{}")
	require.NoError(t, err)
	require.Nil(t, e.BookID)
	require.Equal(t, AIEnrichmentStatusPending, e.Status)
}

func TestCreateAIEnrichment_NilSuggestedTagsBecomesEmpty(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e, err := d.CreateAIEnrichment(t.Context(), userID, nil, "openai", "gpt-4", nil, nil, nil, "{}")
	require.NoError(t, err)
	require.NotNil(t, e.SuggestedTags)
	require.Empty(t, e.SuggestedTags)
}

// --- GetAIEnrichment ---

func TestGetAIEnrichment(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	created := makeAIEnrichment(t, d, userID, nil)

	fetched, err := d.GetAIEnrichment(t.Context(), userID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Equal(t, userID, fetched.UserID)
	require.Equal(t, []string{"fiction", "adventure"}, fetched.SuggestedTags)
}

func TestGetAIEnrichment_AllFields(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	created := makeAIEnrichment(t, d, userID, nil)

	fetched, err := d.GetAIEnrichment(t.Context(), userID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Equal(t, userID, fetched.UserID)
	require.Nil(t, fetched.BookID)
	require.Equal(t, AIEnrichmentStatusPending, fetched.Status)
	require.Equal(t, "openai", fetched.Provider)
	require.Equal(t, "gpt-4", fetched.Model)
	require.Equal(t, []string{"fiction", "adventure"}, fetched.SuggestedTags)
	require.NotNil(t, fetched.ReadingLevel)
	require.Equal(t, "young adult", *fetched.ReadingLevel)
	require.NotNil(t, fetched.GeneratedDescription)
	require.Equal(t, "A thrilling adventure.", *fetched.GeneratedDescription)
	require.Equal(t, `{"raw":"data"}`, fetched.RawResponse)
	require.False(t, fetched.CreatedAt.IsZero())
	require.False(t, fetched.UpdatedAt.IsZero())
}

func TestGetAIEnrichment_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	_, err := d.GetAIEnrichment(t.Context(), userID, "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetAIEnrichment_WrongUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForEnrichment(t, d, "owner@example.com")
	otherID := createTestUserForEnrichment(t, d, "other@example.com")

	created := makeAIEnrichment(t, d, ownerID, nil)

	_, err := d.GetAIEnrichment(t.Context(), otherID, created.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- GetPendingAIEnrichmentByBook ---

func TestGetPendingAIEnrichmentByBook(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	created := makeAIEnrichment(t, d, userID, &book.ID)

	fetched, err := d.GetPendingAIEnrichmentByBook(t.Context(), userID, book.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
}

func TestGetPendingAIEnrichmentByBook_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	_, err = d.GetPendingAIEnrichmentByBook(t.Context(), userID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetPendingAIEnrichmentByBook_IgnoresNonPendingAndReturnsPending(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Reject the first enrichment; the second (still pending) must be returned.
	first := makeAIEnrichment(t, d, userID, &book.ID)
	_, err = d.UpdateAIEnrichmentStatus(t.Context(), userID, first.ID, AIEnrichmentStatusRejected)
	require.NoError(t, err)

	second := makeAIEnrichment(t, d, userID, &book.ID)

	fetched, err := d.GetPendingAIEnrichmentByBook(t.Context(), userID, book.ID)
	require.NoError(t, err)
	require.Equal(t, second.ID, fetched.ID)
}

func TestGetPendingAIEnrichmentByBook_SkipsNonPending(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	// Reject the enrichment so it is no longer pending.
	_, err = d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusRejected)
	require.NoError(t, err)

	_, err = d.GetPendingAIEnrichmentByBook(t.Context(), userID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetPendingAIEnrichmentByBook_WrongUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForEnrichment(t, d, "owner@example.com")
	otherID := createTestUserForEnrichment(t, d, "other@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	makeAIEnrichment(t, d, ownerID, &book.ID)

	_, err = d.GetPendingAIEnrichmentByBook(t.Context(), otherID, book.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- UpdateAIEnrichmentStatus ---

func TestUpdateAIEnrichmentStatus_ToApplied(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e := makeAIEnrichment(t, d, userID, nil)

	updated, err := d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusApplied)
	require.NoError(t, err)
	require.Equal(t, AIEnrichmentStatusApplied, updated.Status)
}

func TestUpdateAIEnrichmentStatus_ToRejected(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e := makeAIEnrichment(t, d, userID, nil)

	updated, err := d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusRejected)
	require.NoError(t, err)
	require.Equal(t, AIEnrichmentStatusRejected, updated.Status)
}

func TestUpdateAIEnrichmentStatus_InvalidStatus(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e := makeAIEnrichment(t, d, userID, nil)

	_, err := d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, "invalid-status")
	require.ErrorIs(t, err, ErrInvalidAIEnrichmentStatus)
}

func TestUpdateAIEnrichmentStatus_AlreadyApplied(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e := makeAIEnrichment(t, d, userID, nil)
	_, err := d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusApplied)
	require.NoError(t, err)

	// Attempting to update again should fail because it's no longer pending.
	_, err = d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusRejected)
	require.ErrorIs(t, err, ErrAIEnrichmentNotPending)
}

func TestUpdateAIEnrichmentStatus_WrongUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForEnrichment(t, d, "owner@example.com")
	otherID := createTestUserForEnrichment(t, d, "other@example.com")

	e := makeAIEnrichment(t, d, ownerID, nil)

	_, err := d.UpdateAIEnrichmentStatus(t.Context(), otherID, e.ID, AIEnrichmentStatusApplied)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUpdateAIEnrichmentStatus_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	_, err := d.UpdateAIEnrichmentStatus(t.Context(), userID, "nonexistent-id", AIEnrichmentStatusApplied)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- DeleteAIEnrichment ---

func TestDeleteAIEnrichment(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	e := makeAIEnrichment(t, d, userID, nil)

	err := d.DeleteAIEnrichment(t.Context(), userID, e.ID)
	require.NoError(t, err)

	_, err = d.GetAIEnrichment(t.Context(), userID, e.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteAIEnrichment_NotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")

	err := d.DeleteAIEnrichment(t.Context(), userID, "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteAIEnrichment_WrongUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForEnrichment(t, d, "owner@example.com")
	otherID := createTestUserForEnrichment(t, d, "other@example.com")

	e := makeAIEnrichment(t, d, ownerID, nil)

	err := d.DeleteAIEnrichment(t.Context(), otherID, e.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// --- ApplyAIEnrichment ---

func TestApplyAIEnrichment_MarksAsApplied(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	result, err := d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
	})
	require.NoError(t, err)
	require.Equal(t, AIEnrichmentStatusApplied, result.Status)
}

func TestApplyAIEnrichment_UnionMergesTags(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Create two tags; set one on the book before applying the enrichment.
	existingTag, err := d.CreateTag(t.Context(), "Existing")
	require.NoError(t, err)
	newTag, err := d.CreateTag(t.Context(), "New")
	require.NoError(t, err)

	err = d.SetBookTags(t.Context(), book.ID, []string{existingTag.ID})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{newTag.ID},
	})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 2)
	tagIDs := make(map[string]bool)
	for _, tg := range tags {
		tagIDs[tg.ID] = true
	}
	require.True(t, tagIDs[existingTag.ID], "existing tag should be preserved")
	require.True(t, tagIDs[newTag.ID], "new tag should be added")
}

func TestApplyAIEnrichment_SetsDescription(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)
	desc := "AI-generated description."

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
		Description:  &desc,
	})
	require.NoError(t, err)

	updated, err := d.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Description)
	require.Equal(t, desc, *updated.Description)
}

func TestApplyAIEnrichment_DoesNotOverwriteExistingDescription(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	existing := "Human-written description."
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book", Description: &existing})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)
	aiDesc := "AI-generated description."

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
		Description:  &aiDesc,
	})
	require.NoError(t, err)

	updated, err := d.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.Description)
	require.Equal(t, existing, *updated.Description, "existing description must not be overwritten")
}

func TestApplyAIEnrichment_NilDescriptionLeavesBookUnchanged(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
		Description:  nil,
	})
	require.NoError(t, err)

	updated, err := d.GetBook(t.Context(), book.ID)
	require.NoError(t, err)
	require.Nil(t, updated.Description)
}

func TestApplyAIEnrichment_NotPending(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Pre-seed a tag to verify the transaction rollback preserves it.
	tag, err := d.CreateTag(t.Context(), "Existing")
	require.NoError(t, err)
	err = d.SetBookTags(t.Context(), book.ID, []string{tag.ID})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	// Reject first, then try to apply.
	_, err = d.UpdateAIEnrichmentStatus(t.Context(), userID, e.ID, AIEnrichmentStatusRejected)
	require.NoError(t, err)

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
	})
	require.ErrorIs(t, err, ErrAIEnrichmentNotPending)

	// Book tags must be unchanged (transaction rolled back).
	bookTags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, bookTags, 1)
}

func TestApplyAIEnrichment_WrongUser(t *testing.T) {
	d := newTestDB(t)
	ownerID := createTestUserForEnrichment(t, d, "owner@example.com")
	otherID := createTestUserForEnrichment(t, d, "other@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	// Pre-seed a tag to verify the transaction rollback preserves it.
	tag, err := d.CreateTag(t.Context(), "Tag")
	require.NoError(t, err)
	err = d.SetBookTags(t.Context(), book.ID, []string{tag.ID})
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, ownerID, &book.ID)

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       otherID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{},
	})
	require.ErrorIs(t, err, ErrAIEnrichmentNotPending)

	// Book tags must be unchanged (transaction rolled back).
	bookTags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, bookTags, 1)
}

func TestApplyAIEnrichment_DeduplicatesNewTagIDs(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag, err := d.CreateTag(t.Context(), "Duplicate")
	require.NoError(t, err)

	e := makeAIEnrichment(t, d, userID, &book.ID)

	// Passing the same tag ID twice should not cause a duplicate-key error.
	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: e.ID,
		NewTagIDs:    []string{tag.ID, tag.ID},
	})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 1)
}

func TestApplyAIEnrichment_EnrichmentNotFound(t *testing.T) {
	d := newTestDB(t)
	userID := createTestUserForEnrichment(t, d, "user@example.com")
	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	_, err = d.ApplyAIEnrichment(t.Context(), ApplyAIEnrichmentInput{
		BookID:       book.ID,
		UserID:       userID,
		EnrichmentID: "nonexistent-id",
		NewTagIDs:    []string{},
	})
	require.ErrorIs(t, err, ErrAIEnrichmentNotPending)
}
