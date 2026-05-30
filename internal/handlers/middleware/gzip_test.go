package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware_CompressesJSON(t *testing.T) {
	body := strings.Repeat(`{"key":"value"}`, 100)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, body)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	require.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
	require.NotEmpty(t, rec.Body.Bytes())

	// Decompress and verify round-trip.
	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	got, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	require.Equal(t, body, string(got))
}

func TestGzipMiddleware_NoGzipHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(w, `{"ok":true}`)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	// No Accept-Encoding: gzip header.
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Empty(t, rec.Header().Get("Content-Encoding"))
	require.Equal(t, `{"ok":true}`, rec.Body.String())
}

func TestGzipMiddleware_VaryAlwaysSet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/books/1", nil)
	// No Accept-Encoding.
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	// Vary must be set even when the client did not request gzip.
	require.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
}

func TestGzipMiddleware_SkipsImage(t *testing.T) {
	binary := []byte{0x89, 'P', 'N', 'G'}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, err := w.Write(binary)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/books/1/cover", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Empty(t, rec.Header().Get("Content-Encoding"), "binary images must not be compressed")
	require.Equal(t, binary, rec.Body.Bytes())
}

func TestGzipMiddleware_SkipsSSE(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher := w.(http.Flusher)
		flusher.Flush()
		_, err := io.WriteString(w, "data: {\"event\":\"complete\"}\n\n")
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/books/1/metadata/events", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Empty(t, rec.Header().Get("Content-Encoding"), "SSE responses must not be compressed")
}

func TestGzipMiddleware_CompressesOPDSFeed(t *testing.T) {
	body := strings.Repeat("<feed xmlns='http://www.w3.org/2005/Atom'></feed>", 50)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		_, err := io.WriteString(w, body)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/opds/v2.0/catalog", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	got, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	require.Equal(t, body, string(got))
}

func TestGzipMiddleware_ExplicitWriteHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err := io.WriteString(w, `{"id":"123"}`)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/books", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	got, err := io.ReadAll(gr)
	require.NoError(t, err)
	require.NoError(t, gr.Close())
	require.Equal(t, `{"id":"123"}`, string(got))
}

func TestShouldGzip(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/atom+xml", true},
		{"text/html; charset=utf-8", true},
		{"text/plain", true},
		{"image/png", false},
		{"image/jpeg", false},
		{"image/webp", false},
		{"image/svg+xml", true},
		{"text/event-stream", false},
		{"application/epub+zip", false},
		{"application/octet-stream", false},
		{"application/pdf", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			require.Equal(t, tt.want, shouldGzip(tt.ct))
		})
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"gzip", true},
		{"gzip, deflate, br", true},
		{"br, gzip", true},
		{"deflate", false},
		{"", false},
		{"gzip;q=1.0", true},
		{"gzip;q=0.5", true},
		{"gzip;q=0", false},
		{"br, gzip;q=0", false},
		{"gzip;q=0.0", false}, // q=0.0 is zero quality (not acceptable)
		{"gzip;q=0.00", false},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			require.Equal(t, tt.want, acceptsGzip(tt.header))
		})
	}
}

func TestGzipMiddleware_SkipsRangeRequests(t *testing.T) {
	body := strings.Repeat(`{"key":"value"}`, 100)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPartialContent)
		_, err := io.WriteString(w, body)
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/books/1/file", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Range", "bytes=0-1023")
	rec := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rec, req)

	require.Empty(t, rec.Header().Get("Content-Encoding"), "Range requests must not be compressed")
	require.Equal(t, body, rec.Body.String())
}
