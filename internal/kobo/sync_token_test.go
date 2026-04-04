package kobo

import (
	"testing"
	"time"
)

func TestParseSyncToken_Empty(t *testing.T) {
	tok := ParseSyncToken("")
	if !tok.BooksLastModified.IsZero() || !tok.ReadingStateLastModified.IsZero() {
		t.Error("expected zero values for empty token")
	}
}

func TestParseSyncToken_Garbage(t *testing.T) {
	tok := ParseSyncToken("not-base64!!!")
	if !tok.BooksLastModified.IsZero() {
		t.Error("expected zero BooksLastModified for garbage token")
	}
}

func TestSyncTokenRoundTrip_Zero(t *testing.T) {
	tok := SyncToken{}
	encoded := EncodeSyncToken(tok)
	decoded := ParseSyncToken(encoded)
	if !decoded.BooksLastModified.IsZero() {
		t.Errorf("BooksLastModified: got %v, want zero", decoded.BooksLastModified)
	}
	if !decoded.ReadingStateLastModified.IsZero() {
		t.Errorf("ReadingStateLastModified: got %v, want zero", decoded.ReadingStateLastModified)
	}
}

func TestSyncTokenRoundTrip_NonZero(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tok := SyncToken{
		BooksLastModified:        now,
		BooksLastID:              "book-abc",
		ReadingStateLastModified: now.Add(-time.Hour),
	}
	encoded := EncodeSyncToken(tok)
	decoded := ParseSyncToken(encoded)
	if !decoded.BooksLastModified.Equal(tok.BooksLastModified) {
		t.Errorf("BooksLastModified: got %v, want %v", decoded.BooksLastModified, tok.BooksLastModified)
	}
	if decoded.BooksLastID != tok.BooksLastID {
		t.Errorf("BooksLastID: got %q, want %q", decoded.BooksLastID, tok.BooksLastID)
	}
	if !decoded.ReadingStateLastModified.Equal(tok.ReadingStateLastModified) {
		t.Errorf("ReadingStateLastModified: got %v, want %v", decoded.ReadingStateLastModified, tok.ReadingStateLastModified)
	}
}

func TestEncodeSyncToken_ProducesBase64(t *testing.T) {
	tok := SyncToken{BooksLastModified: time.Now().UTC()}
	encoded := EncodeSyncToken(tok)
	if encoded == "" {
		t.Error("expected non-empty encoded token")
	}
	// Must be decodeable back to the same token.
	decoded := ParseSyncToken(encoded)
	if decoded.BooksLastModified.IsZero() {
		t.Error("expected non-zero BooksLastModified after round-trip")
	}
}
