package exif

// ExifToolOutput holds the parsed result of exiftool's tab-separated output
// produced by the wrapper's exiftool invocation (e.g. `exiftool -a -u -f -ee3 -U -api ... -t`).
type ExifToolOutput struct {
	// File info
	Directory       string
	ExifToolVersion string
	FileName        string
	FilePath        string
	FileSize        string
	FileType        string
	Format          string
	MIMEType        string

	// Book metadata
	ASIN            string
	CoverImage      *ManifestItem
	CoverImageURL   string
	Author          string
	CalibreID       string
	CreatorFileAs   string
	CreatorRole     string
	Description     string
	GoodreadsID     string
	GoogleID        string
	HardcoverID     string
	ISBN10          string
	ISBN13          string
	Language        string
	PublicationDate string
	Publisher       string
	Subjects        []string
	Title           string

	// Repeated/nested structures
	Identifiers   []Identifier
	MetaTags      []MetaTag
	ManifestItems []ManifestItem

	// Catch-all for any unrecognized scalar fields.
	Extras map[string]string
}

func (e *ExifToolOutput) ISBN() string {
	if e.ISBN13 != "" {
		return e.ISBN13
	}
	if e.ISBN10 != "" {
		return e.ISBN10
	}
	return ""
}

func (e *ExifToolOutput) SetISBN(isbn string) {
	switch len(isbn) {
	case 0:
		e.ISBN10 = ""
		e.ISBN13 = ""
	case 10:
		e.ISBN10 = isbn
		e.ISBN13 = ""
	case 13:
		e.ISBN13 = isbn
		e.ISBN10 = ""
	}
}

// Identifier represents a Dublin Core identifier extracted from EPUB metadata.
// Scheme may be empty for bare URN-style identifiers.
type Identifier struct {
	Value  string
	Scheme string
	ID     string
}

// MetaTag represents a <meta> name/content pair from EPUB metadata.
type MetaTag struct {
	Content string
	Name    string
}

// ManifestItem represents an OPF manifest entry.
type ManifestItem struct {
	Href       string
	ID         string
	MediaType  string
	Properties string
}

// GuideReference represents an OPF guide reference.
type GuideReference struct {
	Href  string
	Title string
	Type  string
}
