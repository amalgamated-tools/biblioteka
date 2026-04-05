package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

// RequestID is the HTTP header name used to propagate request identifiers
// between clients and the server. Incoming requests that already carry this
// header have their value preserved; new IDs are generated for requests that
// do not include it.
const RequestID = "X-Request-ID"

// RequestIDKey is the context key for the X-Request-ID value
const ctxRequestIDKey contextKey = "go-http-RequestId"

// RequestIDHandler is a middleware that generates or extracts request IDs and adds them to the request context.
func RequestIDHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestID)

		if len(id) == 0 {
			if newID, err := uuid.NewRandom(); err == nil {
				id = newID.String()
			} else {
				id = "none"
			}
		}

		r = r.WithContext(WithRequestID(r.Context(), id))

		// ensure the request ID is also available to clients in the response headers
		w.Header().Set(RequestID, id)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// GetRequestID returns the request ID if one is present.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(ctxRequestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// WithRequestID creates a new context based on the supplied parent, with the RequestId
// set to the specified value. If a RequestID already exists, it will be overwritten.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxRequestIDKey, id)
}

// Forward is a request hook that looks for X-Request-Id in the incoming context and adds it to r's headers.
func Forward(r *http.Request) {
	ctx := r.Context()

	if reqID := GetRequestID(ctx); reqID != "" {
		r.Header.Set(RequestID, reqID)
	}
}
