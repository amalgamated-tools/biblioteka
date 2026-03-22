package sidecar

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/google/uuid"
)

// bibliotekaNamespace is a fixed, application-specific namespace UUID for
// deriving deterministic v5 UUIDs from book metadata.
var bibliotekaNamespace = uuid.MustParse("a5d3b2e1-7f4c-4e8a-9d6b-1c2e3f4a5b6d")

// OPFData holds the metadata fields to write into a metadata.opf file.
type OPFData struct {
	Title          string
	Author         string
	ISBN           string
	Language       string
	Date           string
	Publisher      string
	Description    string
	CoverFilename  string // e.g. "cover.jpg", "cover.png"; empty means no cover
	CoverMediaType string // e.g. "image/jpeg", "image/png"
}

// opfPackage is the top-level OPF 2.0 package element.
type opfPackage struct {
	XMLName          xml.Name     `xml:"package"`
	XMLNS            string       `xml:"xmlns,attr"`
	Version          string       `xml:"version,attr"`
	UniqueIdentifier string       `xml:"unique-identifier,attr"`
	Metadata         opfMetadata  `xml:"metadata"`
	Manifest         *opfManifest `xml:"manifest,omitempty"`
}

// opfManifest represents the manifest element.
type opfManifest struct {
	Items []opfItem `xml:"item"`
}

// opfItem represents an item in the manifest.
type opfItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// opfMetadata holds metadata fields. It implements xml.Marshaler to produce
// dc:-prefixed Dublin Core elements that Go's struct tags cannot express.
type opfMetadata struct {
	Title       string
	Creator     string
	Identifier  string
	IdScheme    string
	Language    string
	Date        string
	Publisher   string
	Description string
	HasCover    bool
}

// MarshalXML writes the <metadata> element with Dublin Core namespace-prefixed
// children and conditional cover meta.
func (m opfMetadata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = append(start.Attr,
		xml.Attr{Name: xml.Name{Local: "xmlns:dc"}, Value: "http://purl.org/dc/elements/1.1/"},
		xml.Attr{Name: xml.Name{Local: "xmlns:opf"}, Value: "http://www.idpf.org/2007/opf"},
	)
	if err := e.EncodeToken(start); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode metadata start element: %w", err)
	}

	// dc writes a Dublin Core element, skipping it when value is empty.
	dc := func(name, value string, attrs ...xml.Attr) error {
		if value == "" {
			return nil
		}
		el := xml.StartElement{Name: xml.Name{Local: "dc:" + name}, Attr: attrs}
		if err := e.EncodeToken(el); err != nil {
			// Let's let the top-level caller decide when/how to log errors since they have more context
			return fmt.Errorf("failed to encode dc:%s start element: %w", name, err)
		}
		if err := e.EncodeToken(xml.CharData(value)); err != nil {
			// Let's let the top-level caller decide when/how to log errors since they have more context
			return fmt.Errorf("failed to encode dc:%s char data: %w", name, err)
		}
		if err := e.EncodeToken(el.End()); err != nil {
			// Let's let the top-level caller decide when/how to log errors since they have more context
			return fmt.Errorf("failed to encode dc:%s end element: %w", name, err)
		}
		return nil
	}

	if err := dc("title", m.Title); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:title element: %w", err)
	}
	if err := dc("creator", m.Creator, xml.Attr{Name: xml.Name{Local: "opf:role"}, Value: "aut"}); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:creator element: %w", err)
	}

	// dc:identifier is always present.
	idEl := xml.StartElement{
		Name: xml.Name{Local: "dc:identifier"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: "uid"},
			{Name: xml.Name{Local: "opf:scheme"}, Value: m.IdScheme},
		},
	}
	if err := e.EncodeToken(idEl); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:identifier start element: %w", err)
	}
	if err := e.EncodeToken(xml.CharData(m.Identifier)); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:identifier char data: %w", err)
	}
	if err := e.EncodeToken(idEl.End()); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:identifier end element: %w", err)
	}

	if err := dc("language", m.Language); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:language element: %w", err)
	}
	if err := dc("date", m.Date); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:date element: %w", err)
	}
	if err := dc("publisher", m.Publisher); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:publisher element: %w", err)
	}
	if err := dc("description", m.Description); err != nil {
		// Let's let the top-level caller decide when/how to log errors since they have more context
		return fmt.Errorf("failed to encode dc:description element: %w", err)
	}

	if m.HasCover {
		meta := xml.StartElement{
			Name: xml.Name{Local: "meta"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: "cover"},
				{Name: xml.Name{Local: "content"}, Value: "cover-image"},
			},
		}
		if err := e.EncodeToken(meta); err != nil {
			// Let's let the top-level caller decide when/how to log errors since they have more context
			return fmt.Errorf("failed to encode meta start element: %w", err)
		}
		if err := e.EncodeToken(meta.End()); err != nil {
			// Let's let the top-level caller decide when/how to log errors since they have more context
			return fmt.Errorf("failed to encode meta end element: %w", err)
		}
	}

	return e.EncodeToken(start.End())
}

// WriteOPF generates an OPF 2.0 metadata file and writes it to dir.
// When baseName is empty the file is named "metadata.opf"; when set it is
// named "{baseName}.opf" (used for book_per_file mode where multiple books
// share a directory).
func WriteOPF(ctx context.Context, dir string, data OPFData, baseName string) error {
	if err := validateBaseName(baseName); err != nil {
		return fmt.Errorf("invalid OPF base name %q: %w", baseName, err)
	}
	if data.Title == "" {
		return fmt.Errorf("WriteOPF: Title is required by OPF 2.0")
	}
	if data.Language == "" {
		slog.DebugContext(
			ctx,
			"missing language, defaulting to 'und'",
			slog.String(otelkeys.Path, dir),
			slog.String(otelkeys.BaseName, baseName),
		)
		data.Language = "und"
	}
	if (data.CoverFilename == "") != (data.CoverMediaType == "") {
		return fmt.Errorf("WriteOPF: CoverFilename and CoverMediaType must both be set or both be empty")
	}

	xmlBytes, err := marshalOPF(dir, data)
	if err != nil {
		return fmt.Errorf("marshal OPF: %w", err)
	}

	opfName := "metadata.opf"
	if baseName != "" {
		opfName = baseName + ".opf"
	}
	path := filepath.Join(dir, opfName)
	tmpFile, err := os.CreateTemp(dir, opfName+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp %s: %w", opfName, err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(xmlBytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp %s: %w", opfName, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", opfName, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod temp %s: %w", opfName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp %s: %w", opfName, err)
	}

	return nil
}

func marshalOPF(dir string, data OPFData) ([]byte, error) {
	identifierValue := data.ISBN
	identifierScheme := "ISBN"
	if identifierValue == "" {
		// Derive a stable, deterministic UUID from available metadata so that
		// the identifier does not change across repeated OPF writes for the
		// same book when ISBN is missing.
		stableKey := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s", data.Title, data.Author, data.Publisher, data.Date, dir)
		u := uuid.NewSHA1(bibliotekaNamespace, []byte(stableKey))
		identifierValue = "urn:uuid:" + u.String()
		identifierScheme = "UUID"
	}

	pkg := opfPackage{
		XMLNS:            "http://www.idpf.org/2007/opf",
		Version:          "2.0",
		UniqueIdentifier: "uid",
		Metadata: opfMetadata{
			Title:       data.Title,
			Creator:     data.Author,
			Identifier:  identifierValue,
			IdScheme:    identifierScheme,
			Language:    data.Language,
			Date:        data.Date,
			Publisher:   data.Publisher,
			Description: data.Description,
			HasCover:    data.CoverFilename != "",
		},
	}

	if data.CoverFilename != "" {
		pkg.Manifest = &opfManifest{
			Items: []opfItem{{
				ID:        "cover-image",
				Href:      data.CoverFilename,
				MediaType: data.CoverMediaType,
			}},
		}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	if err := enc.Encode(pkg); err != nil {
		return nil, fmt.Errorf("encode OPF: %w", err)
	}

	return buf.Bytes(), nil
}
