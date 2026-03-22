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

The following fields appear on HTTP access-log entries. Other subsystems (auth, background jobs, OPDS, …) emit additional fields — all field name constants are defined in [`internal/otelkeys/logger_keys.go`](../internal/otelkeys/logger_keys.go) and should be referenced by that package rather than as raw string literals (see [Log field name constants](#log-field-name-constants) below).

| Field | Type | Description |
|-------|------|-------------|
| `time` | string | RFC 3339 timestamp |
| `level` | string | `DEBUG`, `INFO`, `WARN`, or `ERROR` |
| `msg` | string | Human-readable summary of the event |
| `version` | string | Application version |
| `request_id` | string | Value generated (or forwarded) per HTTP request — normally a UUID v4; may be the placeholder `none` if UUID generation fails — correlates all log lines for a single request |
| `user_id` | string | ID of the authenticated user, if available |
| `method` | string | HTTP method |
| `url` | string | Request URL path and query string |
| `remote_addr` | string | Client IP address and port |
| `user_agent` | string | Client `User-Agent` header value |
| `status_code` | integer | HTTP response status code |
| `duration` | integer | Request duration in nanoseconds |
| `error` | string | Error message, present on `ERROR`-level entries |

> **Tip:** Set `LOG_LEVEL=debug` to enable per-request access logs.
> - `Incoming request` is logged at the start of each request and includes `method`, `url`, `remote_addr`, `user_agent`, `request_id`, and `user_id`.
> - `Request completed` is logged after the response is sent and includes `method`, `url`, `status_code`, `duration`, `request_id`, and `user_id`.
>
> At `info` level these per-request lines are suppressed, but other components (for example auth/rate-limiting and background job activity) may still emit `INFO`-level entries.

### Log field name constants

All structured-log field names used in Biblioteka are centralised in [`internal/otelkeys/logger_keys.go`](../internal/otelkeys/logger_keys.go) (package `otelkeys`). Using the constants rather than raw strings ensures that field names remain consistent across the codebase and are easy to refactor.

**For contributors:** when emitting a log entry with `slog.InfoContext` (or any context-aware variant), use the constant instead of a quoted string:

```go
// ✅ correct — uses the otelkeys constant
slog.InfoContext(ctx, "Book created", slog.String(otelkeys.BookID, book.ID))

// ❌ incorrect — raw string literal, bypasses the centralised key registry
slog.InfoContext(ctx, "Book created", slog.String("book_id", book.ID))
```

If a log entry requires a field that does not yet have a constant, add one to `internal/otelkeys/logger_keys.go` (keep the list alphabetical) before using it.

## Request ID Correlation

Every HTTP request is assigned an `X-Request-ID` value for correlation:

- If the **incoming request** already carries an `X-Request-ID` header (e.g. forwarded by a reverse proxy), that value is reused.
- Otherwise the server attempts to generate a new UUID v4; if UUID generation fails, the literal value `none` is used as a fallback.

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
docker compose logs -f --no-log-prefix biblioteka | jq 'select(.level != "DEBUG")'

# Show only errors
docker compose logs -f --no-log-prefix biblioteka | jq 'select(.level == "ERROR")'

# Find all log lines for a specific request
docker compose logs -f --no-log-prefix biblioteka | jq 'select(.request_id == "550e8400-e29b-41d4-a716-446655440000")'

# Show slow requests (> 500 ms)
docker compose logs -f --no-log-prefix biblioteka | jq 'select(.duration != null and .duration > 500000000)'
```

#### Book import troubleshooting

```bash
# Watch all background job activity (scan + process jobs)
docker compose logs -f --no-log-prefix biblioteka \
  | jq 'select(.task_id != null or (.msg | test("scan|process|enqueue"; "i")))'

# Trace a specific file through the import pipeline (use a substring of the path)
docker compose logs biblioteka \
  | jq 'select(.file_path != null and (.file_path | contains("great-gatsby")))'

# Find all failed background jobs (errors during processing)
docker compose logs biblioteka \
  | jq 'select(.level == "ERROR" and (.msg | test("scan|process|job|file"; "i")))'

# See sidecar write failures (WARN-level, best-effort — book import still succeeded)
docker compose logs biblioteka \
  | jq 'select(.level == "WARN" and (.msg | test("sidecar|cover|opf"; "i")))'

# Find all logs for a specific library scan
docker compose logs biblioteka \
  | jq 'select(.library_id == "<library-id>")'
```

## Distributed Tracing

Biblioteka includes OpenTelemetry trace context propagation (`TraceMiddleware`) that wraps every incoming HTTP request in an OTel span. The global tracer provider is used, which defaults to a **no-op provider** (no spans are exported) unless you configure an OTLP exporter before the server starts.

To emit traces, mount a custom `TracerProvider` before `server.NewServer` is called (see `internal/otel/tracing.go`). The span names follow the pattern `METHOD /path` (e.g. `GET /api/books`).

> Most deployments are well-served by structured log correlation via `request_id` alone. Distributed tracing is an advanced integration point for larger environments.

---

## Anonymous Telemetry

Biblioteka can optionally send a single anonymous boot ping to help the maintainers understand how many installations exist. Telemetry is **opt-in and disabled by default**.

### What is sent

When enabled, a one-time HTTP POST is sent to the telemetry endpoint on first boot. The payload contains no personally identifiable information:

| Field         | Example                                 | Description                                 |
|---------------|-----------------------------------------|---------------------------------------------|
| `application` | `"biblioteka"`                          | Constant identifier                         |
| `install_id`  | `"550e8400-e29b-41d4-a716-446655440000"` | Randomly generated UUID, created once and stored locally |
| `version`     | `"1.2.0"`                              | Biblioteka binary version                   |
| `os`          | `"linux"`                               | Operating system (`GOOS`)                   |
| `arch`        | `"amd64"`                               | CPU architecture (`GOARCH`)                 |
| `timestamp`   | `"2026-03-15T21:00:00Z"`               | UTC timestamp of the boot event             |

No IP addresses, hostnames, user data, library contents, or configuration values are included.

### When it fires

Telemetry is sent **at most once per install**. After the first successful transmission the install ID is written to disk (`/data/install_id` when the `/data` directory is present, otherwise `./data/install_id`). Subsequent server restarts detect this file and skip the ping.

### How to control it

| `TELEMETRY_ENABLED` value | Behaviour |
|---------------------------|-----------|
| *(unset)* | Telemetry is **disabled** (default) |
| `false` | Telemetry is explicitly disabled |
| `true` | Telemetry is enabled; ping fires once on first boot |

```bash
# Explicitly disable (same as leaving the variable unset)
TELEMETRY_ENABLED=false docker compose up

# Opt in
TELEMETRY_ENABLED=true docker compose up
```

To override the collection endpoint (e.g. for air-gapped testing):

```bash
TELEMETRY_ENDPOINT=https://my-collector.internal/boot docker compose up
```

### Log messages

When telemetry is enabled you will see the following `WARN`-level message in the startup logs:

```
NOTICE: This application collects anonymous telemetry data to help improve the product.
To disable telemetry, set the environment variable TELEMETRY_ENABLED=false
```

When the variable is unset (the default), a `WARN`-level message confirms that telemetry is disabled:

```
TELEMETRY_ENABLED environment variable not set, telemetry is disabled by default
```
