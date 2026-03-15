# Observability

Biblioteka emits structured logs on stdout. No sidecar, agent, or external collector is required to get started — every event is a JSON object you can ship to any log aggregator.

## Structured Logging

All log output is written to **stdout** in JSON (default) or plain-text format.

### Configuration

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `LOG_FORMAT` | `json` | `json` for machine-readable output; `text` for human-readable output during local development |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. Setting `debug` also enables source-file annotations on each log entry |

### JSON log structure

Each log line is a self-contained JSON object. Example (pretty-printed for readability):

```json
{
  "time": "2026-03-14T02:00:00.123Z",
  "level": "INFO",
  "msg": "Request completed",
  "version": "1.2.0",
  "method": "GET",
  "url": "/api/books",
  "status_code": 200,
  "duration": 3821456,
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "f47ac10b58cc4372a567b409e2087bc1"
}
```

The `version` field is always present and reflects the binary version at startup.

### Common log fields

| Field | Type | Description |
|-------|------|-------------|
| `time` | string | RFC 3339 timestamp |
| `level` | string | `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `msg` | string | Human-readable summary of the event |
| `version` | string | Application version |
| `request_id` | string | UUID generated (or forwarded) per HTTP request — correlates all log lines for a single request |
| `user_id` | string | ID of the authenticated user, if available |
| `method` | string | HTTP method |
| `url` | string | Request URL path and query string |
| `status_code` | integer | HTTP response status code |
| `duration` | integer | Request duration in nanoseconds |
| `error` | string | Error message, present on `ERROR`-level entries |

> **Tip:** Set `LOG_LEVEL=debug` to see `Incoming request` and `Request completed` lines for every HTTP request, which include all fields above. At `info` level only startup messages and errors are emitted.

## Request ID Correlation

Every HTTP request is assigned a unique `X-Request-ID` UUID:

- If the **incoming request** already carries an `X-Request-ID` header (e.g. forwarded by a reverse proxy), that value is reused.
- Otherwise a new UUID v4 is generated.

The request ID is:
1. Added to the request context so every log line emitted while handling the request carries the same `request_id` field.
2. Echoed back to the client in the `X-Request-ID` **response header**.

When reporting a bug or investigating an incident, include the `X-Request-ID` value from the response to correlate all server-side log entries for that request.

### Example: forwarding request IDs from nginx

```nginx
location / {
    proxy_pass         http://localhost:8080;
    proxy_set_header   X-Request-ID $request_id;
    # …other headers…
}
```

## Log Aggregation

Because all logs go to stdout in JSON format, they integrate with any log aggregation tool that can collect container or process stdout:

| Platform | Integration |
|----------|-------------|
| **Docker / Docker Compose** | `docker compose logs -f biblioteka` or configure a logging driver (`json-file`, `awslogs`, `fluentd`, …) |
| **Loki + Grafana** | Use the [Loki Docker driver](https://grafana.com/docs/loki/latest/send-data/docker-driver/) or [Promtail](https://grafana.com/docs/loki/latest/send-data/promtail/) to ship stdout |
| **Elasticsearch / OpenSearch** | Forward logs via Filebeat or Fluent Bit |
| **Splunk / Datadog / CloudWatch** | Point the forwarder at container stdout |

### Useful `jq` queries (local development)

```bash
# Stream all logs at info level or above
docker compose logs -f biblioteka | jq 'select(.level != "DEBUG")'

# Show only errors
docker compose logs biblioteka | jq 'select(.level == "ERROR")'

# Find all log lines for a specific request
docker compose logs biblioteka | jq 'select(.request_id == "550e8400-e29b-41d4-a716-446655440000")'

# Show slow requests (> 500 ms)
docker compose logs biblioteka | jq 'select(.duration != null and .duration > 500000000)'
```

## Distributed Tracing

Biblioteka includes OpenTelemetry trace context propagation (`TraceMiddleware`) that wraps every incoming HTTP request in an OTel span. The global tracer provider is used, which defaults to a **no-op provider** (no spans are exported) unless you configure an OTLP exporter before the server starts.

To emit traces, mount a custom `TracerProvider` before `server.NewServer` is called (see `internal/otel/tracing.go`). The span names follow the pattern `METHOD /path` (e.g. `GET /api/books`).

> Most deployments are well-served by structured log correlation via `request_id` alone. Distributed tracing is an advanced integration point for larger environments.
