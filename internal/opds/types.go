// Package opds provides the Atom/OPDS data types, constants, and feed-building
// helpers used by the OPDS catalog handler. It is intentionally free of HTTP
// dependencies so that feed construction can be tested and reused independently
// of the HTTP handler layer.
package opds

import "encoding/xml"

// Atom/OPDS XML namespace constants.
const (
	XMLNSAtom       = "http://www.w3.org/2005/Atom"
	XMLNSOPDs       = "http://opds-spec.org/2010/catalog"
	XMLNSOpenSearch = "http://a9.com/-/spec/opensearch/1.1/"
)

// MIME type constants for OPDS content types.
const (
	NavContentType = "application/atom+xml;profile=opds-catalog;kind=navigation"
	AcqContentType = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	SearchType     = "application/opensearchdescription+xml"
)

// PageSize is the default number of entries per OPDS page.
const PageSize = 50

// Link relation constants used in OPDS feeds.
const (
	RelSelf        = "self"
	RelStart       = "start"
	RelNext        = "next"
	RelPrevious    = "previous"
	RelSearch      = "search"
	RelSubsection  = "subsection"
	RelAcquisition = "http://opds-spec.org/acquisition"
	RelImage       = "http://opds-spec.org/image"
)

// Feed is the top-level Atom feed element.
type Feed struct {
	XMLName   xml.Name `xml:"feed"`
	XMLNS     string   `xml:"xmlns,attr"`
	XMLNSOPDS string   `xml:"xmlns:opds,attr,omitempty"`
	ID        string   `xml:"id"`
	Title     string   `xml:"title"`
	Updated   string   `xml:"updated"`
	Author    *Author  `xml:"author,omitempty"`
	Links     []Link   `xml:"link"`
	Entries   []Entry  `xml:"entry"`
}

// Entry is an Atom entry element within an OPDS feed.
type Entry struct {
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Content *Content `xml:"content,omitempty"`
	Authors []Author `xml:"author,omitempty"`
	Links   []Link   `xml:"link"`
}

// Link is an Atom link element.
type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

// Author is an Atom author element.
type Author struct {
	Name string `xml:"name"`
}

// Content is an Atom content element.
type Content struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// NavEntity holds the minimal data needed to build a named-entity navigation
// feed entry (e.g. an author or a series).
type NavEntity struct {
	ID      string
	Name    string
	Updated string // pre-formatted RFC3339
}
