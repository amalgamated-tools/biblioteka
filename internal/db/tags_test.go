package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTag(t *testing.T) {
	d := newTestDB(t)

	tag, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)
	require.NotEmpty(t, tag.ID)
	require.Equal(t, "Fiction", tag.Name)
	require.False(t, tag.CreatedAt.IsZero())
}

func TestCreateTag_NormalizesWhitespace(t *testing.T) {
	d := newTestDB(t)

	tests := []struct {
		input string
		want  string
	}{
		{"  Science Fiction  ", "Science Fiction"},
		{"Young   Adult", "Young Adult"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tag, err := d.CreateTag(t.Context(), tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, tag.Name)
		})
	}
}

func TestCreateTag_DuplicateName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTag(t.Context(), "Horror")
	require.NoError(t, err)

	_, err = d.CreateTag(t.Context(), "Horror")
	require.ErrorIs(t, err, ErrTagNameExists)
}

func TestCreateTag_CaseInsensitiveDuplicate(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTag(t.Context(), "Mystery")
	require.NoError(t, err)

	_, err = d.CreateTag(t.Context(), "mystery")
	require.ErrorIs(t, err, ErrTagNameExists)
}

func TestCreateTag_BlankName(t *testing.T) {
	d := newTestDB(t)

	for _, name := range []string{"", " ", "\t"} {
		_, err := d.CreateTag(t.Context(), name)
		require.ErrorIs(t, err, ErrInvalidTagName)
	}
}

func TestGetTag(t *testing.T) {
	d := newTestDB(t)

	created, err := d.CreateTag(t.Context(), "Romance")
	require.NoError(t, err)

	fetched, err := d.GetTag(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Equal(t, "Romance", fetched.Name)
}

func TestGetTag_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetTag(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetTagByName(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTag(t.Context(), "Thriller")
	require.NoError(t, err)

	tag, err := d.GetTagByName(t.Context(), "thriller")
	require.NoError(t, err)
	require.Equal(t, "Thriller", tag.Name)
}

func TestUpdateTag(t *testing.T) {
	d := newTestDB(t)

	tag, err := d.CreateTag(t.Context(), "Sci-Fi")
	require.NoError(t, err)

	updated, err := d.UpdateTag(t.Context(), tag.ID, "Science Fiction")
	require.NoError(t, err)
	require.Equal(t, "Science Fiction", updated.Name)
}

func TestUpdateTag_BlankName(t *testing.T) {
	d := newTestDB(t)

	tag, err := d.CreateTag(t.Context(), "Fantasy")
	require.NoError(t, err)

	_, err = d.UpdateTag(t.Context(), tag.ID, "")
	require.ErrorIs(t, err, ErrInvalidTagName)
}

func TestListTags(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTag(t.Context(), "Horror")
	require.NoError(t, err)
	_, err = d.CreateTag(t.Context(), "Fantasy")
	require.NoError(t, err)

	tags, err := d.ListTags(t.Context())
	require.NoError(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, "Fantasy", tags[0].Name)
	require.Equal(t, "Horror", tags[1].Name)
}

func TestDeleteTag(t *testing.T) {
	d := newTestDB(t)

	tag, err := d.CreateTag(t.Context(), "Nonfiction")
	require.NoError(t, err)

	err = d.DeleteTag(t.Context(), tag.ID)
	require.NoError(t, err)

	_, err = d.GetTag(t.Context(), tag.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestFindOrCreateTag(t *testing.T) {
	d := newTestDB(t)

	tag1, err := d.FindOrCreateTag(t.Context(), "Biography")
	require.NoError(t, err)
	require.NotEmpty(t, tag1.ID)

	// Should return the same tag
	tag2, err := d.FindOrCreateTag(t.Context(), "biography")
	require.NoError(t, err)
	require.Equal(t, tag1.ID, tag2.ID)
}

func TestGetBookTags_Empty(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Empty(t, tags)
}

func TestSetBookTags(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag1, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)
	tag2, err := d.CreateTag(t.Context(), "Adventure")
	require.NoError(t, err)

	err = d.SetBookTags(t.Context(), book.ID, []string{tag1.ID, tag2.ID})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 2)
}

func TestSetBookTags_Replace(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag1, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)
	tag2, err := d.CreateTag(t.Context(), "Mystery")
	require.NoError(t, err)

	err = d.SetBookTags(t.Context(), book.ID, []string{tag1.ID})
	require.NoError(t, err)

	// Replace with different tag
	err = d.SetBookTags(t.Context(), book.ID, []string{tag2.ID})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, "Mystery", tags[0].Name)
}

func TestSetBookTags_DeduplicatesIDs(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)

	require.NoError(t, d.SetBookTags(t.Context(), book.ID, []string{tag.ID, tag.ID, tag.ID}))

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, tag.ID, tags[0].ID)
}

func TestSetBookTags_ClearAll(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)

	err = d.SetBookTags(t.Context(), book.ID, []string{tag.ID})
	require.NoError(t, err)

	err = d.SetBookTags(t.Context(), book.ID, []string{})
	require.NoError(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Empty(t, tags)
}

func TestSetBookTags_RollsBackOnInsertError(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Test Book"})
	require.NoError(t, err)

	tag, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)

	require.NoError(t, d.SetBookTags(t.Context(), book.ID, []string{tag.ID}))

	err = d.SetBookTags(t.Context(), book.ID, []string{"missing-tag-id"})
	require.Error(t, err)

	tags, err := d.GetBookTags(t.Context(), book.ID)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, tag.ID, tags[0].ID)
}

// ---- GetTagsForBooks ----

func TestGetTagsForBooks_EmptyInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetTagsForBooks(t.Context(), []string{})
	require.NoError(t, err, "GetTagsForBooks(empty) error")
	require.Nil(t, result)
}

func TestGetTagsForBooks_NilInput(t *testing.T) {
	d := newTestDB(t)

	result, err := d.GetTagsForBooks(t.Context(), nil)
	require.NoError(t, err, "GetTagsForBooks(nil) error")
	require.Nil(t, result)
}

func TestGetTagsForBooks_BookWithNoTags(t *testing.T) {
	d := newTestDB(t)

	book, err := d.CreateBook(t.Context(), BookInput{Title: "Untagged Book"})
	require.NoError(t, err)

	result, err := d.GetTagsForBooks(t.Context(), []string{book.ID})
	require.NoError(t, err, "GetTagsForBooks() error")
	require.Empty(t, result[book.ID])
}

func TestGetTagsForBooks_MultipleBooks(t *testing.T) {
	d := newTestDB(t)

	book1, err := d.CreateBook(t.Context(), BookInput{Title: "Book One"})
	require.NoError(t, err)
	book2, err := d.CreateBook(t.Context(), BookInput{Title: "Book Two"})
	require.NoError(t, err)

	tagA, err := d.CreateTag(t.Context(), "Fiction")
	require.NoError(t, err)
	tagB, err := d.CreateTag(t.Context(), "History")
	require.NoError(t, err)

	require.NoError(t, d.SetBookTags(t.Context(), book1.ID, []string{tagA.ID}))
	require.NoError(t, d.SetBookTags(t.Context(), book2.ID, []string{tagA.ID, tagB.ID}))

	result, err := d.GetTagsForBooks(t.Context(), []string{book1.ID, book2.ID})
	require.NoError(t, err, "GetTagsForBooks() error")

	require.Len(t, result[book1.ID], 1)
	require.Equal(t, "Fiction", result[book1.ID][0].Name)

	require.Len(t, result[book2.ID], 2)
	names := []string{result[book2.ID][0].Name, result[book2.ID][1].Name}
	require.Contains(t, names, "Fiction")
	require.Contains(t, names, "History")
}
