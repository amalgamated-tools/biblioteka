package opds

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaginationLinks_FirstPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 100, 50, NavContentType)
	if len(links) != 2 {
		require.Failf(t, "failed", "len = %d, want 2 (self + next)", len(links))
	}
	if links[0].Rel != RelSelf {
		t.Errorf("links[0].Rel = %q, want %q", links[0].Rel, RelSelf)
	}
	if links[1].Rel != RelNext {
		t.Errorf("links[1].Rel = %q, want %q", links[1].Rel, RelNext)
	}
}

func TestPaginationLinks_LastPage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 2, 100, 50, NavContentType)
	if len(links) != 2 {
		require.Failf(t, "failed", "len = %d, want 2 (self + previous)", len(links))
	}
	if links[0].Rel != RelSelf {
		t.Errorf("links[0].Rel = %q, want self", links[0].Rel)
	}
	if links[1].Rel != RelPrevious {
		t.Errorf("links[1].Rel = %q, want previous", links[1].Rel)
	}
}

func TestPaginationLinks_MiddlePage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 2, 150, 50, AcqContentType)
	if len(links) != 3 {
		require.Failf(t, "failed", "len = %d, want 3 (self + previous + next)", len(links))
	}
}

func TestPaginationLinks_SinglePage(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/all", 1, 10, 50, AcqContentType)
	if len(links) != 1 {
		require.Failf(t, "failed", "len = %d, want 1 (self only)", len(links))
	}
}

func TestPaginationLinks_URLWithExistingQuery(t *testing.T) {
	links := PaginationLinks("https://example.com/opds/search?q=foo", 1, 100, 50, AcqContentType)
	if len(links) < 1 {
		require.Fail(t, "expected at least one link")
	}
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
