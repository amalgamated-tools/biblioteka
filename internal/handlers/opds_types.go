package handlers

import "encoding/xml"

const (
	xmlnsAtom       = "http://www.w3.org/2005/Atom"
	xmlnsOPDS       = "http://opds-spec.org/2010/catalog"
	xmlnsOpenSearch = "http://a9.com/-/spec/opensearch/1.1/"

	opdsNavContentType = "application/atom+xml;profile=opds-catalog;kind=navigation"
	opdsAcqContentType = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	opdsSearchType     = "application/opensearchdescription+xml"

	opdsPageSize = 50

	relSelf        = "self"
	relStart       = "start"
	relNext        = "next"
	relPrevious    = "previous"
	relSearch      = "search"
	relSubsection  = "subsection"
	relAcquisition = "http://opds-spec.org/acquisition"
	relImage       = "http://opds-spec.org/image"
)

type opdsFeed struct {
	XMLName   xml.Name    `xml:"feed"`
	XMLNS     string      `xml:"xmlns,attr"`
	XMLNSOPDS string      `xml:"xmlns:opds,attr,omitempty"`
	ID        string      `xml:"id"`
	Title     string      `xml:"title"`
	Updated   string      `xml:"updated"`
	Author    *opdsAuthor `xml:"author,omitempty"`
	Links     []opdsLink  `xml:"link"`
	Entries   []opdsEntry `xml:"entry"`
}

type opdsEntry struct {
	Title   string       `xml:"title"`
	ID      string       `xml:"id"`
	Updated string       `xml:"updated"`
	Content *opdsContent `xml:"content,omitempty"`
	Authors []opdsAuthor `xml:"author,omitempty"`
	Links   []opdsLink   `xml:"link"`
}

type opdsLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

type opdsAuthor struct {
	Name string `xml:"name"`
}

type opdsContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}
