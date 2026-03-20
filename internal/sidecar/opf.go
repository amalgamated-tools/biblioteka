package sidecar

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// OPFData holds the metadata fields to write into a metadata.opf file.
type OPFData struct {
	Title       string
	Author      string
	ISBN        string
	Language    string
	Date        string
	Publisher   string
	Description string
	HasCover    bool
}

// WriteOPF generates an OPF 2.0 metadata file and writes it to metadata.opf in dir.
func WriteOPF(dir string, data OPFData) error {
	xmlBytes, err := marshalOPF(data)
	if err != nil {
		return fmt.Errorf("marshal OPF: %w", err)
	}

	path := filepath.Join(dir, "metadata.opf")
	if err := os.WriteFile(path, xmlBytes, 0o644); err != nil {
		return fmt.Errorf("write metadata.opf: %w", err)
	}

	return nil
}

func marshalOPF(data OPFData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	start := xml.StartElement{
		Name: xml.Name{Local: "package"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: "http://www.idpf.org/2007/opf"},
			{Name: xml.Name{Local: "version"}, Value: "2.0"},
			{Name: xml.Name{Local: "unique-identifier"}, Value: "uid"},
		},
	}

	if err := enc.EncodeToken(start); err != nil {
		return nil, err
	}

	metaStart := xml.StartElement{
		Name: xml.Name{Local: "metadata"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:dc"}, Value: "http://purl.org/dc/elements/1.1/"},
			{Name: xml.Name{Local: "xmlns:opf"}, Value: "http://www.idpf.org/2007/opf"},
		},
	}
	if err := enc.EncodeToken(metaStart); err != nil {
		return nil, err
	}

	writeDCElement := func(name, value string) error {
		if value == "" {
			return nil
		}
		s := xml.StartElement{Name: xml.Name{Local: "dc:" + name}}
		if err := enc.EncodeToken(s); err != nil {
			return err
		}
		if err := enc.EncodeToken(xml.CharData(value)); err != nil {
			return err
		}
		return enc.EncodeToken(s.End())
	}

	if err := writeDCElement("title", data.Title); err != nil {
		return nil, err
	}
	if err := writeDCElement("creator", data.Author); err != nil {
		return nil, err
	}
	if data.ISBN != "" {
		s := xml.StartElement{
			Name: xml.Name{Local: "dc:identifier"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "id"}, Value: "uid"},
				{Name: xml.Name{Local: "opf:scheme"}, Value: "ISBN"},
			},
		}
		if err := enc.EncodeToken(s); err != nil {
			return nil, err
		}
		if err := enc.EncodeToken(xml.CharData([]byte(data.ISBN))); err != nil {
			return nil, err
		}
		if err := enc.EncodeToken(s.End()); err != nil {
			return nil, err
		}
	}
	if err := writeDCElement("language", data.Language); err != nil {
		return nil, err
	}
	if err := writeDCElement("date", data.Date); err != nil {
		return nil, err
	}
	if err := writeDCElement("publisher", data.Publisher); err != nil {
		return nil, err
	}
	if err := writeDCElement("description", data.Description); err != nil {
		return nil, err
	}

	if data.HasCover {
		metaEl := xml.StartElement{
			Name: xml.Name{Local: "meta"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: "cover"},
				{Name: xml.Name{Local: "content"}, Value: "cover-image"},
			},
		}
		if err := enc.EncodeToken(metaEl); err != nil {
			return nil, err
		}
		if err := enc.EncodeToken(metaEl.End()); err != nil {
			return nil, err
		}
	}

	if err := enc.EncodeToken(metaStart.End()); err != nil {
		return nil, err
	}

	if data.HasCover {
		manifestStart := xml.StartElement{Name: xml.Name{Local: "manifest"}}
		if err := enc.EncodeToken(manifestStart); err != nil {
			return nil, err
		}
		item := xml.StartElement{
			Name: xml.Name{Local: "item"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "id"}, Value: "cover-image"},
				{Name: xml.Name{Local: "href"}, Value: "cover.jpg"},
				{Name: xml.Name{Local: "media-type"}, Value: "image/jpeg"},
			},
		}
		if err := enc.EncodeToken(item); err != nil {
			return nil, err
		}
		if err := enc.EncodeToken(item.End()); err != nil {
			return nil, err
		}
		if err := enc.EncodeToken(manifestStart.End()); err != nil {
			return nil, err
		}
	}

	if err := enc.EncodeToken(start.End()); err != nil {
		return nil, err
	}

	if err := enc.Flush(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
