// Package filetype provides a canonical mapping from Biblioteka book file
// type strings (e.g. "epub", "pdf") to MIME content types. Both the OPDS
// catalog and the SMTP attachment builder share this single source of truth.
package filetype

import "strings"

// mimeTypes maps lowercase Biblioteka file_type values to MIME content types.
var mimeTypes = map[string]string{
	"azw3":  "application/vnd.amazon.ebook",
	"cbr":   "application/vnd.comicbook-rar",
	"cbz":   "application/vnd.comicbook+zip",
	"djvu":  "image/vnd.djvu",
	"epub":  "application/epub+zip",
	"fb2":   "application/x-fictionbook+xml",
	"kepub": "application/epub+zip",
	"mobi":  "application/x-mobipocket-ebook",
	"pdf":   "application/pdf",
	"txt":   "text/plain",
}

// MIMEType returns the MIME content type for a Biblioteka file type string.
// The lookup is case-insensitive. If the type is not recognized, an empty
// string is returned so the caller can decide on a fallback.
func MIMEType(fileType string) string {
	return mimeTypes[strings.ToLower(fileType)]
}
