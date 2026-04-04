package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
)

// --- Pagination ---

func TestAllBooks_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	// Create enough books to have a second page (opdspkg.PageSize is 50).
	for i := range 55 {
		if _, err := h.DB.CreateBook(ctx, "Book "+padInt(i), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
			t.Fatalf("create book %d: %v", i, err)
		}
	}

	// Page 1: should have "next" link but no "previous" link.
	r := httptest.NewRequest(http.MethodGet, "/opds/all?page=1", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != 50 {
		t.Errorf("page 1: entries = %d, want 50", len(feed.Entries))
	}
	if findLink(feed.Links, opdspkg.RelNext) == nil {
		t.Error("page 1: missing next link")
	}
	if findLink(feed.Links, opdspkg.RelPrevious) != nil {
		t.Error("page 1: should not have previous link")
	}

	// Page 2: should have "previous" link but no "next" link.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/all?page=2", nil)
	w2 := httptest.NewRecorder()
	h.HandleOPDS(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: status = %d, want %d", w2.Code, http.StatusOK)
	}

	feed2 := parseOPDSFeed(t, w2.Body.Bytes())
	if len(feed2.Entries) != 5 {
		t.Errorf("page 2: entries = %d, want 5", len(feed2.Entries))
	}
	if findLink(feed2.Links, opdspkg.RelPrevious) == nil {
		t.Error("page 2: missing previous link")
	}
	if findLink(feed2.Links, opdspkg.RelNext) != nil {
		t.Error("page 2: should not have next link")
	}
}

// --- X-Forwarded-Proto ---

func TestBaseURL_XForwardedProto(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, opdspkg.RelSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	if !strings.HasPrefix(selfLink.Href, "https://") {
		t.Errorf("self link = %q, want https:// prefix", selfLink.Href)
	}
}

func TestBaseURL_InvalidXForwardedProto(t *testing.T) {
	h := setupOPDSHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/opds", nil)
	r.Header.Set("X-Forwarded-Proto", "javascript:")
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	feed := parseOPDSFeed(t, w.Body.Bytes())
	selfLink := findLink(feed.Links, opdspkg.RelSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	// Should fallback to http, not use the injected value.
	if strings.HasPrefix(selfLink.Href, "javascript:") {
		t.Errorf("self link = %q, should not use injected proto", selfLink.Href)
	}
}

// --- Helper unit tests ---

func TestParsePage(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{"", 1},
		{"?page=1", 1},
		{"?page=3", 3},
		{"?page=0", 1},
		{"?page=-1", 1},
		{"?page=abc", 1},
	}

	for _, tt := range tests {
		r := httptest.NewRequest(http.MethodGet, "/opds/all"+tt.query, nil)
		got := parsePage(r)
		if got != tt.want {
			t.Errorf("parsePage(%q) = %d, want %d", tt.query, got, tt.want)
		}
	}
}

func TestPaginationLinks(t *testing.T) {
	// Single page: no next or previous.
	links := opdspkg.PaginationLinks("/opds/all", 1, 10, 50, opdspkg.AcqContentType)
	if findLink(links, opdspkg.RelNext) != nil {
		t.Error("single page: should not have next link")
	}
	if findLink(links, opdspkg.RelPrevious) != nil {
		t.Error("single page: should not have previous link")
	}

	// First of multiple pages: next but no previous.
	links = opdspkg.PaginationLinks("/opds/all", 1, 100, 50, opdspkg.AcqContentType)
	if findLink(links, opdspkg.RelNext) == nil {
		t.Error("first page: should have next link")
	}
	if findLink(links, opdspkg.RelPrevious) != nil {
		t.Error("first page: should not have previous link")
	}

	// Middle page: both next and previous.
	links = opdspkg.PaginationLinks("/opds/all", 2, 150, 50, opdspkg.AcqContentType)
	if findLink(links, opdspkg.RelNext) == nil {
		t.Error("middle page: should have next link")
	}
	if findLink(links, opdspkg.RelPrevious) == nil {
		t.Error("middle page: should have previous link")
	}

	// Last page: previous but no next.
	links = opdspkg.PaginationLinks("/opds/all", 2, 100, 50, opdspkg.AcqContentType)
	if findLink(links, opdspkg.RelNext) != nil {
		t.Error("last page: should not have next link")
	}
	if findLink(links, opdspkg.RelPrevious) == nil {
		t.Error("last page: should have previous link")
	}
}

func TestPaginationLinks_SearchURL(t *testing.T) {
	// URLs with existing query params should use "&" not "?" for page param.
	links := opdspkg.PaginationLinks("/opds/search?q=test", 1, 100, 50, opdspkg.AcqContentType)
	selfLink := findLink(links, opdspkg.RelSelf)
	if selfLink == nil {
		t.Fatal("missing self link")
	}
	if strings.Contains(selfLink.Href, "?q=test?page=") {
		t.Errorf("self link has double '?': %q", selfLink.Href)
	}
	if !strings.Contains(selfLink.Href, "&page=") {
		t.Errorf("self link should use '&' for page param: %q", selfLink.Href)
	}
}

// padInt zero-pads an integer to 3 digits for consistent sorting.
func padInt(n int) string {
	return fmt.Sprintf("%03d", n)
}

// --- Authors/Series pagination ---

func TestAuthorsFeed_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	const totalAuthors = opdspkg.PageSize + 5
	for i := range totalAuthors {
		if _, err := h.DB.CreateAuthor(ctx, fmt.Sprintf("Author %03d", i), nil, nil, nil, nil); err != nil {
			t.Fatalf("create author %d: %v", i, err)
		}
	}

	// Page 1: should have next link but no previous.
	r := httptest.NewRequest(http.MethodGet, "/opds/authors?page=1", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: status = %d, want %d", w.Code, http.StatusOK)
	}
	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != opdspkg.PageSize {
		t.Errorf("page 1: entries = %d, want %d", len(feed.Entries), opdspkg.PageSize)
	}
	if findLink(feed.Links, opdspkg.RelNext) == nil {
		t.Error("page 1: missing next link")
	}
	if findLink(feed.Links, opdspkg.RelPrevious) != nil {
		t.Error("page 1: should not have previous link")
	}

	// Page 2: should have previous link but no next.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/authors?page=2", nil)
	w2 := httptest.NewRecorder()
	h.HandleOPDS(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: status = %d, want %d", w2.Code, http.StatusOK)
	}
	feed2 := parseOPDSFeed(t, w2.Body.Bytes())
	if len(feed2.Entries) != totalAuthors-opdspkg.PageSize {
		t.Errorf("page 2: entries = %d, want %d", len(feed2.Entries), totalAuthors-opdspkg.PageSize)
	}
	if findLink(feed2.Links, opdspkg.RelPrevious) == nil {
		t.Error("page 2: missing previous link")
	}
	if findLink(feed2.Links, opdspkg.RelNext) != nil {
		t.Error("page 2: should not have next link")
	}
}

func TestSeriesFeed_Pagination(t *testing.T) {
	h := setupOPDSHandler(t)
	ctx := t.Context()

	const totalSeries = opdspkg.PageSize + 5
	for i := range totalSeries {
		if _, err := h.DB.CreateSeries(ctx, fmt.Sprintf("Series %03d", i), nil, nil, nil); err != nil {
			t.Fatalf("create series %d: %v", i, err)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/opds/series?page=1", nil)
	w := httptest.NewRecorder()
	h.HandleOPDS(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: status = %d, want %d", w.Code, http.StatusOK)
	}
	feed := parseOPDSFeed(t, w.Body.Bytes())
	if len(feed.Entries) != opdspkg.PageSize {
		t.Errorf("page 1: entries = %d, want %d", len(feed.Entries), opdspkg.PageSize)
	}
	if findLink(feed.Links, opdspkg.RelNext) == nil {
		t.Error("page 1: missing next link")
	}
	if findLink(feed.Links, opdspkg.RelPrevious) != nil {
		t.Error("page 1: should not have previous link")
	}

	// Page 2: should have previous link but no next.
	r2 := httptest.NewRequest(http.MethodGet, "/opds/series?page=2", nil)
	w2 := httptest.NewRecorder()
	h.HandleOPDS(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("page 2: status = %d, want %d", w2.Code, http.StatusOK)
	}
	feed2 := parseOPDSFeed(t, w2.Body.Bytes())
	if len(feed2.Entries) != totalSeries-opdspkg.PageSize {
		t.Errorf("page 2: entries = %d, want %d", len(feed2.Entries), totalSeries-opdspkg.PageSize)
	}
	if findLink(feed2.Links, opdspkg.RelPrevious) == nil {
		t.Error("page 2: missing previous link")
	}
	if findLink(feed2.Links, opdspkg.RelNext) != nil {
		t.Error("page 2: should not have next link")
	}
}
