package opds

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaginationLinks_FirstPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 100, 50, NavContentType)
	require.Len(t, links, 2)
	if links[0].Rel != RelSelf {
		t.Errorf("links[0].Rel = %q, want %q", links[0].Rel, RelSelf)
	}
	if links[1].Rel != RelNext {
		t.Errorf("links[1].Rel = %q, want %q", links[1].Rel, RelNext)
	}
}

func TestPaginationLinks_LastPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 2, 100, 50, NavContentType)
	require.Len(t, links, 2)
	if links[0].Rel != RelSelf {
		t.Errorf("links[0].Rel = %q, want self", links[0].Rel)
	}
	if links[1].Rel != RelPrevious {
		t.Errorf("links[1].Rel = %q, want previous", links[1].Rel)
	}
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
	if got := links[0].Href; got != "https://example.com/opds/search?q=foo&page=1" {
		t.Errorf("Href = %q", got)
	}
}

func TestPaginationLinks_ContentType(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 100, 50, NavContentType)
	for _, l := range links {
		if l.Type != NavContentType {
			t.Errorf("link %q Type = %q, want %q", l.Rel, l.Type, NavContentType)
		}
	}
}
