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
		require.Equal(t, tc.ok, ok, "kobo.FormatForFileType(%q) ok", tc.input)
		require.Equal(t, tc.want, got, "kobo.FormatForFileType(%q)", tc.input)
	}
}

// ---- kobo.EncodeSyncToken produces valid base64 JSON ----

func TestEncodeKoboSyncToken_IsValidBase64JSON(t *testing.T) {
	tok := kobo.SyncToken{BooksLastModified: time.Now()}
	encoded := kobo.EncodeSyncToken(tok)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err, "not valid base64")
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload), "not valid JSON")
	require.Equal(t, "1-0-0", payload["version"])
}

// ---- koboRandomUUID ----

func TestKoboRandomUUID_Format(t *testing.T) {
	uuid, err := koboRandomUUID()
	require.NoError(t, err, "koboRandomUUID")
	// UUID v4 format: 8-4-4-4-12 hex chars
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	require.True(t, re.MatchString(uuid), "UUID %q does not match v4 pattern", uuid)
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
	require.NotNil(t, seriesMeta)
	require.Equal(t, seriesName, seriesMeta.Name)
	require.Equal(t, int(1), seriesMeta.Number)
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
	require.Equal(t, tok.BooksLastID, decoded.BooksLastID)
}
