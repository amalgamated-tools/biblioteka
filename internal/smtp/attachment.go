package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
	"time"
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

// sanitizeFilename strips control characters (including CR/LF) from a
// filename so it is safe to embed in email headers.
func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1 // drop control chars
		}
		return r
	}, name)
}

// lineWrapWriter wraps an io.Writer and inserts CRLF every `width` bytes,
// as required by RFC 2045 for base64 content-transfer-encoding.
type lineWrapWriter struct {
	w     io.Writer
	width int
	col   int
}

func (lw *lineWrapWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		remaining := lw.width - lw.col
		if remaining <= 0 {
			if _, err := lw.w.Write([]byte("\r\n")); err != nil {
				return written, err
			}
			lw.col = 0
			remaining = lw.width
		}
		chunk := p
		if len(chunk) > remaining {
			chunk = p[:remaining]
		}
		n, err := lw.w.Write(chunk)
		written += n
		lw.col += n
		p = p[n:]
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// BuildAttachmentMessage constructs an RFC 5322 multipart/mixed email message
// that carries filename as a base64-encoded attachment. The plain-text body is
// a short human-readable note. params supplies the envelope From header.
func BuildAttachmentMessage(params SendParams, to, filename, fileType string, data []byte) ([]byte, error) {
	mimeType := MIMETypeForFileType(fileType)
	safeName := sanitizeFilename(filename)
	subject := mime.QEncoding.Encode("UTF-8", "Book: "+safeName)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	boundary := mw.Boundary()

	// RFC 5322 message headers.
	fmt.Fprintf(&buf, "From: %s\r\n", params.FromHeader)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n", boundary)
	fmt.Fprintf(&buf, "\r\n")

	// Plain-text body part.
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(textHeader)
	if err != nil {
		return nil, fmt.Errorf("create text part: %w", err)
	}
	qpw := quotedprintable.NewWriter(pw)
	if _, err := fmt.Fprintf(qpw, "Please find attached: %s\r\n", safeName); err != nil {
		return nil, fmt.Errorf("write text body: %w", err)
	}
	if err := qpw.Close(); err != nil {
		return nil, fmt.Errorf("close quoted-printable writer: %w", err)
	}

	// Attachment part — stream base64 through a line-wrapping writer to
	// avoid duplicating the entire payload as a single encoded string.
	attachHeader := textproto.MIMEHeader{}
	attachHeader.Set("Content-Type", mimeType)
	attachHeader.Set("Content-Transfer-Encoding", "base64")
	attachHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeName}))
	ap, err := mw.CreatePart(attachHeader)
	if err != nil {
		return nil, fmt.Errorf("create attachment part: %w", err)
	}
	lw := &lineWrapWriter{w: ap, width: 76}
	b64w := base64.NewEncoder(base64.StdEncoding, lw)
	if _, err := b64w.Write(data); err != nil {
		return nil, fmt.Errorf("write base64 data: %w", err)
	}
	if err := b64w.Close(); err != nil {
		return nil, fmt.Errorf("close base64 encoder: %w", err)
	}
	// Ensure a trailing CRLF after the last base64 line.
	if lw.col > 0 {
		if _, err := ap.Write([]byte("\r\n")); err != nil {
			return nil, fmt.Errorf("write trailing CRLF: %w", err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}
	return buf.Bytes(), nil
}
