package coverutil

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestDecodeDataURL_ValidPNG(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:image/png;base64," + encoded

	mimeType, decoded, err := DecodeDataURL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mimeType != "image/png" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/png")
	}
	if len(decoded) != len(data) {
		t.Errorf("decoded length = %d, want %d", len(decoded), len(data))
	}
}

func TestDecodeDataURL_ValidJPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:image/jpeg;base64," + encoded

	mimeType, decoded, err := DecodeDataURL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want %q", mimeType, "image/jpeg")
	}
	if len(decoded) != len(data) {
		t.Errorf("decoded length = %d, want %d", len(decoded), len(data))
	}
}

func TestDecodeDataURL_NotDataURL(t *testing.T) {
	_, _, err := DecodeDataURL("https://example.com/image.jpg")
	if !errors.Is(err, ErrNotDataURL) {
		t.Errorf("err = %v, want ErrNotDataURL", err)
	}
}

func TestDecodeDataURL_EmptyString(t *testing.T) {
	_, _, err := DecodeDataURL("")
	if !errors.Is(err, ErrNotDataURL) {
		t.Errorf("err = %v, want ErrNotDataURL", err)
	}
}

func TestDecodeDataURL_UnsupportedEncoding(t *testing.T) {
	_, _, err := DecodeDataURL("data:image/png;utf8,hello")
	if err == nil {
		t.Fatal("expected error for unsupported encoding")
	}
}

func TestDecodeDataURL_EmptyMIME(t *testing.T) {
	data := []byte("hello")
	encoded := base64.StdEncoding.EncodeToString(data)
	raw := "data:;base64," + encoded

	mimeType, _, err := DecodeDataURL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mimeType != "text/plain;charset=US-ASCII" {
		t.Errorf("mimeType = %q, want %q", mimeType, "text/plain;charset=US-ASCII")
	}
}

func TestDecodeDataURL_InvalidBase64(t *testing.T) {
	_, _, err := DecodeDataURL("data:image/png;base64,!!!invalid!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeDataURL_ExceedsSizeLimit(t *testing.T) {
	// Create a payload that would exceed 20 MB when decoded.
	bigData := strings.Repeat("A", 30*1024*1024) // 30 MB of base64
	raw := "data:image/png;base64," + bigData

	_, _, err := DecodeDataURL(raw)
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
}
