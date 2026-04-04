package kobo

import (
	"fmt"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// DownloadURLs builds the slice of Kobo download-URL objects for bookID given
// its associated files. base is the scheme+host prefix (e.g.
// "https://example.com") and tokenValue is the Kobo device token.
func DownloadURLs(base, tokenValue, bookID string, files []db.BookFile) []map[string]any {
	var urls []map[string]any
	for _, f := range files {
		koboFmt, ok := FormatForFileType(f.FileType)
		if !ok {
			continue
		}
		urls = append(urls, map[string]any{
			"Format":   koboFmt,
			"Size":     f.FileSize,
			"Url":      fmt.Sprintf("%s/kobo/%s/download/%s/%s", base, tokenValue, bookID, strings.ToLower(f.FileType)),
			"Platform": "Generic",
		})
	}
	return urls
}

// FormatForFileType maps a Biblioteka file_type value to the Kobo format
// identifier. Returns the identifier and true when the format is supported by
// Kobo; returns "", false otherwise.
func FormatForFileType(fileType string) (string, bool) {
	switch strings.ToLower(fileType) {
	case "epub":
		return "EPUB3", true
	case "kepub":
		return "KEPUB", true
	case "mobi":
		return "MOBI", true
	case "pdf":
		return "PDF", true
	case "azw3":
		return "AZW3", true
	default:
		return "", false
	}
}

// BookEntitlement builds the BookEntitlement object expected by Kobo devices.
func BookEntitlement(book *db.Book) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"Accessibility":       "Full",
		"ActivePeriod":        map[string]any{"From": now},
		"Created":             book.CreatedAt.UTC().Format(time.RFC3339),
		"CrossRevisionId":     book.ID,
		"Id":                  book.ID,
		"IsRemoved":           false,
		"IsHiddenFromArchive": false,
		"IsLocked":            false,
		"LastModified":        book.UpdatedAt.UTC().Format(time.RFC3339),
		"OriginCategory":      "Imported",
		"RevisionId":          book.ID,
		"Status":              "Active",
	}
}

// BookMetadata builds the BookMetadata object expected by Kobo devices.
func BookMetadata(book *db.Book, authors []db.Author, series []db.BookSeriesEntry, downloadURLs []map[string]any) map[string]any {
	var contributorRoles []map[string]any
	var contributors []string
	for _, a := range authors {
		contributorRoles = append(contributorRoles, map[string]any{"Name": a.Name})
		contributors = append(contributors, a.Name)
	}

	metadata := map[string]any{
		"Categories":              []string{"00000000-0000-0000-0000-000000000001"},
		"CoverImageId":            book.ID,
		"CrossRevisionId":         book.ID,
		"CurrentDisplayPrice":     map[string]any{"CurrencyCode": "USD", "TotalAmount": 0},
		"CurrentLoveDisplayPrice": map[string]any{"TotalAmount": 0},
		"Description":             ptrStr(book.Description),
		"DownloadUrls":            downloadURLs,
		"EntitlementId":           book.ID,
		"ExternalIds":             []any{},
		"Genre":                   "00000000-0000-0000-0000-000000000001",
		"IsEligibleForKoboLove":   false,
		"IsInternetArchive":       false,
		"IsPreOrder":              false,
		"IsSocialEnabled":         true,
		"Language":                language(book),
		"PhoneticPronunciations":  map[string]any{},
		"PublicationDate":         pubDate(book),
		"Publisher":               map[string]any{"Imprint": "", "Name": ptrStr(book.Publisher)},
		"RevisionId":              book.ID,
		"Title":                   book.Title,
		"WorkId":                  book.ID,
		"ContributorRoles":        contributorRoles,
		"Contributors":            contributors,
	}

	if len(series) > 0 {
		s := series[0]
		metadata["Series"] = map[string]any{
			"Name":        s.Series.Name,
			"Number":      seriesNumber(s.Position),
			"NumberFloat": s.Position,
			"Id":          s.Series.ID,
		}
	}

	return metadata
}

// ReadingStateResponse builds the reading-state object expected by Kobo devices.
func ReadingStateResponse(state *db.KoboReadingState) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)

	updatedAt := now
	if !state.UpdatedAt.IsZero() {
		updatedAt = state.UpdatedAt.UTC().Format(time.RFC3339)
	}
	createdAt := updatedAt
	if !state.CreatedAt.IsZero() {
		createdAt = state.CreatedAt.UTC().Format(time.RFC3339)
	}

	statusInfo := map[string]any{
		"LastModified":        updatedAt,
		"Status":              state.Status,
		"TimesStartedReading": 0,
	}

	currentBookmark := map[string]any{
		"LastModified": updatedAt,
	}
	if state.PercentRead != nil {
		currentBookmark["ProgressPercent"] = *state.PercentRead
		currentBookmark["ContentSourceProgressPercent"] = *state.PercentRead
	}
	if state.LocationValue != nil && state.LocationType != nil && state.LocationSource != nil {
		currentBookmark["Location"] = map[string]any{
			"Value":  *state.LocationValue,
			"Type":   *state.LocationType,
			"Source": *state.LocationSource,
		}
	}

	return map[string]any{
		"EntitlementId":     state.BookID,
		"Created":           createdAt,
		"LastModified":      updatedAt,
		"PriorityTimestamp": updatedAt,
		"StatusInfo":        statusInfo,
		"Statistics":        map[string]any{"LastModified": updatedAt},
		"CurrentBookmark":   currentBookmark,
	}
}

func ptrStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func language(book *db.Book) string {
	if book.Language != nil && *book.Language != "" {
		return *book.Language
	}
	return "en"
}

func pubDate(book *db.Book) string {
	if book.PublicationDate != nil && *book.PublicationDate != "" {
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006"} {
			if t, err := time.Parse(layout, *book.PublicationDate); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
		return *book.PublicationDate
	}
	return time.Time{}.UTC().Format(time.RFC3339)
}

func seriesNumber(pos *float64) any {
	if pos == nil {
		return 1
	}
	if *pos == float64(int(*pos)) {
		return int(*pos)
	}
	return *pos
}
