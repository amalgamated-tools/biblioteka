package kobo

import (
	"strings"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

func strPtr(s string) *string   { return &s }
func f64Ptr(f float64) *float64 { return &f }

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
		if ok != tc.ok || got != tc.want {
			t.Errorf("FormatForFileType(%q) = (%q, %v), want (%q, %v)", tc.ft, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDownloadURLs_OnlySupportedFormats(t *testing.T) {
	files := []db.BookFile{
		{ID: "f1", FileType: "epub", FileSize: 100},
		{ID: "f2", FileType: "txt", FileSize: 10}, // not supported
		{ID: "f3", FileType: "mobi", FileSize: 200},
	}
	urls := DownloadURLs("https://host", "tok", "book1", files)
	if len(urls) != 2 {
		t.Fatalf("len = %d, want 2", len(urls))
	}
	if urls[0]["Format"] != "EPUB3" {
		t.Errorf("first format = %v, want EPUB3", urls[0]["Format"])
	}
	if urls[1]["Format"] != "MOBI" {
		t.Errorf("second format = %v, want MOBI", urls[1]["Format"])
	}
}

func TestDownloadURLs_URLContainsToken(t *testing.T) {
	files := []db.BookFile{{ID: "f1", FileType: "epub"}}
	urls := DownloadURLs("https://host", "mytoken", "book1", files)
	if len(urls) != 1 {
		t.Fatalf("len = %d, want 1", len(urls))
	}
	got := urls[0]["Url"].(string)
	if !strings.Contains(got, "mytoken") {
		t.Errorf("URL %q does not contain token", got)
	}
}

func TestDownloadURLs_Empty(t *testing.T) {
	urls := DownloadURLs("https://host", "tok", "book1", nil)
	if urls != nil {
		t.Errorf("expected nil for no files, got %v", urls)
	}
}

func TestBookEntitlement_Fields(t *testing.T) {
	now := db.Timestamp{Time: time.Now().UTC()}
	book := &db.Book{
		ID:        "bk1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	ent := BookEntitlement(book)
	if ent["Id"] != "bk1" {
		t.Errorf("Id = %v", ent["Id"])
	}
	if ent["Accessibility"] != "Full" {
		t.Errorf("Accessibility = %v", ent["Accessibility"])
	}
	if ent["Status"] != "Active" {
		t.Errorf("Status = %v", ent["Status"])
	}
}

func TestBookMetadata_Authors(t *testing.T) {
	book := &db.Book{ID: "bk1", Title: "My Book"}
	authors := []db.Author{{ID: "a1", Name: "Alice"}, {ID: "a2", Name: "Bob"}}
	meta := BookMetadata(book, authors, nil, nil)

	contribs, _ := meta["Contributors"].([]string)
	if len(contribs) != 2 || contribs[0] != "Alice" {
		t.Errorf("Contributors = %v", contribs)
	}
}

func TestBookMetadata_Series(t *testing.T) {
	pos := 3.0
	book := &db.Book{ID: "bk1", Title: "Book Three"}
	series := []db.BookSeriesEntry{
		{Series: db.Series{ID: "s1", Name: "My Series"}, Position: &pos},
	}
	meta := BookMetadata(book, nil, series, nil)

	s, ok := meta["Series"].(map[string]any)
	if !ok {
		t.Fatalf("Series not in metadata")
	}
	if s["Name"] != "My Series" {
		t.Errorf("Series.Name = %v", s["Name"])
	}
	if s["Number"] != 3 {
		t.Errorf("Series.Number = %v, want 3", s["Number"])
	}
}

func TestBookMetadata_Language_Default(t *testing.T) {
	book := &db.Book{ID: "bk1", Title: "No Lang"}
	meta := BookMetadata(book, nil, nil, nil)
	if meta["Language"] != "en" {
		t.Errorf("Language = %v, want en", meta["Language"])
	}
}

func TestBookMetadata_Language_Set(t *testing.T) {
	book := &db.Book{ID: "bk1", Title: "French", Language: strPtr("fr")}
	meta := BookMetadata(book, nil, nil, nil)
	if meta["Language"] != "fr" {
		t.Errorf("Language = %v, want fr", meta["Language"])
	}
}

func TestReadingStateResponse_Defaults(t *testing.T) {
	state := &db.KoboReadingState{
		BookID: "bk1",
		Status: "ReadyToRead",
	}
	resp := ReadingStateResponse(state)
	if resp["EntitlementId"] != "bk1" {
		t.Errorf("EntitlementId = %v", resp["EntitlementId"])
	}
	si, _ := resp["StatusInfo"].(map[string]any)
	if si["Status"] != "ReadyToRead" {
		t.Errorf("StatusInfo.Status = %v", si["Status"])
	}
}

func TestReadingStateResponse_WithProgress(t *testing.T) {
	pct := 42.5
	state := &db.KoboReadingState{
		BookID:      "bk1",
		Status:      "Reading",
		PercentRead: &pct,
	}
	resp := ReadingStateResponse(state)
	cb, _ := resp["CurrentBookmark"].(map[string]any)
	if cb["ProgressPercent"] != 42.5 {
		t.Errorf("ProgressPercent = %v", cb["ProgressPercent"])
	}
}
