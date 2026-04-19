package calibre

import (
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
	// Each layout from calibreDateLayouts should produce the same UTC instant.
	want := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		name  string
		input string
	}{
		{"layout0", "2024-03-15T10:30:00+00:00"},
		{"layout1", "2024-03-15 10:30:00+00:00"},
		{"layout2", "2024-03-15T10:30:00+00:00"},
		{"layout3", "2024-03-15 10:30:00+00:00"},
		{"RFC3339", "2024-03-15T10:30:00Z"},
		{"layout5", "2024-03-15T10:30:00"},
		{"layout6", "2024-03-15 10:30:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseCalibreDate(tc.input)
			require.False(t, got.IsZero(), "layout %q produced zero Time for %q", tc.name, tc.input)
			require.Equal(t, want.Year(), got.Year())
			require.Equal(t, want.Month(), got.Month())
			require.Equal(t, want.Day(), got.Day())
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
