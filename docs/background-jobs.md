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

Creates a `book` record and a `book_file` record in the database. The book title is derived from the filename (e.g. `Pride and Prejudice.epub` → `"Pride and Prejudice"`). Full metadata extraction (author, ISBN, publisher, etc.) is not performed during import and is planned for a future enhancement; use the standalone [`cmd/cli`](../README.md#cli-tool) tool to extract metadata from individual files in the meantime.

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

2. **Scheduled** — The asynq scheduler fires `scan:libraries` every 24 hours, which cascades into `scan:library` → `scan:path` → `process:file`. These scheduled tasks are enqueued by the scheduler itself and are not deduplicated — they run on every scheduled tick.

API-triggered jobs call `Worker.Enqueue`, which serialises the payload to JSON and pushes an asynq task onto the `"default"` queue with the configured deduplication options. Scheduled jobs are created by the asynq scheduler without going through `Worker.Enqueue`.

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
