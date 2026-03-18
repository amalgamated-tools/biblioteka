package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleBookMetadata handles GET /v1/library/{uuid}/metadata.
func (h *KoboHandler) HandleBookMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeKoboJSON(w, http.StatusOK, []any{})
		return
	}
	tokenValue := auth.KoboTokenFromContext(r.Context())
	bookID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/library/"), "/metadata")
	if bookID == "" {
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch book for kobo metadata", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	authors, err := h.DB.GetBookAuthors(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch authors for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	files, err := h.DB.ListBookFiles(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch files for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	series, err := h.DB.GetBookSeries(r.Context(), bookID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to fetch series for kobo metadata",
			slog.String(otelkeys.ID, bookID),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	base := schemeAndHost(r)
	downloadURLs := koboDownloadURLs(base, tokenValue, bookID, files)
	writeKoboJSON(w, http.StatusOK, []any{koboBookMetadata(book, authors, series, downloadURLs)})
}

func koboDownloadURLs(base, tokenValue, bookID string, files []db.BookFile) []map[string]any {
	var urls []map[string]any
	for _, f := range files {
		koboFmt, ok := koboFormatForFileType(f.FileType)
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

// koboFormatForFileType maps our file_type values to Kobo format identifiers.
func koboFormatForFileType(fileType string) (string, bool) {
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

func koboBookEntitlement(book *db.Book) map[string]any {
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

func koboBookMetadata(book *db.Book, authors []db.Author, series []db.BookSeriesEntry, downloadURLs []map[string]any) map[string]any {
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
		"Description":             koboPtrStr(book.Description),
		"DownloadUrls":            downloadURLs,
		"EntitlementId":           book.ID,
		"ExternalIds":             []any{},
		"Genre":                   "00000000-0000-0000-0000-000000000001",
		"IsEligibleForKoboLove":   false,
		"IsInternetArchive":       false,
		"IsPreOrder":              false,
		"IsSocialEnabled":         true,
		"Language":                koboLanguage(book),
		"PhoneticPronunciations":  map[string]any{},
		"PublicationDate":         koboPubDate(book),
		"Publisher":               map[string]any{"Imprint": "", "Name": koboPtrStr(book.Publisher)},
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
			"Number":      koboSeriesNumber(s.Position),
			"NumberFloat": s.Position,
			"Id":          s.Series.ID,
		}
	}

	return metadata
}

func koboPtrStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func koboLanguage(book *db.Book) string {
	if book.Language != nil && *book.Language != "" {
		return *book.Language
	}
	return "en"
}

func koboPubDate(book *db.Book) string {
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

func koboSeriesNumber(pos *float64) any {
	if pos == nil {
		return 1
	}
	if *pos == float64(int(*pos)) {
		return int(*pos)
	}
	return *pos
}
