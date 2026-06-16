package calibre

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ── parseCalibreDate ──────────────────────────────────────────────────────────

func TestParseCalibreDate_EmptyString(t *testing.T) {
	result := parseCalibreDate("")
	require.True(t, result.IsZero(), "empty string should return zero Time")
}

func TestParseCalibreDate_WhitespaceOnly(t *testing.T) {
	result := parseCalibreDate("   ")
	require.True(t, result.IsZero(), "whitespace-only string should return zero Time")
}

func TestParseCalibreDate_Unparseable(t *testing.T) {
	result := parseCalibreDate("not-a-date")
	require.True(t, result.IsZero(), "unparseable string should return zero Time")
}

func TestParseCalibreDate_SentinelYear(t *testing.T) {
	// Calibre stores "0101-01-01" for books with no publication date.
	result := parseCalibreDate("0101-01-01")
	require.False(t, result.IsZero(), "sentinel date should parse to a non-zero Time")
	require.Equal(t, calibreSentinelYear, result.Year(),
		"sentinel date year should equal calibreSentinelYear (%d)", calibreSentinelYear)
}

func TestParseCalibreDate_AllLayouts(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  time.Time
	}{
		// layout0: "2006-01-02T15:04:05-07:00" — non-zero offset so it is distinct from the literal +00:00 layouts
		{"layout0", "2024-03-15T10:30:00-05:00", time.Date(2024, 3, 15, 15, 30, 0, 0, time.UTC)},
		// layout1: "2006-01-02 15:04:05-07:00"
		{"layout1", "2024-03-15 10:30:00-05:00", time.Date(2024, 3, 15, 15, 30, 0, 0, time.UTC)},
		// layout2: "2006-01-02T15:04:05+00:00"
		{"layout2", "2024-03-15T10:30:00+00:00", time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
		// layout3: "2006-01-02 15:04:05+00:00"
		{"layout3", "2024-03-15 10:30:00+00:00", time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
		{"RFC3339", "2024-03-15T10:30:00Z", time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
		{"layout5", "2024-03-15T10:30:00", time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
		{"layout6", "2024-03-15 10:30:00", time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCalibreDate(tc.input)
			require.False(t, got.IsZero(), "layout %q produced zero Time for %q", tc.name, tc.input)
			require.Equal(t, tc.want.Year(), got.Year())
			require.Equal(t, tc.want.Month(), got.Month())
			require.Equal(t, tc.want.Day(), got.Day())
			require.Equal(t, tc.want.Hour(), got.Hour())
			require.Equal(t, tc.want.Minute(), got.Minute())
		})
	}
}

func TestParseCalibreDate_DateOnlyLayout(t *testing.T) {
	// The "2006-01-02" layout (date-only).
	result := parseCalibreDate("2024-03-15")
	require.False(t, result.IsZero())
	require.Equal(t, 2024, result.Year())
	require.Equal(t, time.March, result.Month())
	require.Equal(t, 15, result.Day())
}

func TestParseCalibreDate_LeadingTrailingWhitespace(t *testing.T) {
	result := parseCalibreDate("  2024-03-15  ")
	require.False(t, result.IsZero(), "whitespace-padded valid date should parse correctly")
	require.Equal(t, 2024, result.Year())
}

func TestParseCalibreDate_ReturnsUTC(t *testing.T) {
	// Dates should always be returned in UTC.
	result := parseCalibreDate("2024-03-15T10:30:00+05:00")
	require.False(t, result.IsZero())
	require.Equal(t, "UTC", result.Location().String())
}

// ── Format helpers ────────────────────────────────────────────────────────────

func TestFormat_FileName(t *testing.T) {
	cases := []struct {
		name       string
		format     Format
		wantSuffix string
	}{
		{
			name:       "EPUB uppercase format code",
			format:     Format{FormatCode: "EPUB", Name: "My Book"},
			wantSuffix: "My Book.epub",
		},
		{
			name:       "PDF format code",
			format:     Format{FormatCode: "PDF", Name: "Report"},
			wantSuffix: "Report.pdf",
		},
		{
			name:       "already lowercase format code",
			format:     Format{FormatCode: "mobi", Name: "Story"},
			wantSuffix: "Story.mobi",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantSuffix, tc.format.FileName())
		})
	}
}

func TestCollectBookMap_CollectsRows(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	_, err = sqlDB.Exec(`
		CREATE TABLE t (book INTEGER NOT NULL, val TEXT NOT NULL);
		INSERT INTO t (book, val) VALUES (1, 'a'), (1, 'b'), (2, 'c');
	`)
	require.NoError(t, err)

	got, err := collectBookMap(t.Context(), sqlDB, `SELECT book, val FROM t ORDER BY rowid`,
		func(r *sql.Rows) (int64, string, error) {
			var bookID int64
			var value string
			if err := r.Scan(&bookID, &value); err != nil {
				return 0, "", err
			}
			return bookID, value, nil
		},
		func(result map[int64][]string, bookID int64, value string) {
			result[bookID] = append(result[bookID], value)
		},
	)
	require.NoError(t, err)
	require.Equal(t, map[int64][]string{
		1: {"a", "b"},
		2: {"c"},
	}, got)
}

func TestLoadBooks_LoadsRelatedMetadata(t *testing.T) {
	cdb := newTestCalibreDB(t)

	bookID := insertCalibreBook(t, cdb, "Dune", "Frank Herbert/Dune (1)", "1965-08-01 00:00:00+00:00", 1.5)
	secondBookID := insertCalibreBook(t, cdb, "Neuromancer", "William Gibson/Neuromancer (1)", "", 1.0)

	authorAID := insertCalibreAuthor(t, cdb, "Frank Herbert")
	authorBID := insertCalibreAuthor(t, cdb, "Brian Herbert")
	linkBookAuthor(t, cdb, bookID, authorAID)
	linkBookAuthor(t, cdb, bookID, authorBID)

	mainSeriesID := insertCalibreSeries(t, cdb, "Dune Saga")
	sideSeriesID := insertCalibreSeries(t, cdb, "Sci-Fi Classics")
	linkBookSeries(t, cdb, bookID, mainSeriesID)
	linkBookSeries(t, cdb, bookID, sideSeriesID)

	firstPublisherID := insertCalibrePublisher(t, cdb, "Chilton Books")
	secondPublisherID := insertCalibrePublisher(t, cdb, "Ace")
	linkBookPublisher(t, cdb, bookID, firstPublisherID)
	linkBookPublisher(t, cdb, bookID, secondPublisherID)

	insertCalibreComment(t, cdb, bookID, "A science fiction masterpiece.")
	insertCalibreFormat(t, cdb, bookID, "EPUB", "Dune", 512000)
	insertCalibreFormat(t, cdb, bookID, "PDF", "Dune", 1024000)
	insertCalibreIdentifier(t, cdb, bookID, "isbn13", "9780441013593")
	insertCalibreIdentifier(t, cdb, bookID, "goodreads", "234225")

	insertCalibreLanguage(t, cdb, bookID, "eng")

	res, err := cdb.db.ExecContext(t.Context(), `INSERT INTO languages (lang_code) VALUES (?)`, "fra")
	require.NoError(t, err)

	secondLangID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = cdb.db.ExecContext(
		t.Context(),
		`INSERT INTO books_languages_link (book, lang_code, item_order) VALUES (?, ?, ?)`,
		bookID,
		secondLangID,
		1,
	)
	require.NoError(t, err)

	insertCalibreComment(t, cdb, secondBookID, "Cyberpunk classic.")

	books, err := cdb.LoadBooks(t.Context())
	require.NoError(t, err)
	require.Len(t, books, 2)

	require.Equal(t, []Book{
		{
			CalibreID:   bookID,
			Title:       "Dune",
			Path:        "Frank Herbert/Dune (1)",
			SeriesIndex: 1.5,
			Pubdate:     time.Date(1965, time.August, 1, 0, 0, 0, 0, time.UTC),
			Authors:     []string{"Frank Herbert", "Brian Herbert"},
			Series: []SeriesEntry{
				{Name: "Dune Saga", Position: 1.5},
				{Name: "Sci-Fi Classics", Position: 1.5},
			},
			Publisher:   "Chilton Books",
			Description: "A science fiction masterpiece.",
			Formats: []Format{
				{FormatCode: "EPUB", Name: "Dune", UncompressedSize: 512000},
				{FormatCode: "PDF", Name: "Dune", UncompressedSize: 1024000},
			},
			Identifiers: map[string]string{
				"goodreads": "234225",
				"isbn13":    "9780441013593",
			},
			Language: "eng",
		},
		{
			CalibreID:   secondBookID,
			Title:       "Neuromancer",
			Path:        "William Gibson/Neuromancer (1)",
			SeriesIndex: 1.0,
			Description: "Cyberpunk classic.",
		},
	}, books)
}

func TestLoadBookLanguages_MissingTables(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	cdb := &DB{db: sqlDB}

	languages, err := cdb.loadBookLanguages(t.Context())
	require.NoError(t, err)
	require.Empty(t, languages)
}
