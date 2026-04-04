package kobo

import (
	"fmt"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

// DownloadURL represents a Kobo download URL entry.
type DownloadURL struct {
	Format   string `json:"Format"`
	Size     int64  `json:"Size"`
	URL      string `json:"Url"`
	Platform string `json:"Platform"`
}

// ActivePeriod represents the active period of a Kobo entitlement.
type ActivePeriod struct {
	From string `json:"From"`
}

// Entitlement represents the BookEntitlement object expected by Kobo devices.
type Entitlement struct {
	Accessibility       string       `json:"Accessibility"`
	ActivePeriod        ActivePeriod `json:"ActivePeriod"`
	Created             string       `json:"Created"`
	CrossRevisionID     string       `json:"CrossRevisionId"`
	ID                  string       `json:"Id"`
	IsRemoved           bool         `json:"IsRemoved"`
	IsHiddenFromArchive bool         `json:"IsHiddenFromArchive"`
	IsLocked            bool         `json:"IsLocked"`
	LastModified        string       `json:"LastModified"`
	OriginCategory      string       `json:"OriginCategory"`
	RevisionID          string       `json:"RevisionId"`
	Status              string       `json:"Status"`
}

// DisplayPrice represents a display price in a Kobo metadata response.
type DisplayPrice struct {
	CurrencyCode string `json:"CurrencyCode,omitempty"`
	TotalAmount  int    `json:"TotalAmount"`
}

// PublisherInfo represents the publisher object in a Kobo metadata response.
type PublisherInfo struct {
	Imprint string  `json:"Imprint"`
	Name    *string `json:"Name"`
}

// ContributorRole represents a contributor role in a Kobo metadata response.
type ContributorRole struct {
	Name string `json:"Name"`
}

// SeriesInfo represents the series object in a Kobo metadata response.
type SeriesInfo struct {
	Name string `json:"Name"`
	// Number is either an int (when position is a whole number) or a float64
	// to match the Kobo protocol wire format, which expects e.g. 3 not 3.0.
	Number      any      `json:"Number"`
	NumberFloat *float64 `json:"NumberFloat"`
	ID          string   `json:"Id"`
}

// Metadata represents the BookMetadata object expected by Kobo devices.
type Metadata struct {
	Categories              []string          `json:"Categories"`
	CoverImageID            string            `json:"CoverImageId"`
	CrossRevisionID         string            `json:"CrossRevisionId"`
	CurrentDisplayPrice     DisplayPrice      `json:"CurrentDisplayPrice"`
	CurrentLoveDisplayPrice DisplayPrice      `json:"CurrentLoveDisplayPrice"`
	Description             *string           `json:"Description"`
	DownloadUrls            []DownloadURL     `json:"DownloadUrls"`
	EntitlementID           string            `json:"EntitlementId"`
	ExternalIds             []any             `json:"ExternalIds"`
	Genre                   string            `json:"Genre"`
	IsEligibleForKoboLove   bool              `json:"IsEligibleForKoboLove"`
	IsInternetArchive       bool              `json:"IsInternetArchive"`
	IsPreOrder              bool              `json:"IsPreOrder"`
	IsSocialEnabled         bool              `json:"IsSocialEnabled"`
	Language                string            `json:"Language"`
	PhoneticPronunciations  map[string]any    `json:"PhoneticPronunciations"`
	PublicationDate         string            `json:"PublicationDate"`
	Publisher               PublisherInfo     `json:"Publisher"`
	RevisionID              string            `json:"RevisionId"`
	Title                   string            `json:"Title"`
	WorkId                  string            `json:"WorkId"`
	ContributorRoles        []ContributorRole `json:"ContributorRoles"`
	Contributors            []string          `json:"Contributors"`
	Series                  *SeriesInfo       `json:"Series,omitempty"`
}

// Location represents a reading location within a Kobo book.
type Location struct {
	Value  string `json:"Value"`
	Type   string `json:"Type"`
	Source string `json:"Source"`
}

// Bookmark represents the current bookmark in a Kobo reading state.
type Bookmark struct {
	LastModified                 string    `json:"LastModified"`
	ProgressPercent              *float64  `json:"ProgressPercent,omitempty"`
	ContentSourceProgressPercent *float64  `json:"ContentSourceProgressPercent,omitempty"`
	Location                     *Location `json:"Location,omitempty"`
}

// ReadingStatistics represents reading statistics in a Kobo reading state.
type ReadingStatistics struct {
	LastModified string `json:"LastModified"`
}

// StatusInfo represents reading status information in a Kobo reading state.
type StatusInfo struct {
	LastModified        string `json:"LastModified"`
	Status              string `json:"Status"`
	TimesStartedReading int    `json:"TimesStartedReading"`
}

// ReadingState represents the reading-state object expected by Kobo devices.
type ReadingState struct {
	EntitlementId     string            `json:"EntitlementId"`
	Created           string            `json:"Created"`
	LastModified      string            `json:"LastModified"`
	PriorityTimestamp string            `json:"PriorityTimestamp"`
	StatusInfo        StatusInfo        `json:"StatusInfo"`
	Statistics        ReadingStatistics `json:"Statistics"`
	CurrentBookmark   Bookmark          `json:"CurrentBookmark"`
}

// DownloadURLs builds the slice of Kobo download-URL objects for bookID given
// its associated files. base is the scheme+host prefix (e.g.
// "https://example.com") and tokenValue is the Kobo device token.
func DownloadURLs(base, tokenValue, bookID string, files []db.BookFile) []DownloadURL {
	var urls []DownloadURL
	for _, f := range files {
		koboFmt, ok := FormatForFileType(f.FileType)
		if !ok {
			continue
		}
		urls = append(urls, DownloadURL{
			Format:   koboFmt,
			Size:     f.FileSize,
			Url:      fmt.Sprintf("%s/kobo/%s/download/%s/%s", base, tokenValue, bookID, strings.ToLower(f.FileType)),
			Platform: "Generic",
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
func BookEntitlement(book *db.Book) *Entitlement {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Entitlement{
		Accessibility:       "Full",
		ActivePeriod:        ActivePeriod{From: now},
		Created:             book.CreatedAt.UTC().Format(time.RFC3339),
		CrossRevisionId:     book.ID,
		Id:                  book.ID,
		IsRemoved:           false,
		IsHiddenFromArchive: false,
		IsLocked:            false,
		LastModified:        book.UpdatedAt.UTC().Format(time.RFC3339),
		OriginCategory:      "Imported",
		RevisionId:          book.ID,
		Status:              "Active",
	}
}

// BookMetadata builds the BookMetadata object expected by Kobo devices.
func BookMetadata(book *db.Book, authors []db.Author, series []db.BookSeriesEntry, downloadURLs []DownloadURL) *Metadata {
	var contributorRoles []ContributorRole
	var contributors []string
	for _, a := range authors {
		contributorRoles = append(contributorRoles, ContributorRole{Name: a.Name})
		contributors = append(contributors, a.Name)
	}

	m := &Metadata{
		Categories:              []string{"00000000-0000-0000-0000-000000000001"},
		CoverImageId:            book.ID,
		CrossRevisionId:         book.ID,
		CurrentDisplayPrice:     DisplayPrice{CurrencyCode: "USD", TotalAmount: 0},
		CurrentLoveDisplayPrice: DisplayPrice{TotalAmount: 0},
		Description:             book.Description,
		DownloadUrls:            downloadURLs,
		EntitlementId:           book.ID,
		ExternalIds:             []any{},
		Genre:                   "00000000-0000-0000-0000-000000000001",
		IsEligibleForKoboLove:   false,
		IsInternetArchive:       false,
		IsPreOrder:              false,
		IsSocialEnabled:         true,
		Language:                language(book),
		PhoneticPronunciations:  map[string]any{},
		PublicationDate:         pubDate(book),
		Publisher:               PublisherInfo{Imprint: "", Name: book.Publisher},
		RevisionId:              book.ID,
		Title:                   book.Title,
		WorkId:                  book.ID,
		ContributorRoles:        contributorRoles,
		Contributors:            contributors,
	}

	if len(series) > 0 {
		s := series[0]
		m.Series = &SeriesInfo{
			Name:        s.Series.Name,
			Number:      seriesNumber(s.Position),
			NumberFloat: s.Position,
			Id:          s.Series.ID,
		}
	}

	return m
}

// ReadingStateResponse builds the reading-state object expected by Kobo devices.
func ReadingStateResponse(state *db.KoboReadingState) *ReadingState {
	now := time.Now().UTC().Format(time.RFC3339)

	updatedAt := now
	if !state.UpdatedAt.IsZero() {
		updatedAt = state.UpdatedAt.UTC().Format(time.RFC3339)
	}
	createdAt := updatedAt
	if !state.CreatedAt.IsZero() {
		createdAt = state.CreatedAt.UTC().Format(time.RFC3339)
	}

	bookmark := Bookmark{
		LastModified: updatedAt,
	}
	if state.PercentRead != nil {
		bookmark.ProgressPercent = state.PercentRead
		bookmark.ContentSourceProgressPercent = state.PercentRead
	}
	if state.LocationValue != nil && state.LocationType != nil && state.LocationSource != nil {
		bookmark.Location = &Location{
			Value:  *state.LocationValue,
			Type:   *state.LocationType,
			Source: *state.LocationSource,
		}
	}

	return &ReadingState{
		EntitlementId:     state.BookID,
		Created:           createdAt,
		LastModified:      updatedAt,
		PriorityTimestamp: updatedAt,
		StatusInfo: StatusInfo{
			LastModified:        updatedAt,
			Status:              state.Status,
			TimesStartedReading: 0,
		},
		Statistics:      ReadingStatistics{LastModified: updatedAt},
		CurrentBookmark: bookmark,
	}
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
