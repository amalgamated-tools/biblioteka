# Background Jobs

Biblioteka uses [asynq](https://github.com/hibiken/asynq), a Redis-backed task queue for Go, to run background jobs such as scanning library paths and importing book files. A built-in scheduler triggers periodic scans, and a web dashboard lets admins inspect queues and retry failures.

## Architecture Overview

```
┌───────────────────────────┐        ┌─────────────────────────┐
│   HTTP Server             │        │    Redis                │
│                           │        │                         │
│  POST /api/libraries ─────┼──────▶ │  "critical" queue (×6)  │
│  POST /api/books ─────────┼──────▶ │  "default"  queue (×3)  │
│  Scheduled (every 24 h) ──┼──────▶ │  "low"      queue (×1)  │
│  Scheduled (every 1 m) ───┼──────▶ │                         │
└───────────────────────────┘        └───────────┬─────────────┘
                                             │
                                     ┌───────▼───────────┐
                                     │  asynq Worker      │
                                     │  (4 goroutines)    │
                                     │                    │
                                     │  scan:libraries    │
                                     │  scan:library      │
                                     │  scan:path         │
                                     │  process:file      │
                                     │  enrich:goodreads  │
                                     │  enrich:ai         │
                                     │  scan:watch-folder │
                                     └────────────────────┘
```

The HTTP server and the asynq worker can run in the same process (the default `all` mode) or as separate processes using the `-mode` flag. Both modes use the same Redis instance (via `REDIS_URL`) but create their own Redis connections. See [Run Modes](../README.md#run-modes) for details.

## Prerequisites

| Dependency | Version | Purpose |
|------------|---------|---------|
| Redis      | 7+      | Job queue storage and scheduling |

Set the `REDIS_URL` environment variable to point at your Redis instance. The default is `redis://localhost:6379`. Standard Redis URL formats are supported (e.g. `redis://:password@host:6379/0`, `rediss://host:6379` for TLS).

## Job Catalog

### `scan:libraries`

| | |
|---|---|
| **Source** | `internal/jobs/scan_libraries.go` — `NewScanLibrariesHandler` |
| **Trigger** | Scheduled every 24 hours |
| **Payload** | _none (empty struct)_ |

Fetches all libraries from the database, filters to those with `monitored = true` and at least one configured path, and enqueues a `scan:library` job for each.

### `scan:library`

| | |
|---|---|
| **Source** | `internal/jobs/scan_library.go` — `NewScanLibraryHandler` |
| **Trigger** | Enqueued by `scan:libraries`, or immediately when a library is created via the API |
| **Payload** | `{ "library_id": "<uuid>", "paths": ["/books", "/more-books"] }` |

Enqueues a `scan:path` job for every path in the library.

### `scan:path`

| | |
|---|---|
| **Source** | `internal/jobs/scan_path.go` — `NewScanPathHandler` |
| **Trigger** | Enqueued by `scan:library` |
| **Payload** | `{ "path": "/books", "library_id": "<uuid>", "library_root": "/books" }` |

`NewScanPathHandler` parses the JSON payload and delegates to `ScanDirectory` (`internal/jobs/scan_directory.go`), which recursively walks the given directory and enqueues a `process:file` job for every file with a supported extension (`.epub`, `.mobi`, `.pdf`, `.azw3`). Inaccessible files are logged as warnings and skipped.

The `library_id` and `library_root` fields are optional, but `scan:library` always populates them. When present, they are forwarded verbatim to each `process:file` payload so the file handler can (a) associate the book with the correct library and (b) derive author, title, and series information from the file's path relative to the library root (see [Path-based metadata](#path-based-metadata) below).

### `process:file`

| | |
|---|---|
| **Source** | `internal/jobs/process_file.go` — `NewProcessFileHandler` |
| **Trigger** | Enqueued by `scan:path` |
| **Payload** | `{ "path": "/books/novel.epub", "file_name": "novel.epub", "file_type": "epub", "file_size": 524288, "library_id": "<uuid>", "library_root": "/books" }` |

The `library_id` and `library_root` fields are optional. When `library_root` is present, the handler also derives metadata from the file's directory structure (see [Path-based metadata](#path-based-metadata) below).

Creates a `book` record and a `book_file` record in the database. The `process:file` handler runs `internal/metadata.Extractor.ExtractMetadata` on every imported file and uses the result to populate the book record:

| Extracted field | Stored as | Notes |
|----------------|-----------|-------|
| `Title` | `books.title` | Falls back to path-derived title, then filename without extension |
| `ISBN` (10 or 13 digits) | `books.isbn_10` / `books.isbn_13` | Not stored when absent |
| `Description` (when non-empty) | `books.description` | Availability depends on format; see [metadata.md](metadata.md) |
| `Publisher` (when non-empty) | `books.publisher` | Availability depends on format; see [metadata.md](metadata.md) |
| `Language` (when non-empty) | `books.language` | Availability depends on format; see [metadata.md](metadata.md) |
| `PublicationDate` (when non-empty) | `books.publication_date` | ExifTool's `YYYY:MM:DD` format is normalized to `YYYY-MM-DD`; availability depends on format |
| `Author` (when non-empty) | `authors` + `book_authors` join | The extracted name is whitespace-normalized (trimmed, internal runs collapsed). An existing author is looked up **case-insensitively** (`"J.R.R. Tolkien"` and `"j.r.r. tolkien"` match the same record). If no match is found, a new author record is created. If a concurrent worker creates the same author first, the handler retries the lookup rather than failing. If association fails after the book record is already committed, the failure is logged as a warning and does **not** fail the job (preventing duplicate book records on retry). Falls back to path-derived author when the embedded value is absent or `"Unknown"`. |
| `CoverImageURL` (when non-empty) | `books.cover_image_url` | For EPUB files, automatically extracted from the embedded cover in the archive and stored as a base64 `data:` URL. For other formats (PDF, MOBI, AZW3), cover extraction is not yet automatic; the field remains `null` unless a URL is set manually via `PUT /api/books/{id}`. |

`Format` is extracted but stored on the `book_files` record via the `file_type` payload field, not from the extractor output directly. When ExifTool is absent, the job logs at `DEBUG` level and falls back to the filename-derived title; other extraction failures are logged at `WARN` level. See [docs/metadata.md](metadata.md) for extraction details. Use the standalone [`cmd/cli`](../README.md#cli-tool) tool to inspect metadata from individual files.

#### Path-based metadata

When `library_root` is set in the payload, `internal/pathparser.ParseBookPath` extracts author, title, series name, and series position from the file's path relative to the library root. This runs for every file regardless of whether ExifTool is present.

> **Publication year note:** The parser also detects a trailing `(YYYY)` year in filenames and uses it to produce a clean title (e.g. `Frankenstein (2009)` → `Frankenstein`). The year value itself is **not** stored in `books.publication_date`; that field is only populated from embedded file metadata extracted by ExifTool.

**Recognised directory layouts**

| Pattern | Author | Title | Series |
|---------|--------|-------|--------|
| `Author/Title.ext` | `Author` dir | filename (stripped) | — |
| `Author/N. Title (Year).ext` | `Author` dir | filename (stripped) | — |
| `Author/Series/N. Title (Year).ext` | `Author` dir | filename (stripped) | `Series` dir |
| `Author - Title/file.ext` | left of ` - ` | right of ` - ` | — |
| `Author - Title.ext` | left of ` - ` | right of ` - ` | — |

Key rules applied by the parser:
- **Leading series-position prefix** (`N. `) is stripped from the filename to produce the bare title.
- **Trailing `(YYYY)`** is detected and removed from the title (e.g. `The Hobbit (1937)` → `The Hobbit`). The year is not stored as `publication_date`.
- **Trailing ` - Author Name`** in filenames is stripped when the suffix looks like a personal name: two to four words, every word starting with an uppercase letter, no digits, and not beginning with a common English article (`a`, `an`, `the`). This prevents subtitle fragments like `A Novel`, `An Unabridged`, or `The Remake` from being mistakenly removed. Single-word suffixes (e.g. `Unabridged`) are always kept.
- A directory is treated as a **series** only when the filename has a leading series-position prefix **and** the directory name is not effectively the same as the parsed title (to avoid phantom series from Calibre-style `Author/Title/file.ext` layouts). The comparison is case-insensitive and ignores all non-alphanumeric characters, so `Star Wars: Episode IV` and `Star Wars Episode IV` are considered identical.
- Paths outside `library_root` fall back to filename-only parsing.

**Metadata precedence** (highest to lowest):

1. Embedded file metadata (ExifTool) — used when present and non-empty
2. Path-derived metadata (`pathparser`) — fills gaps left by step 1 (author falls back when value is absent or `"Unknown"`)
3. Filename without extension — last resort for title

#### File reorganization

When `library_root` is set and the library's `organization_type` is `book_per_folder` or `book_per_file`, the handler moves the file into the corresponding directory structure under the library root after resolving the author name and title:

- `book_per_folder`: `<library_root>/<Author>/<Title>/<filename>` — requires both author **and** title; reorganization is silently skipped when either is absent.
- `book_per_file`: `<library_root>/<Author>/<filename>` — requires only the author; title is not needed.

Directory names are sanitized (path separators, control characters, and leading dots removed). As a defense-in-depth measure, the handler also verifies that the computed target path stays within `library_root` after sanitization, guarding against path traversal through untrusted author/title metadata embedded in book files. The move uses `os.Rename` when source and destination are on the same filesystem, falling back to copy-then-delete for cross-filesystem moves; source file permissions and modification time are preserved. Intermediate empty source directories are removed after a successful move.

**Failure handling:**
- If reorganization fails for most reasons (e.g. permissions, cross-device copy errors), the handler logs a warning, cleans up any partial copy created at the destination, and continues processing the file at its original path.
- If the source file disappears *during* reorganization (i.e. `os.IsNotExist` is returned by the move/copy step), the handler checks whether the cross-filesystem copy already landed at `newPath`. If it did (copy succeeded but the source-remove failed), the handler continues processing from `newPath`. If no successful copy exists, the job returns a hard error so that asynq can retry it rather than committing a `book_files` row pointing at a non-existent path.
- If the source file is missing when the job starts, the handler first checks whether the original path is already indexed in the database. If it is, the job skips without error (and, if a `library_id` is present, attempts to link the book to that library). If the original path is not in the database, the handler then tries to locate the file at its expected reorganized path: `<library_root>/<Author>/<Title>/<filename>` for `book_per_folder`, or `<library_root>/<Author>/<filename>` for `book_per_file`. If found there and already indexed, the job skips without error. If found there but not yet indexed, the handler resumes processing from that new location. If the file cannot be found at any expected location, the job logs a warning and returns without error. A database error (other than "not found") at any of these lookup steps is propagated as a hard error so that asynq retries the job rather than silently dropping it.
- After a successful move, the handler checks whether the new path is already indexed before creating new database records, preventing duplicates from concurrent workers.

See [Administration — File organization](administration.md#file-organization) for details on configuring this feature per-library.

#### Sidecar files

After a book record is created (and the file has been optionally reorganized), the handler calls `internal/sidecar.WriteSidecarFiles`, which writes two companion files alongside the book file:

| File | Condition | Description |
|------|-----------|-------------|
| `cover.<ext>` | Only when `CoverImageURL` is non-empty | Cover image decoded from the stored `data:` URL |
| `metadata.opf` | Always | OPF 2.0 file with Dublin Core metadata |

**Sidecar filenames and `book_per_file` mode**

In `book_per_folder` and `none` libraries, sidecar files use the default names shown above (`cover.<ext>` and `metadata.opf`). In `book_per_file` libraries, multiple books share the same author directory, so sidecar files are prefixed with the **book's filename stem** to prevent collisions:

| Library `organization_type` | Cover filename | OPF filename |
|-----------------------------|----------------|--------------|
| `book_per_folder` (default) | `cover.<ext>` | `metadata.opf` |
| `none` | `cover.<ext>` | `metadata.opf` |
| `book_per_file` | `<book-stem>.<ext>` | `<book-stem>.opf` |

For example, a book file named `the-great-gatsby.epub` in a `book_per_file` library produces `the-great-gatsby.jpg` and `the-great-gatsby.opf` alongside it.

**Cover image**

The cover is decoded from the base64 `data:` URL stored in `books.cover_image_url` (populated during extraction for EPUB files). The file extension is determined by the decoded MIME type:

| MIME type | Output filename (`book_per_folder`/`none`) | Output filename (`book_per_file`) |
|-----------|-------------------------------------------|------------------------------------|
| `image/jpeg` | `cover.jpg` | `<book-stem>.jpg` |
| `image/png` | `cover.png` | `<book-stem>.png` |
| `image/webp` | `cover.webp` | `<book-stem>.webp` |
| `image/avif` | `cover.avif` | `<book-stem>.avif` |

Cover files are written **atomically**: the new image is first written to a temporary file (`cover.<ext>.tmp`), then renamed into its final position. Only after the rename succeeds are stale cover files of other formats removed (best-effort cleanup to avoid orphaned files when cover formats change). This ensures the directory always contains either the previous cover or the new one — never nothing — even if the process is interrupted mid-write.

Cover data URLs are capped at **20 MB** of decoded bytes; inputs exceeding this limit are rejected with a warning and no cover file is written.

**OPF metadata file (`metadata.opf` or `<book-stem>.opf`)**

`metadata.opf` (or `<book-stem>.opf` in `book_per_file` mode) is an [OPF 2.0](https://idpf.org/epub/20/spec/OPF_2.0.1_draft.htm) file suitable for use by e-reader applications (Kobo, KOReader, and others). It contains Dublin Core metadata:

| OPF field | Source | Notes |
|-----------|--------|-------|
| `dc:title` | Book title | Required; `WriteOPF` returns an error when empty |
| `dc:creator` | Author name | `opf:role="aut"` attribute included |
| `dc:identifier` | `books.isbn_10` or `books.isbn_13` | Falls back to a deterministic UUID v5 derived from title, author, publisher, publication date, and file path when no ISBN is present; scheme is `ISBN` or `UUID` accordingly. UUID values use the `urn:uuid:` URN prefix required by OPF 2.0 §2.2.10 (e.g. `urn:uuid:a5d3b2e1-7f4c-4e8a-9d6b-1c2e3f4a5b6d`) so that strict EPUB validators and importers such as Calibre accept the identifier |
| `dc:language` | `books.language` | Defaults to `"und"` (undetermined) when absent |
| `dc:date` | `books.publication_date` | Omitted when empty |
| `dc:publisher` | `books.publisher` | Omitted when empty |
| `dc:description` | `books.description` | Omitted when empty |
| `<meta name="cover">` | Present when cover file was written | Points to `cover-image` manifest item |
| `<manifest>` | Present when cover file was written | Lists the cover image with its MIME type |

**Failure handling:** both the cover write and the OPF write are best-effort. A failure in either step is logged at `WARN` level and does **not** fail the `process:file` job. The book record committed to the database is not affected.

### `enrich:goodreads`

| | |
|---|---|
| **Source** | `internal/jobs/enrich_goodreads.go` — `NewEnrichGoodreadsHandler` |
| **Trigger** | Enqueued by the API immediately after a book is created via `POST /api/books` |
| **Payload** | `{ "book_id": "<uuid>", "user_id": "<uuid>" }` |

Fetches Goodreads metadata for a newly imported book and stores the result as a `goodreads_metadata` record with `status = "pending"` for user review. The job tries the following lookup strategies in order, stopping at the first successful match:

| Priority | Strategy | Condition |
|----------|-----------|-----------|
| 1 (highest) | ISBN-13 lookup | `books.isbn13` is non-empty |
| 2 | ISBN-10 lookup | `books.isbn10` is non-empty |
| 3 | ASIN lookup | `books.asin` is non-empty |
| 4 | Goodreads ID lookup | `books.goodreads_id` is non-empty |
| 5 (lowest) | Title search | `books.title` is non-empty |

When no match is found, the job exits cleanly with no record created. A failed Goodreads API call at one strategy level is logged at `WARN` level, and the job falls through to the next strategy rather than failing immediately.

The resulting `goodreads_metadata` record stores the Goodreads book title, identifiers (Goodreads ID, ASIN, ISBN, ISBN-13, work ID, and legacy integer IDs), author information (name, Goodreads author ID, legacy author ID, and profile image URL), language, and cover image URL. The `description` field is not populated (Goodreads does not return it via the search/lookup APIs used). The `hardcover_id`, `google_books_id`, `publication_date`, and `publisher` fields are also left unpopulated — the Goodreads search/lookup APIs used do not surface them. The user must review and apply the record via the metadata review UI before any data is merged into the book record.

**Failure handling:** if the Goodreads client call or the database write fails, the error is returned and asynq retries the job up to `DefaultMaxRetry` (5) times with exponential back-off. A failed enrichment never blocks or modifies the book record already committed by `POST /api/books`.

### `enrich:ai`

| | |
|---|---|
| **Source** | `internal/jobs/enrich_ai.go` — `NewEnrichAIHandler` |
| **Trigger** | Enqueued on demand via `POST /api/books/{id}/metadata/ai-fetch` |
| **Payload** | `{ "book_id": "<uuid>", "user_id": "<uuid>" }` |

Calls the configured LLM provider to generate metadata for a book and stores the result as an `ai_enrichments` record with `status = "pending"` for user review. The job executes the following steps in order:

1. Fetches the book record and its associated author names from the database.
2. Builds a structured prompt via `llm.BuildEnrichPrompt` using the book's title, author list, and existing description.
3. Sends the prompt to the LLM provider (currently Ollama at `/api/chat` with `stream: false`).
4. Parses the raw response with `llm.ParseEnrichmentResult`, which strips optional markdown code fences before unmarshalling JSON.
5. Writes the `ai_enrichments` record to the database with `status = "pending"`.

Real-time progress events (`fetching_book`, `building_prompt`, `generating`) are published to the Redis pub/sub channel for the book and user, making them visible in the SSE stream at `GET /api/books/{id}/metadata/events`.

The pending candidate must be explicitly reviewed and either applied (via `POST /api/books/{id}/metadata/ai-apply`) or rejected (via `POST /api/books/{id}/metadata/ai-reject`) before any changes are written to the book record.

**Enrichment result fields stored in `ai_enrichments`:**

| Field | Description |
|-------|-------------|
| `suggested_tags` | 5–10 concise tags for library cataloging |
| `reading_level` | One of `"children"`, `"young_adult"`, `"adult"`, `"academic"` |
| `generated_description` | 2–3 sentence catalog description |
| `raw_response` | Verbatim LLM response (stored for debugging and auditing) |

**Failure handling:** LLM generation errors and parse failures are logged at `ERROR` level, an error progress event is published, and the error is returned so asynq retries the job up to `DefaultMaxRetry` (5) times with exponential back-off. When the LLM provider is `nil` (not configured at startup), the job publishes an error event and returns `nil` — asynq does not retry it.

> **Note:** AI enrichment requires a running Redis worker and a configured LLM provider. See [`PUT /api/config/llm`](api/config.md#put-apiconfigllm--admin--jwt-only) for configuration details.

### `scan:watch-folder`

| | |
|---|---|
| **Source** | `internal/jobs/scan_watch_folder.go` — `NewScanWatchFolderHandler` |
| **Trigger** | Scheduled every 1 minute (`@every 1m`) |
| **Payload** | _none (empty struct)_ |

Reads the `watch_folder_path` and `watch_folder_library_id` settings from the database. If both are configured and non-empty, delegates to `ScanDirectory` to walk the watch folder and enqueue a `process:file` job for each supported file found (`.epub`, `.mobi`, `.pdf`, `.azw3`).

The watch folder path is used as the scan root. `library_root` is intentionally left empty, so watch-folder imports keep files in their original location and `process:file` does not reorganize them into the library's configured storage path.

If the watch folder is not configured (no `watch_folder_path` setting, or the path is empty), the job exits cleanly with a debug log and no work is performed. If the library ID is missing, the job logs a warning and exits — scanning without a target library would produce orphaned book records.

**Failure handling:** a failure to read settings or walk the directory is returned and asynq retries the job up to `DefaultMaxRetry` (5) times. Individual file-enqueue failures follow the same deduplication and error-handling semantics as `scan:path` (see above).

> **Note:** The watch folder feature requires a running Redis worker. When the server runs in `server`-only mode (no worker), the scheduler does not start and the watch folder is never polled. See [Watch Folder](administration.md#watch-folder) in the Administration Guide for configuration details.

### Job Chain

A full scan flows through the jobs in a fan-out pattern:

```
scan:libraries
 └─▶ scan:library  (one per monitored library)
      └─▶ scan:path  (one per library path)
           └─▶ process:file  (one per supported file found)
```

Book creation also triggers a parallel enrichment job:

```
POST /api/books
 └─▶ enrich:goodreads  (one per created book)
```

AI enrichment is triggered on demand by the API:

```
POST /api/books/{id}/metadata/ai-fetch
 └─▶ enrich:ai  (one per request, deduplicated while a pending record exists)
```

The watch folder runs on its own schedule:

```
scheduler (@every 1m)
 └─▶ scan:watch-folder
      └─▶ process:file  (one per supported file found in the watch folder)
```

## Scheduling

Periodic jobs are registered with the asynq scheduler at startup in `cmd/server/main.go`, **but only when the process is running in `worker` or `all` mode**:

| Job | Schedule | Description |
|-----|----------|-------------|
| `scan:libraries` | `@every 24h` | Triggers a full scan of all monitored libraries |
| `scan:watch-folder` | `@every 1m` | Polls the configured watch folder for new book files |

Both schedules are registered using `w.RegisterSchedule`. The cron specification follows the format accepted by asynq — both classic cron expressions (`0 3 * * *`) and convenience shortcuts (`@every 24h`, `@daily`, `@every 1m`) are supported.

## Worker Configuration

Configuration lives in `internal/worker/worker.go` as package-level constants:

| Constant | Value | Meaning |
|----------|-------|---------|
| `QueueCritical` | `"critical"` | High-priority queue (weight 6) — opt in via `jobs.WithQueue(worker.QueueCritical)` |
| `QueueName` | `"default"` | Normal-priority queue (weight 3) — scan jobs and enrichment |
| `QueueLow` | `"low"` | Low-priority queue (weight 1) — scheduled background maintenance |
| `DefaultConcurrency` | `4` | Maximum number of jobs executing in parallel |
| `DefaultMaxRetry` | `5` | How many times a failed job is retried |
| `DefaultShutdownTimeout` | `8s` | Time allowed for in-flight jobs to complete on shutdown |

The worker uses **weighted priority queues**: jobs in the `critical` queue are processed 6× more often than `low` jobs. Scheduled root triggers (`scan:libraries`, `scan:watch-folder`) are placed on the `low` queue; all other jobs use `default` unless explicitly overridden with `jobs.WithQueue`.

Deduplication is **opt-in** via `jobs.WithUnique(d)`. Callers that need deduplication (e.g., scan jobs) pass this option explicitly; user-triggered metadata fetches intentionally omit it so users can always re-run them.

Worker logs are emitted through the application's structured `slog` logger (JSON, including OTEL fields) rather than asynq's default stdout logger. The worker's `BaseContext` is set to the root application context, so tracer and logger values are available in all job handlers. OpenTelemetry trace context (W3C `traceparent`) and the request ID are propagated across the queue boundary via asynq v0.26.0 task headers, restoring end-to-end traces.

On shutdown, the worker calls `srv.Stop()` (stops dequeueing new tasks) before `srv.Shutdown()` (waits for in-flight tasks), giving running jobs up to `DefaultShutdownTimeout` (8 seconds) to complete cleanly.

### Operational safety nets

Two built-in mechanisms protect against silent failures at the worker level:

- **Redis health-check logging** — The worker configures a `HealthCheckFunc` that emits a structured `WARN`-level log entry (`"Redis health check failed"`) whenever asynq detects that the Redis connection has become temporarily unreachable. The log includes the underlying error. This makes transient Redis connectivity issues visible in structured log streams without requiring an external health probe.

- **Unknown task-type logging** — A `notFoundLoggingMiddleware` wraps the asynq handler mux. When a task arrives in the queue but no handler is registered for its type (e.g. a job payload queued by a newer binary version against an older worker), the middleware detects `asynq.ErrHandlerNotFound` and emits a `WARN`-level log entry (`"no handler registered for job type"`) with the task type. The task is still returned to the queue with an error so asynq can retry or move it to the **Dead** queue.

## How Jobs Are Enqueued

Jobs enter the queue in two ways:

1. **API-triggered** — Two handlers enqueue jobs on demand:
   - When a user creates a library via `POST /api/libraries` and the library has paths, the handler immediately enqueues a `scan:library` job via `Worker.Enqueue` with `jobs.WithUnique(24*time.Hour)` (see `internal/handlers/libraries.go`).
   - When a book is created via `POST /api/books`, the handler immediately enqueues an `enrich:goodreads` job via `Worker.Enqueue` with `jobs.WithUnique(24*time.Hour)` (see `internal/handlers/book_crud.go`). A failure to enqueue is logged at `WARN` level and does not fail the book-creation response.
   - Metadata fetch endpoints (`POST /api/books/{id}/metadata/{fetch|ai-fetch}`) enqueue enrichment jobs **without** `WithUnique`, so users can always re-trigger a metadata refresh regardless of whether the job was already queued.

   Scan-related jobs use the 24-hour deduplication window described in [Deduplication](#deduplication); metadata fetch jobs do not.

2. **Scheduled** — The asynq scheduler fires two periodic jobs:
   - `scan:libraries` every 24 hours — the trigger is issued directly by the asynq scheduler (not through `Worker.Enqueue`) and carries no deduplication. When the handler runs, it calls `Worker.Enqueue` to create `scan:library` jobs, which cascade into `scan:path` and `process:file` jobs — all of which go through `Worker.Enqueue` and benefit from the 24-hour deduplication window.
   - `scan:watch-folder` every 1 minute — the trigger is also issued directly by the asynq scheduler. The handler calls `ScanDirectory`, which enqueues `process:file` jobs through `Worker.Enqueue`.

API-triggered scan jobs call `Worker.Enqueue`, which serializes the payload to JSON, injects W3C trace context and the current request ID into task headers, and pushes an asynq task onto the appropriate queue. The root scheduled triggers (`scan:libraries` and `scan:watch-folder`) are created directly by the asynq scheduler and do not go through `Worker.Enqueue`.

### Deduplication

The `book_files` table has a `UNIQUE` constraint on `file_path` (`idx_book_files_file_path`). If a `process:file` job tries to insert a `book_file` row for a path that is already indexed, the database rejects the insert and the handler skips creating a duplicate record. The handler also proactively checks whether the target path is already indexed before creating database records — both at the start of the job (when the original path already exists in the database) and after a file reorganization (in case a concurrent worker processed the same file first).

Scan jobs additionally use a **24-hour asynq deduplication window** via the `jobs.WithUnique(24*time.Hour)` option, which prevents the same job payload from being enqueued more than once within that window. Metadata fetch jobs (`enrich:goodreads`, `enrich:ai` triggered by user request) do **not** use the unique option so that users can always re-trigger a fetch without waiting for any deduplication window to expire.

**Remaining edge cases:**

- **Files reachable from multiple paths** — If the same physical file is reachable under two different library paths (e.g. via symlinks or overlapping mounts), each path produces a distinct job payload with a different `file_path`. Both jobs can succeed and create separate `book_file` rows pointing at different paths on disk.
- **Redis data loss** — If the Redis store is cleared or the server is restarted against a fresh Redis instance, the 24-hour deduplication window is reset. A subsequent scan may re-enqueue jobs for files already indexed; however, the database-level `UNIQUE(file_path)` constraint prevents new duplicate `book_file` rows from being created for files that have not moved.
- **Watch folder + Redis restart** — Because `scan:watch-folder` runs every 1 minute and its `process:file` jobs go through `Worker.Enqueue`, clearing Redis resets the deduplication window for all watch-folder enqueues. On the next 1-minute poll, the watch-folder handler will re-enqueue `process:file` jobs for every supported file in the watch folder. The burst of enqueue attempts is expected behavior — not a sign of malfunction — and the database `UNIQUE(file_path)` constraint prevents duplicate book records from being created for files that are already indexed.

## Monitoring Dashboard (Asynqmon)

The [Asynqmon](https://github.com/hibiken/asynqmon) web UI is mounted at **`/asynqmon/`** and is restricted to admin users.

**Authentication options:**

| Method | How |
|--------|-----|
| Browser | Sign in as an admin user through the web UI, then navigate to `/asynqmon/` — the session cookie (`biblioteka_token`) is sent automatically. |
| API client | Include an `Authorization: Bearer <JWT>` header with an admin-level token. |

The dashboard lets you:

- View pending, active, completed, and failed jobs
- Inspect job payloads and error messages
- Retry or delete failed jobs
- See queue statistics and throughput

**Common failure causes:**

| Cause | What you see in Asynqmon | Details |
|-------|--------------------------|---------|
| Library deleted during scan | No Failed entry | If a library is deleted while a `scan:library` job is already queued, the job still completes successfully — filesystem paths are embedded in the payload at enqueue time and no database lookup occurs. Downstream `process:file` jobs treat a missing library record as a warning rather than an error, so no entries appear in the Asynqmon Failed tab. The observable consequence is that scanned books are created without a library association (orphaned books). To recover, re-add the library and run a fresh scan, or delete the orphaned book records directly. |
| ExifTool absent | No Failed entry | When ExifTool is not installed, embedded metadata extraction fails, but the job still succeeds. `process:file` may still derive title and author from the filename/library path, while embedded fields such as ISBN, description, and cover are not extracted. ExifTool absence is typically reported once at startup at `WARN`; for affected imports, check the log stream using the [structured log queries in Observability](observability.md#book-import-troubleshooting). |
| `enrich:goodreads` or `enrich:ai` exhausting retries | Dead queue | If all 5 retries are exhausted (e.g. Goodreads is unreachable or the LLM provider is misconfigured), asynq moves the job to the **Dead** queue — a separate view in Asynqmon distinct from the Retrying/Failed view. Dead jobs do **not** resolve on their own and require explicit operator action: open the Dead tab in Asynqmon and either **retry** the job (once the root cause is resolved) or **delete** it. A job in the Dead queue does not affect the existing book record; only enrichment metadata remains missing. |
| `enrich:ai` with no LLM provider configured | Completed queue (no error) | When the LLM provider is not configured at startup, `enrich:ai` publishes an error progress event and returns `nil`. asynq does not retry it and it does not enter the Dead queue. The book record is unaffected. Fix: configure an LLM provider via [`PUT /api/config/llm`](api/config.md#put-apiconfigllm--admin--jwt-only), then restart the server. The config endpoint reports `restart_required: true`, and without a restart the worker will continue skipping `enrich:ai` jobs because the provider is wired at startup. |

> **The Dead queue vs. the Retrying/Failed view:** Asynqmon's main queue view shows jobs that are actively retrying (status `Retry`). The **Dead** tab shows jobs that have exhausted all `DefaultMaxRetry` (5) attempts. A job in the Dead tab will never retry on its own — it is waiting for a human decision. Check the Dead tab after any sustained service disruption.

> **For structured-log-based alerting on job failures**, see [Observability → Alerting Guidance](observability.md#alerting-guidance) and the [jq queries for background job activity](observability.md#book-import-troubleshooting).

## Project Layout

```
cmd/server/main.go            # Registers handlers, schedules, starts worker
internal/
  jobs/
    enrich_goodreads.go            # enrich:goodreads handler — fetches Goodreads metadata and creates a pending goodreads_metadata record
    enrich_ai.go               # enrich:ai handler — generates metadata via LLM and creates a pending ai_enrichments record
    scan_directory.go          # ScanDirectory: walks a path and enqueues process:file jobs; defines Enqueuer interface, LookupSupportedFileType, and SupportedFileTypes (canonical source of truth for supported book formats)
    scan_libraries.go          # scan:libraries handler (scans all monitored libraries)
    scan_library.go            # scan:library handler (scans a single library)
    scan_path.go               # scan:path handler (NewScanPathHandler → ScanDirectory)
    scan_watch_folder.go       # scan:watch-folder handler — reads watch-folder settings and delegates to ScanDirectory
    process_file.go            # process:file handler — deserializes payload, delegates to ProcessBookFile
    process_book_file.go       # ProcessBookFile public entry point
    book_metadata_helpers.go   # deriveTitle, extractBookMetadata, resolveAuthorAndTitle
    book_path_helpers.go       # validatePayload, resolveSourcePath, checkDuplicate, defaultBookFileLookup (bookFileLookupFunc type)
    book_record_helpers.go     # maybeReorganizeFile, createBookRecord, linkBookAssociations
  coverutil/
    decode.go                  # DecodeDataURL: decodes base64 data: URLs; enforces the 20 MB size limit
  organize/
    organize.go                # ReorganizeFile / ReorganizeFileFlat: move files into canonical library layouts
    rename_noreplace_linux.go  # renameNoReplace for Linux: uses RENAME_NOREPLACE via renameat2(2)
    rename_noreplace_other.go  # renameNoReplace for non-Linux: link+remove fallback
  pathparser/
    pathparser.go              # ParseBookPath: extracts author/title/series from directory structure; strips trailing year tokens from titles
  sidecar/
    sidecar.go                 # WriteSidecarFiles: orchestrates cover and OPF writing after each import
    cover.go                   # WriteCover: decodes CoverImageURL data URL and writes cover.<ext> to disk
    opf.go                     # WriteOPF: marshals and writes metadata.opf (OPF 2.0 Dublin Core)
    naming.go                  # sidecarTarget: resolves sidecar directory and filename stem (book_per_file uses book stem; other modes use default names)
  worker/
    worker.go                  # Worker struct: Register, Enqueue, Start, Close
```

## Adding a New Job

1. **Define the job name and payload** in a new file under `internal/jobs/`:

   ```go
   const JobExample = "example:task"

   type ExamplePayload struct {
       ID string `json:"id"`
   }
   ```

2. **Write the handler** returning a `func(ctx context.Context, payload []byte) error`:

   ```go
   func NewExampleHandler(database *db.DB) func(ctx context.Context, payload []byte) error {
       return func(ctx context.Context, payload []byte) error {
           var p ExamplePayload
           if err := json.Unmarshal(payload, &p); err != nil {
               return fmt.Errorf("unmarshal example payload: %w", err)
           }
           // ... do work ...
           return nil
       }
   }
   ```

3. **Register the handler** in the `if runWorker` block in `cmd/server/main.go`:

   ```go
   w.Register(cancelCtx, jobs.JobExample, jobs.NewExampleHandler(database))
   ```

   Handlers registered here execute only when the process runs in `worker` or `all` mode.

4. **(Optional) Schedule it** if it should run periodically:

   ```go
   if _, err := w.RegisterSchedule("@every 24h", jobs.JobExample, jobs.ExamplePayload{}); err != nil {
       return fmt.Errorf("schedule example job: %w", err)
   }
   ```

   Register the schedule in `cmd/server/main.go` after calling `w.Register`. Both classic cron expressions (`0 3 * * *`) and asynq shortcuts (`@every 24h`, `@daily`) are accepted.

5. **(Optional) Enqueue from a handler** if it should be triggered by an API call:

   ```go
   if _, err := h.Enqueuer.Enqueue(ctx, jobs.JobExample, jobs.ExamplePayload{ID: "abc"},
       jobs.WithUnique(24*time.Hour)); err != nil {
       return fmt.Errorf("enqueue example job: %w", err)
   }
   ```

   Pass `jobs.WithUnique(24*time.Hour)` for deduplication (scan jobs), or omit it for user-retriggerable jobs such as metadata fetches. Use `jobs.WithMaxRetry(n)` to override the default retry count and `jobs.WithQueue(worker.QueueCritical)` to place urgent jobs on the high-priority queue.

6. **Write tests** — see the existing `*_test.go` files in `internal/jobs/` for patterns using mock enqueuers and in-memory SQLite databases.
