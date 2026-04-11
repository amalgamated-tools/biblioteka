package kobo

import (
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"

	"github.com/stretchr/testify/require"
)

func TestFormatForFileType(t *testing.T) {
	cases := []struct {
		ft   string
		want string
		ok   bool
	}{
		{"epub", "EPUB3", true},
		{"EPUB", "EPUB3", true},
		{"kepub", "KEPUB", true},
		{"mobi", "MOBI", true},
		{"pdf", "PDF", true},
		{"azw3", "AZW3", true},
		{"txt", "", false},
		{"cbz", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := FormatForFileType(tc.ft)
		require.Equal(t, tc.ok, ok)
		require.Equal(t, tc.want, got)
	}
}

func TestDownloadURLs_OnlySupportedFormats(t *testing.T) {
	files := []db.BookFile{
		{ID: "f1", FileType: "epub", FileSize: 100},
		{ID: "f2", FileType: "txt", FileSize: 10}, // not supported
		{ID: "f3", FileType: "mobi", FileSize: 200},
	}
	urls := DownloadURLs("https://host", "tok", "book1", files)
	require.Len(t, urls, 2)
	require.Equal(t, "EPUB3", urls[0].Format)
	require.Equal(t, "MOBI", urls[1].Format)
}

func TestDownloadURLs_URLContainsToken(t *testing.T) {
	files := []db.BookFile{{ID: "f1", FileType: "epub"}}
	urls := DownloadURLs("https://host", "mytoken", "book1", files)
	require.Len(t, urls, 1)
	require.Contains(t, urls[0].URL, "mytoken")
}

func TestDownloadURLs_Empty(t *testing.T) {
	urls := DownloadURLs("https://host", "tok", "book1", nil)
	require.Nil(t, urls)
}

func TestBookEntitlement_Fields(t *testing.T) {
	now := db.Timestamp{Time: time.Now().UTC()}
	book := &db.Book{
		ID:        "bk1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	ent := BookEntitlement(book)
	require.Equal(t, "bk1", ent.ID)
	require.Equal(t, "Full", ent.Accessibility)
	require.Equal(t, "Active", ent.Status)
}

func TestBookMetadata_Authors(t *testing.T) {
	book := &db.Book{ID: "bk1", Title: "My Book"}
	authors := []db.Author{{ID: "a1", Name: "Alice"}, {ID: "a2", Name: "Bob"}}
	meta := BookMetadata(book, authors, nil, nil)

	require.Len(t, meta.Contributors, 2)
	require.Equal(t, "Alice", meta.Contributors[0])
}

func TestBookMetadata_Series(t *testing.T) {
	pos := 3.0
	book := &db.Book{ID: "bk1", Title: "Book Three"}
	series := []db.BookSeriesEntry{
		{Series: db.Series{ID: "s1", Name: "My Series"}, Position: &pos},
	}
	meta := BookMetadata(book, nil, series, nil)

	require.NotNil(t, meta.Series)
	require.Equal(t, "My Series", meta.Series.Name)
	require.Equal(t, 3, meta.Series.Number)
}

func TestBookMetadata_Language_Default(t *testing.T) {
	book := &db.Book{ID: "bk1", Title: "No Lang"}
	meta := BookMetadata(book, nil, nil, nil)
	require.Equal(t, "en", meta.Language)
}
	book := &db.Book{ID: "bk1", Title: "French", Language: new("fr")}
	lang := "fr"
	book := &db.Book{ID: "bk1", Title: "French", Language: &lang}
	meta := BookMetadata(book, nil, nil, nil)
	require.Equal(t, "fr", meta.Language)
}

func TestReadingStateResponse_Defaults(t *testing.T) {
	state := &db.KoboReadingState{
		BookID: "bk1",
		Status: "ReadyToRead",
	}
	resp := ReadingStateResponse(state)
	require.Equal(t, "bk1", resp.EntitlementID)
	require.Equal(t, "ReadyToRead", resp.StatusInfo.Status)
}

func TestReadingStateResponse_WithProgress(t *testing.T) {
	pct := 42.5
	state := &db.KoboReadingState{
		BookID:      "bk1",
		Status:      "Reading",
		PercentRead: &pct,
	}
	resp := ReadingStateResponse(state)
	require.NotNil(t, resp.CurrentBookmark.ProgressPercent)
	require.Equal(t, 42.5, *resp.CurrentBookmark.ProgressPercent)
}
