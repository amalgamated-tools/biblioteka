# Background Jobs

Biblioteka uses [asynq](https://github.com/hibiken/asynq), a Redis-backed task queue for Go, to run background jobs such as scanning library paths and importing book files. A built-in scheduler triggers periodic scans, and a web dashboard lets admins inspect queues and retry failures.

## Architecture Overview

```
┌───────────────────────────┐        ┌───────────────┐
│   HTTP Server             │        │    Redis       │
│                           │        │                │
│  POST /api/libraries ─────┼──────▶ │  "default"     │
│  Scheduled (every 24 h) ──┼──────▶ │   queue        │
└───────────────────────────┘        └───────┬───────┘
                                             │
                                     ┌───────▼───────┐
                                     │  asynq Worker  │
                                     │  (4 goroutines)│
                                     │                │
                                     │  scan:libraries│
                                     │  scan:library  │
                                     │  scan:path     │
                                     │  process:file  │
                                     └────────────────┘
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

Recursively walks the directory and enqueues a `process:file` job for every file with a supported extension (`.epub`, `.mobi`, `.pdf`, `.azw3`). Inaccessible files are logged as warnings and skipped.

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
| `Description` (when non-empty) | `books.description` | EPUB only |
| `Publisher` (when non-empty) | `books.publisher` | EPUB only |
| `Language` (when non-empty) | `books.language` | EPUB only |
| `PublicationDate` (when non-empty) | `books.publication_date` | EPUB only |
| `Author` (when non-empty) | `authors` + `book_authors` join | The extracted name is whitespace-normalized (trimmed, internal runs collapsed). An existing author is looked up **case-insensitively** (`"J.R.R. Tolkien"` and `"j.r.r. tolkien"` match the same record). If no match is found, a new author record is created. If a concurrent worker creates the same author first, the handler retries the lookup rather than failing. If association fails after the book record is already committed, the failure is logged as a warning and does **not** fail the job (preventing duplicate book records on retry). Falls back to path-derived author when the embedded value is absent or `"Unknown"`. |

`Format` is extracted but stored on the `book_files` record via the `file_type` payload field, not from the extractor output directly. If ExifTool is absent or extraction fails for any other reason, the job logs a warning and falls back to the filename-derived title. See [docs/metadata.md](metadata.md) for extraction details. Use the standalone [`cmd/cli`](../README.md#cli-tool) tool to inspect metadata from individual files.

#### Path-based metadata

When `library_root` is set in the payload, `internal/pathparser.ParseBookPath` extracts author, title, series name, and series position from the file's path relative to the library root. This runs for every file regardless of whether ExifTool is present.

> **Publication year note:** The parser also detects a trailing `(YYYY)` year in filenames and uses it to produce a clean title (e.g. `Frankenstein (2009)` → `Frankenstein`). The year value itself is **not** stored in `books.publication_date`; that field is only populated from embedded EPUB metadata.

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
- **Trailing ` - Author Name`** in filenames is stripped when the suffix looks like a personal name (two to four capitalised words, no digits). Single-word suffixes are kept to avoid corrupting subtitles.
- A directory is treated as a **series** only when the filename has a leading series-position prefix **and** the directory name is not effectively the same as the parsed title (to avoid phantom series from Calibre-style `Author/Title/file.ext` layouts).
- Paths outside `library_root` fall back to filename-only parsing.

**Metadata precedence** (highest to lowest):

1. Embedded file metadata (ExifTool) — used when present and non-empty
2. Path-derived metadata (`pathparser`) — fills gaps left by step 1 (author falls back when value is absent or `"Unknown"`)
3. Filename without extension — last resort for title

#### File reorganization

When both `library_root` is set and the `organize_files` database setting equals `"true"`, the handler moves the file into a canonical `Author/Title/` directory structure under the library root after resolving the author name and title:

```
<library_root>/<Author>/<Title>/<filename>
```

Directory names are sanitized (path separators, control characters, and leading dots removed). The move uses `os.Rename` when source and destination are on the same filesystem, falling back to copy-then-delete for cross-filesystem moves. Intermediate empty source directories are removed after a successful move.

**Failure handling:**
- If reorganization fails for any reason, the handler logs a warning and continues processing the file at its original path.
- If the source file disappears during reorganization (indicating the file was already moved by a prior attempt), the job returns an error so asynq can retry.
- After a successful move, the handler checks whether the new path is already indexed before creating new database records, preventing duplicates from concurrent workers.

See [Administration — File organization](administration.md#file-organization) for how to enable this feature.

### Job Chain

A full scan flows through the jobs in a fan-out pattern:

```
scan:libraries
 └─▶ scan:library  (one per monitored library)
      └─▶ scan:path  (one per library path)
           └─▶ process:file  (one per supported file found)
```

## Scheduling

Periodic jobs are registered with the asynq scheduler at startup in `cmd/server/main.go`, **but only when the process is running in `worker` or `all` mode**:

```go
w.RegisterSchedule("@every 24h", jobs.JobScanLibraries, struct{}{})
```

The cron specification follows the format accepted by asynq — both classic cron expressions (`0 3 * * *`) and convenience shortcuts (`@every 24h`, `@daily`) are supported.

## Worker Configuration

Configuration lives in `internal/worker/worker.go` as package-level constants:

| Constant | Value | Meaning |
|----------|-------|---------|
| `QueueName` | `"default"` | All jobs are placed on this queue |
| `DefaultConcurrency` | `4` | Maximum number of jobs executing in parallel |
| `DefaultMaxRetry` | `5` | How many times a failed job is retried |

Tasks enqueued via `Worker.Enqueue` use a **24-hour deduplication window** (via `asynq.Unique(24*time.Hour)`). If the same job with the same payload is enqueued again through `Worker.Enqueue` within 24 hours, asynq returns an enqueue error (typically `asynq.ErrDuplicateTask`), and the duplicate task is not processed. Callers can choose whether to log or ignore this error — it is not silently skipped.

## How Jobs Are Enqueued

Jobs enter the queue in two ways:

1. **API-triggered** — When a user creates a library via `POST /api/libraries` and the library has paths, the handler immediately enqueues a `scan:library` job via `Worker.Enqueue` (see `internal/handlers/libraries.go`), and therefore benefits from the 24-hour deduplication window described above.

2. **Scheduled** — The asynq scheduler fires `scan:libraries` every 24 hours. The `scan:libraries` trigger itself is issued directly by the asynq scheduler (not through `Worker.Enqueue`) and carries no deduplication. However, when the `scan:libraries` handler runs, it calls `Worker.Enqueue` to create `scan:library` jobs, which cascade into `scan:path` and `process:file` jobs — all of which go through `Worker.Enqueue` and therefore benefit from the 24-hour deduplication window.

API-triggered jobs call `Worker.Enqueue`, which serialises the payload to JSON and pushes an asynq task onto the `"default"` queue with the configured deduplication options. The root `scan:libraries` scheduled trigger is created directly by the asynq scheduler and does not go through `Worker.Enqueue`.

### Deduplication

The `book_files` table has a `UNIQUE` constraint on `file_path` (`idx_book_files_file_path`). If a `process:file` job tries to insert a `book_file` row for a path that is already indexed, the database rejects the insert and the handler skips creating a duplicate record. The handler also proactively checks whether the target path is already indexed before creating database records — both at the start of the job (when the original path already exists in the database) and after a file reorganization (in case a concurrent worker processed the same file first).

Additionally, the 24-hour asynq deduplication window (via `asynq.Unique(24*time.Hour)`) prevents the same job payload from being enqueued more than once within that window.

**Remaining edge cases:**

- **Files reachable from multiple paths** — If the same physical file is reachable under two different library paths (e.g. via symlinks or overlapping mounts), each path produces a distinct job payload with a different `file_path`. Both jobs can succeed and create separate `book_file` rows pointing at different paths on disk.
- **Redis data loss** — If the Redis store is cleared or the server is restarted against a fresh Redis instance, the 24-hour deduplication window is reset. A subsequent scan may re-enqueue jobs for files already indexed; however, the database-level `UNIQUE(file_path)` constraint prevents new duplicate `book_file` rows from being created for files that have not moved.

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

## Project Layout

```
cmd/server/main.go            # Registers handlers, schedules, starts worker
internal/
  jobs/
    process_file.go            # process:file handler
    scan_path.go               # scan:path handler + Enqueuer interface
    scan_libraries.go          # scan:libraries handler (scans all monitored libraries)
    scan_library.go            # scan:library handler (scans a single library)
    process_book_file.go       # ProcessBookFile: metadata extraction, path parsing, organization logic
  organize/
    organize.go                # ReorganizeFile: moves files into Author/Title/ layout
  pathparser/
    pathparser.go              # ParseBookPath: extracts author/title/series from directory structure; strips trailing year tokens from titles
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
   if _, err := h.Enqueuer.Enqueue(ctx, jobs.JobExample, jobs.ExamplePayload{ID: "abc"}); err != nil {
       return fmt.Errorf("enqueue example job: %w", err)
   }
   ```

6. **Write tests** — see the existing `*_test.go` files in `internal/jobs/` for patterns using mock enqueuers and in-memory SQLite databases.
