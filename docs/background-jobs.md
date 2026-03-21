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

- `book_per_folder`: `<library_root>/<Author>/<Title>/<filename>`
- `book_per_file`: `<library_root>/<Author>/<filename>`

Directory names are sanitized (path separators, control characters, and leading dots removed). The move uses `os.Rename` when source and destination are on the same filesystem, falling back to copy-then-delete for cross-filesystem moves. Intermediate empty source directories are removed after a successful move.

**Failure handling:**
- If reorganization fails for most reasons (e.g. permissions, cross-device copy errors), the handler logs a warning, cleans up any partial copy created at the destination, and continues processing the file at its original path.
- If the source file disappears *during* reorganization (i.e. `os.IsNotExist` is returned by the move/copy step), the job returns an error so that asynq can retry it rather than committing a `book_files` row pointing at a non-existent path.
- If the source file is missing when the job starts, the handler first checks whether the original path is already indexed in the database. If it is, the job skips without error (and, if a `library_id` is present, attempts to link the book to that library). If the original path is not in the database, the handler then tries to locate the file at its expected reorganized path: `<library_root>/<Author>/<Title>/<filename>` for `book_per_folder`, or `<library_root>/<Author>/<filename>` for `book_per_file`. If found there and already indexed, the job skips without error. If found there but not yet indexed, the handler resumes processing from that new location. If the file cannot be found at any expected location, the job logs a warning and returns without error. A database error (other than "not found") at any of these lookup steps is propagated as a hard error so that asynq retries the job rather than silently dropping it.
- After a successful move, the handler checks whether the new path is already indexed before creating new database records, preventing duplicates from concurrent workers.

See [Administration — File organization](administration.md#file-organization) for details on configuring this feature per-library.

#### Sidecar files

After a book record is created (and the file has been optionally reorganized), the handler calls `internal/sidecar.WriteSidecarFiles`, which writes two companion files alongside the book file:

| File | Condition | Description |
|------|-----------|-------------|
| `cover.<ext>` | Only when `CoverImageURL` is non-empty | Cover image decoded from the stored `data:` URL |
| `metadata.opf` | Always | OPF 2.0 file with Dublin Core metadata |

**Cover image (`cover.<ext>`)**

The cover is decoded from the base64 `data:` URL stored in `books.cover_image_url` (populated during extraction for EPUB files). The file extension is determined by the decoded MIME type:

| MIME type | Output filename |
|-----------|-----------------|
| `image/jpeg` | `cover.jpg` |
| `image/png` | `cover.png` |
| `image/webp` | `cover.webp` |
| `image/avif` | `cover.avif` |

Cover files are written **atomically**: the new image is first written to a temporary file (`cover.<ext>.tmp`), then renamed into its final position. Only after the rename succeeds are stale cover files of other formats removed (best-effort cleanup to avoid orphaned files when cover formats change). This ensures the directory always contains either the previous cover or the new one — never nothing — even if the process is interrupted mid-write.

Cover data URLs are capped at **20 MB** of decoded bytes; inputs exceeding this limit are rejected with a warning and no cover file is written.

**OPF metadata file (`metadata.opf`)**

`metadata.opf` is an [OPF 2.0](https://idpf.org/epub/20/spec/OPF_2.0.1_draft.htm) file suitable for use by e-reader applications (Kobo, KOReader, and others). It contains Dublin Core metadata:

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
    scan_directory.go          # ScanDirectory: walks a path and enqueues process:file jobs; defines Enqueuer interface and supportedExtensions
    scan_path.go               # scan:path handler (NewScanPathHandler → ScanDirectory)
    scan_libraries.go          # scan:libraries handler (scans all monitored libraries)
    scan_library.go            # scan:library handler (scans a single library)
    process_file.go            # process:file handler — deserializes payload, delegates to ProcessBookFile
    process_book_file.go       # ProcessBookFile public entry point
    book_metadata_helpers.go   # deriveTitle, extractBookMetadata, resolveAuthorAndTitle
    book_path_helpers.go       # validatePayload, resolveSourcePath, checkDuplicate, defaultBookFileLookup (bookFileLookupFunc type)
    book_record_helpers.go     # maybeReorganizeFile, createBookRecord, linkBookAssociations
  coverutil/
    decode.go                  # DecodeDataURL: decodes base64 data: URLs; enforces the 20 MB size limit
  organize/
    organize.go                # ReorganizeFile / ReorganizeFileFlat: move files into canonical library layouts
  pathparser/
    pathparser.go              # ParseBookPath: extracts author/title/series from directory structure; strips trailing year tokens from titles
  sidecar/
    sidecar.go                 # WriteSidecarFiles: orchestrates cover and OPF writing after each import
    cover.go                   # WriteCover: decodes CoverImageURL data URL and writes cover.<ext> to disk
    opf.go                     # WriteOPF: marshals and writes metadata.opf (OPF 2.0 Dublin Core)
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
