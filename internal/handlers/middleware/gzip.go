package middleware

import (
	"compress/gzip"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// compressibleTypes is the set of MIME types for which gzip compression is
// applied. Only text-based formats benefit from compression; binary formats
// (images, epub+zip, mobi, pdf) and streaming responses (SSE) are excluded.
var compressibleTypes = map[string]bool{
	"application/json":       true,
	"application/xml":        true,
	"application/atom+xml":   true, // OPDS feeds
	"application/javascript": true,
	"text/css":               true,
	"text/html":              true,
	"text/javascript":        true,
	"text/plain":             true,
	"text/xml":               true,
	"image/svg+xml":          true,
}

// gzipPool reuses gzip.Writer instances across requests to avoid the
// per-request allocation cost of gzip.NewWriter.
var gzipPool = sync.Pool{
	New: func() any {
		gz, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return gz
	},
}

// gzipResponseWriter wraps http.ResponseWriter to gzip-compress the response
// body when the response Content-Type is in the compressible set.
//
// Compression is decided lazily at the time of the first write or Flush call:
// this ensures that the Content-Type header — set by the handler before any
// write — is visible when the decision is made.
//
// For SSE and binary responses the writer passes through to the underlying
// ResponseWriter without compression.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz   *gzip.Writer
	once sync.Once
}

// decide initialises the gzip writer (or decides to skip compression) by
// inspecting the Content-Type header. It is idempotent via sync.Once.
func (grw *gzipResponseWriter) decide() {
	grw.once.Do(func() {
		ct := grw.ResponseWriter.Header().Get("Content-Type")
		if shouldGzip(ct) {
			gz := gzipPool.Get().(*gzip.Writer)
			gz.Reset(grw.ResponseWriter)
			grw.gz = gz
			grw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
			// Remove Content-Length: the compressed size is not known in advance.
			grw.ResponseWriter.Header().Del("Content-Length")
		}
	})
}

// Unwrap allows http.ResponseController to reach the underlying ResponseWriter
// for operations such as SetWriteDeadline (used by the SSE handler).
func (grw *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return grw.ResponseWriter
}

// WriteHeader decides on compression, then delegates to the underlying writer.
func (grw *gzipResponseWriter) WriteHeader(status int) {
	grw.decide()
	grw.ResponseWriter.WriteHeader(status)
}

// Write decides on compression (if not already decided), then writes to either
// the gzip writer or the underlying ResponseWriter directly.
func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	grw.decide()
	if grw.gz != nil {
		return grw.gz.Write(b)
	}
	return grw.ResponseWriter.Write(b)
}

// Flush decides on compression, flushes the gzip stream if active, then
// flushes the underlying ResponseWriter. This satisfies http.Flusher for SSE.
func (grw *gzipResponseWriter) Flush() {
	grw.decide()
	if grw.gz != nil {
		_ = grw.gz.Flush()
	}
	if f, ok := grw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// shouldGzip reports whether the given Content-Type value warrants gzip
// compression. Parameters such as "; charset=utf-8" are stripped before
// the lookup.
func shouldGzip(contentType string) bool {
	ct := strings.ToLower(contentType)
	if idx := strings.IndexByte(ct, ';'); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	return compressibleTypes[ct]
}

// GzipMiddleware is an HTTP middleware that transparently gzip-compresses
// responses when:
//  1. The client signals support via "Accept-Encoding: gzip".
//  2. The response Content-Type is a known text-based format.
//
// Binary responses (images, book files) and streaming responses (SSE) are
// passed through uncompressed. The middleware adds a "Vary: Accept-Encoding"
// header to all responses so that caches correctly serve different encodings
// to different clients.
//
// A sync.Pool is used to reuse gzip.Writer instances, reducing per-request
// allocation overhead.
// acceptsGzip reports whether the client's Accept-Encoding header indicates
// support for gzip. It correctly handles quality values, treating q=0 as an
// explicit rejection of gzip.
func acceptsGzip(header string) bool {
	for _, token := range strings.Split(header, ",") {
		token = strings.TrimSpace(token)
		if strings.HasPrefix(strings.ToLower(token), "gzip") {
			// Reject explicit q=0 (but not q=0.5, q=0.1, etc.).
			if strings.Contains(token, "q=0") && !strings.Contains(token, "q=0.") {
				return false
			}
			return true
		}
	}
	return false
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always advertise that the response varies by Accept-Encoding so
		// that intermediate caches store separate entries per encoding.
		w.Header().Add("Vary", "Accept-Encoding")

		// Skip compression for Range requests to preserve Content-Range semantics.
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
			next.ServeHTTP(w, r)
			return
		}

		grw := &gzipResponseWriter{ResponseWriter: w}
		defer func() {
			if grw.gz != nil {
				if err := grw.gz.Close(); err != nil {
					// The gzip trailer could not be written; the client will
					// receive a truncated stream. Log at debug level because
					// client disconnects are the most common cause.
					slog.DebugContext(r.Context(), "gzip close error", slog.Any(otelkeys.Error, err))
				}
				grw.gz.Reset(nil)
				gzipPool.Put(grw.gz)
				grw.gz = nil
			}
		}()

		next.ServeHTTP(grw, r)
	})
}
