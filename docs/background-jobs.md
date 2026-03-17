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
| **Source** | `internal/jobs/scan_libraries.go` — `NewScanLibraryHandler` |
| **Trigger** | Enqueued by `scan:libraries`, or immediately when a library is created via the API |
| **Payload** | `{ "library_id": "<uuid>", "paths": ["/books", "/more-books"] }` |

Enqueues a `scan:path` job for every path in the library.

### `scan:path`

| | |
|---|---|
| **Source** | `internal/jobs/scan_path.go` — `NewScanPathHandler` |
| **Trigger** | Enqueued by `scan:library` |
| **Payload** | `{ "path": "/books" }` |

Recursively walks the directory and enqueues a `process:file` job for every file with a supported extension (`.epub`, `.mobi`, `.pdf`, `.azw3`). Inaccessible files are logged as warnings and skipped.

### `process:file`

| | |
|---|---|
| **Source** | `internal/jobs/process_file.go` — `NewProcessFileHandler` |
| **Trigger** | Enqueued by `scan:path` |
| **Payload** | `{ "path": "/books/novel.epub", "file_name": "novel.epub", "file_type": "epub", "file_size": 524288 }` |

Creates a `book` record and a `book_file` record in the database. The `process:file` handler runs `internal/metadata.Extractor.ExtractMetadata` on every imported file and uses the result to populate the book record:

| Extracted field | Stored as | Notes |
|----------------|-----------|-------|
| `Title` | `books.title` | Falls back to filename without extension |
| `ISBN` (10 or 13 digits) | `books.isbn_10` / `books.isbn_13` | Not stored when absent |
| `Description` (when non-empty) | `books.description` | EPUB only |
| `Publisher` (when non-empty) | `books.publisher` | EPUB only |
| `Language` (when non-empty) | `books.language` | EPUB only |
| `PublicationDate` (when non-empty) | `books.publication_date` | EPUB only |
| `Author` (when non-empty) | `authors` + `book_authors` join | The extracted name is whitespace-normalized (trimmed, internal runs collapsed). An existing author is looked up **case-insensitively** (`"J.R.R. Tolkien"` and `"j.r.r. tolkien"` match the same record). If no match is found, a new author record is created. If a concurrent worker creates the same author first, the handler retries the lookup rather than failing. If association fails after the book record is already committed, the failure is logged as a warning and does **not** fail the job (preventing duplicate book records on retry). |

`Format` is extracted but stored on the `book_files` record via the `file_type` payload field, not from the extractor output directly. If ExifTool is absent or extraction fails for any other reason, the job logs a warning and falls back to the filename-derived title. See [docs/metadata.md](metadata.md) for extraction details. Use the standalone [`cmd/cli`](../README.md#cli-tool) tool to inspect metadata from individual files.

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

### Deduplication limitations

The 24-hour deduplication window is the **only** mechanism that prevents a file from being processed more than once. The `book_files` table has no `UNIQUE` constraint on `file_path`, and the `process:file` handler always creates a new book and book_file record without checking whether the file already exists in the database.

This has two practical consequences:

- **Redis data loss** — If the Redis store is cleared or the server is restarted against a fresh Redis instance, the deduplication state is lost. A subsequent scan will re-process all files and create duplicate book and book_file entries.
- **Files reachable from multiple paths** — If the same physical file is reachable under two different library paths, or under paths belonging to two separate libraries, each path produces a distinct job payload and both are processed independently, resulting in duplicate entries.

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
    scan_libraries.go          # scan:library & scan:libraries handlers
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
