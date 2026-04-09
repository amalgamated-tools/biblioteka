package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

// fileTypeMIME maps Biblioteka file_type values to MIME content types.
var fileTypeMIME = map[string]string{
	"epub":  "application/epub+zip",
	"pdf":   "application/pdf",
	"mobi":  "application/x-mobipocket-ebook",
	"azw3":  "application/vnd.amazon.ebook",
	"cbz":   "application/vnd.comicbook+zip",
	"cbr":   "application/vnd.comicbook-rar",
	"fb2":   "application/x-fictionbook+xml",
	"kepub": "application/epub+zip",
	"txt":   "text/plain",
	"djvu":  "image/vnd.djvu",
}

// MIMETypeForFileType returns the MIME content type for a given book file
// type string. If the type is not recognised, "application/octet-stream" is
// returned so the file is always treated as a generic download.
func MIMETypeForFileType(fileType string) string {
	if mt := fileTypeMIME[strings.ToLower(fileType)]; mt != "" {
		return mt
	}
	return "application/octet-stream"
}

// BuildAttachmentMessage constructs an RFC 5322 multipart/mixed email message
// that carries filename as a base64-encoded attachment. The plain-text body is
// a short human-readable note. params supplies the envelope From header.
func BuildAttachmentMessage(params SendParams, to, filename, fileType string, data []byte) []byte {
	mimeType := MIMETypeForFileType(fileType)
	subject := fmt.Sprintf("Book: %s", filename)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	boundary := mw.Boundary()

	// RFC 5322 message headers.
	fmt.Fprintf(&buf, "From: %s\r\n", params.FromHeader)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buf, "\r\n")

	// Plain-text body part.
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, _ := mw.CreatePart(textHeader)
	qpw := quotedprintable.NewWriter(pw)
	fmt.Fprintf(qpw, "Please find attached: %s\r\n", filename)
	qpw.Close()

	// Attachment part.
	attachHeader := textproto.MIMEHeader{}
	attachHeader.Set("Content-Type", mimeType)
	attachHeader.Set("Content-Transfer-Encoding", "base64")
	attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	ap, _ := mw.CreatePart(attachHeader)
	encoded := base64.StdEncoding.EncodeToString(data)
	// Wrap base64 at 76 characters per line as recommended by RFC 2045.
	for len(encoded) > 76 {
		fmt.Fprintf(ap, "%s\r\n", encoded[:76])
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		fmt.Fprintf(ap, "%s\r\n", encoded)
	}

	mw.Close()
	return buf.Bytes()
}
