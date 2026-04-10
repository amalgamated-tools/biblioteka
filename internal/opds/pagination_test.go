package opds

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaginationLinks_FirstPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 100, 50, NavContentType)
	require.Len(t, links, 2)
	require.Equal(t, RelSelf, links[0].Rel)
	require.Equal(t, RelNext, links[1].Rel)
}

func TestPaginationLinks_LastPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 2, 100, 50, NavContentType)
	require.Len(t, links, 2)
	require.Equal(t, RelSelf, links[0].Rel)
	require.Equal(t, RelPrevious, links[1].Rel)
}

func TestPaginationLinks_MiddlePage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 2, 150, 50, AcqContentType)
	require.Len(t, links, 3)
}

func TestPaginationLinks_SinglePage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 10, 50, AcqContentType)
	require.Len(t, links, 1)
}

func TestPaginationLinks_URLWithExistingQuery(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/search?q=foo", 1, 100, 50, AcqContentType)
	require.NotEmpty(t, links)
	// separator must be & not ? since the URL already has a query string
	got := links[0].Href
	require.Equal(t, "https://example.com/opds/search?q=foo&page=1", got)
}

func TestPaginationLinks_ContentType(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 100, 50, NavContentType)
	for _, l := range links {
		require.Equal(t, NavContentType, l.Type)
	}
}
