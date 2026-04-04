package handlers

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/kobo"

	"github.com/stretchr/testify/require"
)

// ---- Kobo format mapping ----

func TestKoboFormatForFileType(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{"epub", "EPUB3", true},
		{"EPUB", "EPUB3", true},
		{"kepub", "KEPUB", true},
		{"mobi", "MOBI", true},
		{"pdf", "PDF", true},
		{"azw3", "AZW3", true},
		{"txt", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := kobo.FormatForFileType(tc.input)
		if ok != tc.ok || got != tc.want {
			t.Errorf("kobo.FormatForFileType(%q) = (%q, %v), want (%q, %v)",
				tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

// ---- kobo.EncodeSyncToken produces valid base64 JSON ----

func TestEncodeKoboSyncToken_IsValidBase64JSON(t *testing.T) {
	tok := kobo.SyncToken{BooksLastModified: time.Now()}
	encoded := kobo.EncodeSyncToken(tok)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err, "not valid base64")
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		require.NoError(t, err, "not valid JSON")
	}
	if payload["version"] != "1-0-0" {
		t.Errorf("version = %v, want 1-0-0", payload["version"])
	}
}

// ---- koboRandomUUID ----

func TestKoboRandomUUID_Format(t *testing.T) {
	uuid, err := koboRandomUUID()
	require.NoError(t, err, "koboRandomUUID")
	// UUID v4 format: 8-4-4-4-12 hex chars
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !re.MatchString(uuid) {
		t.Errorf("UUID %q does not match v4 pattern", uuid)
	}
}

// ---- kobo.BookMetadata with series ----

func TestKoboBookMetadata_WithSeries(t *testing.T) {
	h, _ := setupKoboHandler(t)

	seriesName := "The Dark Tower"
	s, err := h.DB.CreateSeries(t.Context(), seriesName, nil, nil, nil)
	require.NoError(t, err, "create series")

	book, err := h.DB.CreateBook(t.Context(), "The Gunslinger", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err, "create book")

	pos := 1.0
	series := []db.BookSeriesEntry{{Series: *s, Position: &pos}}
	meta := kobo.BookMetadata(book, nil, series, nil)

	seriesMeta := meta.Series
	if seriesMeta == nil {
		require.Fail(t, "expected Series in metadata")
	}
	if seriesMeta.Name != seriesName {
		t.Errorf("Series.Name = %v, want %q", seriesMeta.Name, seriesName)
	}
	if seriesMeta.Number != int(1) {
		t.Errorf("Series.Number = %v, want 1", seriesMeta.Number)
	}
}

// ---- kobo.SyncToken with BooksLastID ----

func TestKoboSyncTokenRoundTrip_WithBooksLastID(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := kobo.SyncToken{
		BooksLastModified: now,
		BooksLastID:       "some-book-id",
	}
	encoded := kobo.EncodeSyncToken(tok)
	decoded := kobo.ParseSyncToken(encoded)
	if decoded.BooksLastID != tok.BooksLastID {
		t.Errorf("BooksLastID: got %q, want %q", decoded.BooksLastID, tok.BooksLastID)
	}
}
