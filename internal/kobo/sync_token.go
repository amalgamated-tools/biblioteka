// Package kobo provides protocol-level types and helpers for the Kobo device
// sync API. It is free of HTTP and database dependencies so that the codec and
// metadata builders can be tested and reused independently of the HTTP handler
// layer.
package kobo

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// SyncPageSize is the maximum number of books returned per Kobo sync request.
const SyncPageSize = 100

// SyncToken tracks the high-water marks used by the Kobo library sync
// protocol to avoid re-sending already-delivered books and reading-state
// updates.
type SyncToken struct {
	BooksLastModified        time.Time
	BooksLastID              string
	ReadingStateLastModified time.Time
}

type syncTokenPayload struct {
	Version string         `json:"version"`
	Data    map[string]any `json:"data"`
}

// ParseSyncToken decodes a base64-encoded Kobo sync-token header value.
// Invalid or empty input returns a zero-value SyncToken.
func ParseSyncToken(header string) SyncToken {
	if header == "" {
		return SyncToken{}
	}
	decoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return SyncToken{}
	}
	var payload syncTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return SyncToken{}
	}
	var result SyncToken
	if s, ok := payload.Data["BooksLastModified"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			result.BooksLastModified = t
		}
	}
	if s, ok := payload.Data["BooksLastID"].(string); ok {
		result.BooksLastID = s
	}
	if s, ok := payload.Data["ReadingStateLastModified"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			result.ReadingStateLastModified = t
		}
	}
	return result
}

// EncodeSyncToken serialises tok as a base64-encoded JSON payload suitable for
// the x-kobo-synctoken response header. Returns an empty string on failure so
// callers can safely omit the header.
func EncodeSyncToken(tok SyncToken) string {
	payload := syncTokenPayload{
		Version: "1-0-0",
		Data: map[string]any{
			"BooksLastModified":        tok.BooksLastModified.UTC().Format(time.RFC3339Nano),
			"BooksLastID":              tok.BooksLastID,
			"ReadingStateLastModified": tok.ReadingStateLastModified.UTC().Format(time.RFC3339Nano),
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}
