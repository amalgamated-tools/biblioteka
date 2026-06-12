package otel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/amalgamated-tools/biblioteka/internal/otel"

// StartTracer is a wrapper around oteltrace.Tracer.Start that uses the global tracer.
func StartTracer(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	globalTracer := otel.GetTracerProvider().Tracer(instrumentationName)
	return globalTracer.Start(ctx, spanName, opts...) //nolint:spancheck // span lifecycle is managed by the caller (e.g., defer span.End() in HTTP middleware)
}

// TraceMiddleware is a middleware that adds tracing to HTTP requests.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Each HTTP request gets its own trace context
		spanName := r.Method
		if r.URL != nil {
			spanName += " " + SanitizePathForTelemetry(r.URL.Path)
		}

		ctx, span := StartTracer(
			r.Context(), // Start from the request context
			spanName,
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer span.End()

		// Continue with the traced context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
