package opds

import (
"testing"

"github.com/stretchr/testify/require"
)

func TestMIMETypeForFileType(t *testing.T) {
tests := []struct {
name     string
fileType string
want     string
}{
{name: "epub", fileType: "epub", want: "application/epub+zip"},
{name: "case-insensitive", fileType: "PDF", want: "application/pdf"},
{name: "kepub alias", fileType: "kepub", want: "application/epub+zip"},
{name: "unknown type", fileType: "unknown", want: ""},
{name: "empty type", fileType: "", want: ""},
}

for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got := MIMETypeForFileType(tc.fileType)
require.Equal(t, tc.want, got, "case %q: MIMETypeForFileType(%q) returned unexpected MIME type", tc.name, tc.fileType)
})
}
}

func TestPaginationLinks(t *testing.T) {
tests := []struct {
name        string
selfURL     string
page        int
total       int
pageSize    int
contentType string
wantRels    []string
wantSelf    string
wantPrev    string
wantNext    string
}{
{
name:        "single page at exact boundary",
selfURL:     "https://example.com/opds/all",
page:        1,
total:       50,
pageSize:    50,
contentType: AcqContentType,
wantRels:    []string{RelSelf},
wantSelf:    "https://example.com/opds/all?page=1",
},
{
name:        "first of multiple pages",
selfURL:     "https://example.com/opds/all",
page:        1,
total:       100,
pageSize:    50,
contentType: NavContentType,
wantRels:    []string{RelSelf, RelNext},
wantSelf:    "https://example.com/opds/all?page=1",
wantNext:    "https://example.com/opds/all?page=2",
},
{
name:        "middle page",
selfURL:     "https://example.com/opds/all",
page:        2,
total:       151,
pageSize:    50,
contentType: AcqContentType,
wantRels:    []string{RelSelf, RelPrevious, RelNext},
wantSelf:    "https://example.com/opds/all?page=2",
wantPrev:    "https://example.com/opds/all?page=1",
wantNext:    "https://example.com/opds/all?page=3",
},
{
name:        "last page at exact boundary",
selfURL:     "https://example.com/opds/all",
page:        2,
total:       100,
pageSize:    50,
contentType: NavContentType,
wantRels:    []string{RelSelf, RelPrevious},
wantSelf:    "https://example.com/opds/all?page=2",
wantPrev:    "https://example.com/opds/all?page=1",
},
{
name:        "URL with existing query string",
selfURL:     "https://example.com/opds/search?q=foo%2Bbar&lang=en",
page:        1,
total:       100,
pageSize:    50,
contentType: AcqContentType,
wantRels:    []string{RelSelf, RelNext},
wantSelf:    "https://example.com/opds/search?q=foo%2Bbar&lang=en&page=1",
wantNext:    "https://example.com/opds/search?q=foo%2Bbar&lang=en&page=2",
},
{
name:        "page beyond total pages",
selfURL:     "https://example.com/opds/all",
page:        3,
total:       100,
pageSize:    50,
contentType: AcqContentType,
wantRels:    []string{RelSelf, RelPrevious},
wantSelf:    "https://example.com/opds/all?page=3",
wantPrev:    "https://example.com/opds/all?page=2",
},
}

for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
links := PaginationLinks(tc.selfURL, tc.page, tc.total, tc.pageSize, tc.contentType)

rels := make([]string, 0, len(links))
for i, link := range links {
rels = append(rels, link.Rel)
require.Equal(t, tc.contentType, link.Type, "case %q: link %d should keep the requested content type", tc.name, i)
}
require.Equal(t, tc.wantRels, rels, "case %q: unexpected link relation order", tc.name)

selfLink := findLinkByRel(links, RelSelf)
require.NotNil(t, selfLink, "case %q: self link must be present", tc.name)
require.Equal(t, tc.wantSelf, selfLink.Href, "case %q: self link URL mismatch", tc.name)

prevLink := findLinkByRel(links, RelPrevious)
if tc.wantPrev == "" {
require.Nil(t, prevLink, "case %q: previous link should be absent", tc.name)
} else {
require.NotNil(t, prevLink, "case %q: previous link should be present", tc.name)
require.Equal(t, tc.wantPrev, prevLink.Href, "case %q: previous link URL mismatch", tc.name)
}

nextLink := findLinkByRel(links, RelNext)
if tc.wantNext == "" {
require.Nil(t, nextLink, "case %q: next link should be absent", tc.name)
} else {
require.NotNil(t, nextLink, "case %q: next link should be present", tc.name)
require.Equal(t, tc.wantNext, nextLink.Href, "case %q: next link URL mismatch", tc.name)
}
})
}
}

func findLinkByRel(links []Link, rel string) *Link {
for i := range links {
if links[i].Rel == rel {
return &links[i]
}
}
return nil
}
