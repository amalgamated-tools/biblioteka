package handlers

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"
)

// --- XML marshaling (opds_types.go) ---

func TestOPDSFeed_XMLMarshal(t *testing.T) {
	feed := &opdspkg.Feed{
		XMLNS:     opdspkg.XMLNSAtom,
		XMLNSOPDS: opdspkg.XMLNSOPDS,
		ID:        "urn:test",
		Title:     "Test Feed",
		Updated:   "2024-01-01T00:00:00Z",
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// Element name must be "feed"
	if !strings.Contains(s, "<feed ") && !strings.Contains(s, "<feed>") {
		t.Errorf("XML does not contain <feed> element: %s", s)
	}
	// Atom xmlns must be present as attribute
	if !strings.Contains(s, opdspkg.XMLNSAtom) {
		t.Errorf("XML missing Atom xmlns attribute: %s", s)
	}
	// OPDS xmlns must be present
	if !strings.Contains(s, opdspkg.XMLNSOPDS) {
		t.Errorf("XML missing OPDS xmlns attribute: %s", s)
	}
	// ID and Title must be child elements
	if !strings.Contains(s, "<id>urn:test</id>") {
		t.Errorf("XML missing <id> element: %s", s)
	}
	if !strings.Contains(s, "<title>Test Feed</title>") {
		t.Errorf("XML missing <title> element: %s", s)
	}
}

func TestOPDSFeed_XMLMarshal_OmitEmptyOPDSNS(t *testing.T) {
	feed := &opdspkg.Feed{
		XMLNS:   opdspkg.XMLNSAtom,
		ID:      "urn:test",
		Title:   "Nav Feed",
		Updated: "2024-01-01T00:00:00Z",
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// When XMLNSOPDS is empty it should be omitted (omitempty)
	if strings.Contains(s, "xmlns:opds") {
		t.Errorf("xmlns:opds attribute should be absent when empty, got: %s", s)
	}
}

func TestOPDSLink_XMLMarshal(t *testing.T) {
	link := opdspkg.Link{
		Rel:  opdspkg.RelSelf,
		Href: "http://example.com/opds",
		Type: opdspkg.NavContentType,
	}

	data, err := xml.Marshal(link)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `rel="self"`) {
		t.Errorf("XML missing rel attribute: %s", s)
	}
	if !strings.Contains(s, `href="http://example.com/opds"`) {
		t.Errorf("XML missing href attribute: %s", s)
	}
	if !strings.Contains(s, `type=`) {
		t.Errorf("XML missing type attribute: %s", s)
	}
}

func TestOPDSContent_XMLMarshal(t *testing.T) {
	content := opdspkg.Content{
		Type:  "text",
		Value: "Some description text",
	}

	data, err := xml.Marshal(content)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `type="text"`) {
		t.Errorf("XML missing type attribute: %s", s)
	}
	// Value must be encoded as character data (not a child element)
	if !strings.Contains(s, "Some description text") {
		t.Errorf("XML missing chardata value: %s", s)
	}
	if strings.Contains(s, "<Value>") {
		t.Errorf("Value should be chardata, not a child element: %s", s)
	}
}

func TestOPDSEntry_XMLMarshal_Full(t *testing.T) {
	entry := opdspkg.Entry{
		Title:   "My Book",
		ID:      "urn:book:1",
		Updated: "2024-01-01T00:00:00Z",
		Content: &opdspkg.Content{Type: "text", Value: "A description"},
		Authors: []opdspkg.Author{{Name: "Jane Doe"}},
		Links: []opdspkg.Link{
			{Rel: opdspkg.RelAcquisition, Href: "http://example.com/dl/1", Type: "application/epub+zip"},
		},
	}

	data, err := xml.Marshal(entry)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, "<title>My Book</title>") {
		t.Errorf("XML missing title: %s", s)
	}
	if !strings.Contains(s, "<id>urn:book:1</id>") {
		t.Errorf("XML missing id: %s", s)
	}
	if !strings.Contains(s, "A description") {
		t.Errorf("XML missing content: %s", s)
	}
	if !strings.Contains(s, "<name>Jane Doe</name>") {
		t.Errorf("XML missing author name: %s", s)
	}
}

func TestOPDSEntry_XMLMarshal_NoContent(t *testing.T) {
	entry := opdspkg.Entry{
		Title:   "No Desc",
		ID:      "urn:book:2",
		Updated: "2024-01-01T00:00:00Z",
		Links:   []opdspkg.Link{{Rel: opdspkg.RelSelf, Href: "/x", Type: opdspkg.AcqContentType}},
	}

	data, err := xml.Marshal(entry)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	// Content should be absent when nil (omitempty)
	if strings.Contains(s, "<content") {
		t.Errorf("XML should have no <content> when nil, got: %s", s)
	}
}

func TestOPDSFeed_XMLMarshal_LinksAndEntries(t *testing.T) {
	feed := &opdspkg.Feed{
		XMLNS:   opdspkg.XMLNSAtom,
		ID:      "urn:test:2",
		Title:   "Books",
		Updated: "2024-01-01T00:00:00Z",
		Links: []opdspkg.Link{
			{Rel: opdspkg.RelSelf, Href: "/opds/all?page=1", Type: opdspkg.AcqContentType},
			{Rel: opdspkg.RelNext, Href: "/opds/all?page=2", Type: opdspkg.AcqContentType},
		},
		Entries: []opdspkg.Entry{
			{Title: "Book One", ID: "urn:b:1", Updated: "2024-01-01T00:00:00Z"},
		},
	}

	data, err := xml.Marshal(feed)
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	s := string(data)

	if !strings.Contains(s, `rel="self"`) {
		t.Errorf("XML missing self link: %s", s)
	}
	if !strings.Contains(s, `rel="next"`) {
		t.Errorf("XML missing next link: %s", s)
	}
	if !strings.Contains(s, "<title>Book One</title>") {
		t.Errorf("XML missing entry: %s", s)
	}
}

// --- writeOPDSError (opds_helpers.go) ---

func TestWriteOPDSError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.AcqContentType, "urn:biblioteka:opds:error", "Something went wrong")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.AcqContentType)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "<?xml") {
		t.Errorf("body should start with XML declaration, got: %s", body[:min(len(body), 50)])
	}

	// Must be a valid feed with the provided title
	feed := parseOPDSFeed(t, w.Body.Bytes())
	if feed.Title != "Something went wrong" {
		t.Errorf("title = %q, want %q", feed.Title, "Something went wrong")
	}
	if feed.ID != "urn:biblioteka:opds:error" {
		t.Errorf("id = %q, want %q", feed.ID, "urn:biblioteka:opds:error")
	}
}

func TestWriteOPDSError_NavContentType(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.NavContentType, "urn:test", "Nav error")

	if ct := w.Header().Get("Content-Type"); ct != opdspkg.NavContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.NavContentType)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- writeOPDSFeed (opds_helpers.go) ---

func TestWriteOPDSFeed_Direct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()

	feed := &opdspkg.Feed{
		XMLNS:   opdspkg.XMLNSAtom,
		ID:      "urn:test",
		Title:   "Direct Feed",
		Updated: "2024-01-01T00:00:00Z",
	}
	writeOPDSFeed(r, w, opdspkg.AcqContentType, feed)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != opdspkg.AcqContentType {
		t.Errorf("Content-Type = %q, want %q", ct, opdspkg.AcqContentType)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "<?xml") {
		t.Errorf("body should start with XML declaration, got: %s", body[:min(len(body), 50)])
	}
	parsed := parseOPDSFeed(t, w.Body.Bytes())
	if parsed.Title != "Direct Feed" {
		t.Errorf("title = %q, want %q", parsed.Title, "Direct Feed")
	}
}
