package filetype

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMIMEType(t *testing.T) {
	tests := []struct {
		fileType string
		want     string
	}{
		{"epub", "application/epub+zip"},
		{"EPUB", "application/epub+zip"},
		{"pdf", "application/pdf"},
		{"mobi", "application/x-mobipocket-ebook"},
		{"azw3", "application/vnd.amazon.ebook"},
		{"cbz", "application/vnd.comicbook+zip"},
		{"cbr", "application/vnd.comicbook-rar"},
		{"fb2", "application/x-fictionbook+xml"},
		{"kepub", "application/epub+zip"},
		{"txt", "text/plain"},
		{"djvu", "image/vnd.djvu"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			require.Equal(t, tt.want, MIMEType(tt.fileType))
		})
	}
}

func TestMIMETypeOrOctetStream(t *testing.T) {
	tests := []struct {
		fileType string
		want     string
	}{
		{"epub", "application/epub+zip"},
		{"EPUB", "application/epub+zip"},
		{"pdf", "application/pdf"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			require.Equal(t, tt.want, MIMETypeOrOctetStream(tt.fileType))
		})
	}
}
