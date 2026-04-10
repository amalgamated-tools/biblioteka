package opds

import (
	"strconv"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/filetype"
)

// MIMETypeForFileType returns the MIME type for a given file type string.
// If the file type is not recognized, it returns an empty string.
// This delegates to the shared filetype package which is the single source of
// truth for Biblioteka file-type-to-MIME mappings.
func MIMETypeForFileType(fileType string) string {
	return filetype.MIMEType(fileType)
}

// PaginationLinks builds the self/previous/next link set for paginated OPDS
// feeds. selfURL is the base URL without a page parameter; page is the current
// 1-based page number; total is the total number of entries; pageSize is the
// page capacity; contentType is used for all generated links.
func PaginationLinks(selfURL string, page, total, pageSize int, contentType string) []Link {
	// For URLs that already have query params (like search), use & instead of ?.
	sep := "?"
	if strings.Contains(selfURL, "?") {
		sep = "&"
	}

	links := []Link{
		{Rel: RelSelf, Href: selfURL + sep + "page=" + strconv.Itoa(page), Type: contentType},
	}

	if page > 1 {
		links = append(links, Link{
			Rel:  RelPrevious,
			Href: selfURL + sep + "page=" + strconv.Itoa(page-1),
			Type: contentType,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if page < totalPages {
		links = append(links, Link{
			Rel:  RelNext,
			Href: selfURL + sep + "page=" + strconv.Itoa(page+1),
			Type: contentType,
		})
	}

	return links
}
