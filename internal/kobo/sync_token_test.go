package kobo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseSyncToken_Empty(t *testing.T) {
	tok := ParseSyncToken("")
	require.True(t, tok.BooksLastModified.IsZero() || !tok.ReadingStateLastModified.IsZero())
}

func TestParseSyncToken_Garbage(t *testing.T) {
	tok := ParseSyncToken("not-base64!!!")
	require.True(t, tok.BooksLastModified.IsZero())
}

func TestSyncTokenRoundTrip_Zero(t *testing.T) {
	tok := SyncToken{}
	encoded := EncodeSyncToken(tok)
	decoded := ParseSyncToken(encoded)
	require.True(t, decoded.BooksLastModified.IsZero())
	require.True(t, decoded.ReadingStateLastModified.IsZero())
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
	require.True(t, decoded.BooksLastModified.Equal(tok.BooksLastModified))
	require.Equal(t, tok.BooksLastID, decoded.BooksLastID)
	require.True(t, decoded.ReadingStateLastModified.Equal(tok.ReadingStateLastModified))
}

func TestEncodeSyncToken_ProducesBase64(t *testing.T) {
	tok := SyncToken{BooksLastModified: time.Now().UTC()}
	encoded := EncodeSyncToken(tok)
	require.NotEqual(t, "", encoded)
	// Must be decodeable back to the same token.
	decoded := ParseSyncToken(encoded)
	require.False(t, decoded.BooksLastModified.IsZero())
}
