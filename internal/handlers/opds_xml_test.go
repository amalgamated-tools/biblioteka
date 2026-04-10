package handlers

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	opdspkg "github.com/amalgamated-tools/biblioteka/internal/opds"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	// Element name must be "feed"
	require.Contains(t, s, "<feed ")
	// Atom xmlns must be present as attribute
	require.Contains(t, s, opdspkg.XMLNSAtom)
	// OPDS xmlns must be present
	require.Contains(t, s, opdspkg.XMLNSOPDS)
	// ID and Title must be child elements
	require.Contains(t, s, "<id>urn:test</id>")
	require.Contains(t, s, "<title>Test Feed</title>")
}

func TestOPDSFeed_XMLMarshal_OmitEmptyOPDSNS(t *testing.T) {
	feed := &opdspkg.Feed{
		XMLNS:   opdspkg.XMLNSAtom,
		ID:      "urn:test",
		Title:   "Nav Feed",
		Updated: "2024-01-01T00:00:00Z",
	}

	data, err := xml.Marshal(feed)
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	// When XMLNSOPDS is empty it should be omitted (omitempty)
	require.NotContains(t, s, "xmlns:opds")
}

func TestOPDSLink_XMLMarshal(t *testing.T) {
	link := opdspkg.Link{
		Rel:  opdspkg.RelSelf,
		Href: "http://example.com/opds",
		Type: opdspkg.NavContentType,
	}

	data, err := xml.Marshal(link)
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	require.Contains(t, s, `rel="self"`)
	require.Contains(t, s, `href="http://example.com/opds"`)
	require.Contains(t, s, `type=`)
}

func TestOPDSContent_XMLMarshal(t *testing.T) {
	content := opdspkg.Content{
		Type:  "text",
		Value: "Some description text",
	}

	data, err := xml.Marshal(content)
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	require.Contains(t, s, `type="text"`)
	// Value must be encoded as character data (not a child element)
	require.Contains(t, s, "Some description text")
	require.NotContains(t, s, "<Value>")
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
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	require.Contains(t, s, "<title>My Book</title>")
	require.Contains(t, s, "<id>urn:book:1</id>")
	require.Contains(t, s, "A description")
	require.Contains(t, s, "<name>Jane Doe</name>")
}

func TestOPDSEntry_XMLMarshal_NoContent(t *testing.T) {
	entry := opdspkg.Entry{
		Title:   "No Desc",
		ID:      "urn:book:2",
		Updated: "2024-01-01T00:00:00Z",
		Links:   []opdspkg.Link{{Rel: opdspkg.RelSelf, Href: "/x", Type: opdspkg.AcqContentType}},
	}

	data, err := xml.Marshal(entry)
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	// Content should be absent when nil (omitempty)
	require.NotContains(t, s, "<content")
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
	require.NoError(t, err, "xml.Marshal")
	s := string(data)

	require.Contains(t, s, `rel="self"`)
	require.Contains(t, s, `rel="next"`)
	require.Contains(t, s, "<title>Book One</title>")
}

// --- writeOPDSError (opds_helpers.go) ---

func TestWriteOPDSError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/all", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.AcqContentType, "urn:biblioteka:opds:error", "Something went wrong")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)

	body := w.Body.String()
	require.True(t, strings.HasPrefix(body, "<?xml"))

	// Must be a valid feed with the provided title
	feed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "Something went wrong", feed.Title)
	require.Equal(t, "urn:biblioteka:opds:error", feed.ID)
}

func TestWriteOPDSError_NavContentType(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/opds/authors", nil)
	w := httptest.NewRecorder()

	writeOPDSError(r, w, http.StatusInternalServerError, opdspkg.NavContentType, "urn:test", "Nav error")

	ct := w.Header().Get("Content-Type")

	require.Equal(t, opdspkg.NavContentType, ct)
	require.Equal(t, http.StatusInternalServerError, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)
	ct := w.Header().Get("Content-Type")
	require.Equal(t, opdspkg.AcqContentType, ct)
	body := w.Body.String()
	require.True(t, strings.HasPrefix(body, "<?xml"))
	parsed := parseOPDSFeed(t, w.Body.Bytes())
	require.Equal(t, "Direct Feed", parsed.Title)
}
