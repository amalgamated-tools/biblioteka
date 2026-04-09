package smtp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMIMETypeForFileType(t *testing.T) {
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
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			require.Equal(t, tt.want, MIMETypeForFileType(tt.fileType))
		})
	}
}

func TestBuildAttachmentMessage(t *testing.T) {
	params := SendParams{
		Addr:       "smtp.example.com:587",
		From:       "noreply@example.com",
		FromHeader: "noreply@example.com",
		TLS:        "starttls",
	}
	data := []byte("fake epub content")
	msg := BuildAttachmentMessage(params, "reader@example.com", "MyBook.epub", "epub", data)

	msgStr := string(msg)

	require.Contains(t, msgStr, "From: noreply@example.com\r\n")
	require.Contains(t, msgStr, "To: reader@example.com\r\n")
	require.Contains(t, msgStr, "Subject: Book: MyBook.epub\r\n")
	require.Contains(t, msgStr, "MIME-Version: 1.0\r\n")
	require.Contains(t, msgStr, "multipart/mixed")
	require.Contains(t, msgStr, "Content-Type: application/epub+zip")
	require.Contains(t, msgStr, `attachment; filename="MyBook.epub"`)
	require.Contains(t, msgStr, "base64")
}

func TestBuildAttachmentMessage_DisplayNameFrom(t *testing.T) {
	params := SendParams{
		Addr:       "smtp.example.com:587",
		From:       "noreply@example.com",
		FromHeader: `"My Library" <noreply@example.com>`,
		TLS:        "starttls",
	}
	data := []byte("content")
	msg := BuildAttachmentMessage(params, "reader@example.com", "Book.pdf", "pdf", data)

	msgStr := string(msg)
	require.True(t,
		strings.Contains(msgStr, `From: "My Library" <noreply@example.com>`),
		"From header should use display name format",
	)
	require.Contains(t, msgStr, "Content-Type: application/pdf")
}

func TestBuildAttachmentMessage_UnknownFileType(t *testing.T) {
	params := SendParams{
		Addr:       "smtp.example.com:587",
		From:       "noreply@example.com",
		FromHeader: "noreply@example.com",
		TLS:        "starttls",
	}
	data := []byte("binary content")
	msg := BuildAttachmentMessage(params, "reader@example.com", "Book.xyz", "xyz", data)

	msgStr := string(msg)
	require.Contains(t, msgStr, "Content-Type: application/octet-stream")
}
