// Package otel bootstraps OpenTelemetry-based observability for the Biblioteka
// server. It configures the global slog logger (JSON or text, with optional
// source locations), initialises distributed tracing via the OTLP exporter
// when OTEL_EXPORTER_OTLP_ENDPOINT is set, and provides a TraceMiddleware that
// creates a per-request span for every HTTP handler.
package otel
