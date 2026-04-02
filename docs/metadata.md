# Metadata Extraction

Biblioteka extracts metadata — title, author, ISBN, description, publisher, language, publication date, and embedded cover art when available — from book files using [ExifTool](https://exiftool.org/).

| Supported formats | External dependency |
|-------------------|---------------------|
| `.epub`, `.mobi`, `.azw3`, `.pdf` | [ExifTool](https://exiftool.org/) must be on `PATH` |

The extractor is implemented in [`internal/metadata/extractor.go`](../internal/metadata/extractor.go) and exposed to end users via the standalone [`cmd/cli`](../cmd/cli/main.go) utility.

> **Import pipeline status:** Automatic metadata extraction is **now active** in the `process:file` background job (since v0.0.5). When a file is imported during a library scan, the extractor runs and populates the book record with `Title`, `ISBN` (stored as ISBN-10 or ISBN-13), `Description`, `Publisher`, `Language`, `PublicationDate`, and `cover_image_url` when extracted values are non-empty. If extraction fails (for example, because ExifTool is not installed), the job falls back to deriving the book title from the filename. Author records are also created automatically from extracted author metadata and linked to the imported book.

---

## Extracted fields

| Field | Notes |
|-------|-------|
| `Title` | Falls back to filename (without extension) when ExifTool cannot find a `Title` tag |
| `Author` | Tries `Author` tag first, then `Creator` (used by EPUBs). Returns `""` when not found; no author record is created for an empty author |
| `ISBN` | Tries `ISBN` tag first, then `Identifier` tag. Normalized via `NormalizeISBN` (strips `urn:isbn:`/`isbn:` prefixes, validates as 10 or 13 digits). Returns `""` when no valid identifier is present |
| `Format` | Uppercase file extension (e.g. `"EPUB"`, `"PDF"`) |
| `Description` | From ExifTool `Description` tag; availability depends on the file format and its embedded metadata |
| `Publisher` | From ExifTool `Publisher` tag |
| `Language` | From ExifTool `Language` tag |
| `PublicationDate` | From ExifTool `PublicationDate` tag; normalized from ExifTool's `YYYY:MM:DD` format to `YYYY-MM-DD` |
| `CoverImageURL` | For EPUB files with embedded cover art, resolved from ExifTool's cover manifest tags and stored as a `data:` URL. Cover images larger than 20 MB are skipped; a warning is logged and the field is left empty. |

---

## ExifTool tag mapping

All formats are handled by [ExifTool](https://exiftool.org/) running as a stay-open subprocess managed by the custom [`internal/exif`](../internal/exif/) package. The package owns the ExifTool process lifecycle, TSV output parsing, and EPUB cover extraction; it replaces the previously used `go-exiftool` third-party library.

| ExifTool tag | `BookMetadata` field | Fallback |
|--------------|---------------------|----------|
| `Title` | `Title` | Filename stem |
| `Author`, `Creator` | `Author` | `""` (empty; no author record created) |
| `ISBN`, `Identifier` | `ISBN` | `""` (empty; normalized and validated) |
| `Description` | `Description` | `""` |
| `Publisher` | `Publisher` | `""` |
| `Language` | `Language` | `""` |
| `PublicationDate` | `PublicationDate` | `""` |
| `MetaName`, `MetaContent`, `ManifestItemId`, `ManifestItemHref`, `ManifestItemMedia-type` | `CoverImageURL` | `""` when no embedded cover is found |

When ExifTool is **not installed**, `NewExtractor()` still returns a valid `*Extractor` (with a warning logged), but calling `ExtractMetadata` on any file returns an error:

```
exiftool is not available on this system
```

### Installing ExifTool

| Platform | Command |
|----------|---------|
| Ubuntu / Debian | `apt-get install libimage-exiftool-perl` |
| macOS | `brew install exiftool` |
| Alpine | `apk add exiftool` |
| Windows | Download from [exiftool.org](https://exiftool.org/) and add to `PATH` |

The Dockerfiles included in this repository already install ExifTool.

---

## CLI tool

`cmd/cli` is a standalone utility for importing book files and scanning directories outside of the running server. It is useful for importing individual files, verifying metadata extraction, or triggering a directory scan without starting the full server.

```bash
# Build
go build -o biblioteka-cli ./cmd/cli
```

> **Note:** The CLI requires a database to be configured via the same environment variables as the server (see [deployment.md](deployment.md)).

### `db-migrate` — run database migrations

Applies any pending database migrations and exits. Useful for running migrations without starting the full HTTP server — for example, in a one-off init container or a CI schema-dump step.

```bash
./biblioteka-cli db-migrate
```

**Output on success:**

```
Database migrations completed successfully
```

The server runs the same migrations automatically on startup via `db.SetupDatabase`, so this command is only needed when you want to migrate without binding to a port (e.g. in an init container, a `make db-dump` step, or to verify the schema before running tests).

---

### `process-file` — import a single book file

Extracts metadata from one file, stores a book and book_file record in the database, and creates an author record when one is found. Records are written directly to the database rather than going through the background job queue.

```bash
./biblioteka-cli process-file /path/to/book.epub
./biblioteka-cli process-file /path/to/book.pdf
```

**Legacy shorthand** (backwards-compatible): passing a file path directly without a subcommand invokes `process-file`:

```bash
./biblioteka-cli /path/to/book.epub
```

**Example output:**

```
Successfully processed file: /path/to/book.epub
```

> **Note:** When ExifTool is not installed, imports still succeed but metadata is derived from the filename only. Install ExifTool to enable full metadata extraction (title, author, ISBN, etc.).

### `scan-directory` — enqueue a directory for processing

Recursively walks a directory and enqueues a `process:file` background job for every supported file (`.epub`, `.mobi`, `.pdf`, `.azw3`). Jobs are pushed to the Redis queue defined by `REDIS_URL` and processed by a running worker.

```bash
./biblioteka-cli scan-directory /path/to/library
./biblioteka-cli scan-directory /path/to/library <library-id>
```

| Argument | Required | Description |
|---|---|---|
| `<directory>` | Yes | Path to the directory to scan (resolved to an absolute path) |
| `<library-id>` | No | UUID of an existing library record to associate the imported books with |

When `<library-id>` is supplied the directory is also used as the `library_root`, enabling [path-based metadata](background-jobs.md#path-based-metadata) and [file reorganization](background-jobs.md#file-reorganization) in the worker.

**Requirements:** a Redis instance reachable at `REDIS_URL` (default `redis://localhost:6379`) and at least one worker process running to consume the enqueued jobs.

---

## Sidecar files

After every book import the `process:file` job writes companion files into the same directory as the book file:

| File | Condition | Contents |
|------|-----------|----------|
| `metadata.opf` | **Always** | OPF 2.0 Dublin Core metadata: title, author, identifier, language, publication date, publisher, description, and a cover manifest entry when a cover is present. |
| `cover.<ext>` | Only when a cover image is available (EPUB imports with embedded art) | Cover image decoded from the stored `data:` URL (JPEG, PNG, WebP, or AVIF). |

These sidecar files are written by `internal/sidecar` after the book record is committed. See [background-jobs.md — Sidecar files](background-jobs.md#sidecar-files) for the full specification.

---

## Goodreads lookup

The CLI includes commands for querying the Goodreads catalog by text query, ISBN, ASIN, or Goodreads ID. These are useful for enriching book records with Goodreads IDs and supplementary metadata before or after an import.

> **No configuration required.** The Goodreads client uses bundled credentials and requires no API key setup from the user.

### Commands

#### `goodreads-search` — search by text query

Searches the Goodreads catalog for books matching a free-text query. Returns up to the first page of results.

```bash
./biblioteka-cli goodreads-search "The Name of the Wind"
./biblioteka-cli goodreads-search "Patrick Rothfuss kingkiller"
```

#### `goodreads-search-isbn` — search by ISBN

Looks up books by ISBN-10 or ISBN-13. Hyphens and spaces are stripped automatically before lookup. The check digit is validated; an ISBN with an invalid check digit causes the command to exit with an error. Returns up to 5 results.

```bash
./biblioteka-cli goodreads-search-isbn 9780756404741
./biblioteka-cli goodreads-search-isbn 978-0-7564-0474-1
./biblioteka-cli goodreads-search-isbn 0756404746
```

**Validation errors:**

| Condition | Error message |
|-----------|---------------|
| Empty input | `ISBN cannot be empty` |
| Non-digit character (except trailing `X`/`x` in ISBN-10) | `invalid ISBN: <value> (unexpected character '<char>')` |
| Wrong length (not 10 or 13 digits) | `invalid ISBN: <value>` |
| Invalid ISBN-10 check digit | `invalid ISBN-10 check digit: <value>` |
| Invalid ISBN-13 check digit | `invalid ISBN-13 check digit: <value>` |

> **Note:** A trailing `X` or `x` is accepted as a valid ISBN-10 check digit (the standard representation of check value 10). For example, `080442957X` is a valid ISBN-10.

#### `goodreads-get-by-asin` — fetch by Amazon ASIN

Retrieves a single book record using its Amazon ASIN.

```bash
./biblioteka-cli goodreads-get-by-asin B0034P1031
```

#### `goodreads-get-by-id` — fetch by Goodreads ID

Retrieves a single book record using its Goodreads KCA book ID (e.g. `kca://book/amzn1.gr.book.v1.xyz`).

```bash
./biblioteka-cli goodreads-get-by-id "kca://book/amzn1.gr.book.v1.xyz"
```

#### `goodreads-get-by-legacy-id` — fetch by legacy integer ID

Retrieves a single book record using the legacy numeric Goodreads book ID shown in older Goodreads URLs.

```bash
./biblioteka-cli goodreads-get-by-legacy-id 186074
```

### Output format

**Search commands** (`goodreads-search`, `goodreads-search-isbn`) print one line per result:

```
N. <title> by <author> (Goodreads ID: <id>)
```

Example:

```
Goodreads search results for query: The Name of the Wind
1. The Name of the Wind by Patrick Rothfuss (Goodreads ID: kca://book/amzn1.gr.book.v1.xyz)
```

**Single-result commands** (`goodreads-get-by-asin`, `goodreads-get-by-id`, `goodreads-get-by-legacy-id`) print the following fields:

| Printed field | Commands |
|---------------|----------|
| `Title` | All three |
| `Author` | All three |
| `ASIN` | `goodreads-get-by-id`, `goodreads-get-by-legacy-id` |
| `Goodreads ID` | All three |

Example (`goodreads-get-by-legacy-id`):

```
Goodreads search result for legacy ID: 186074
Title: The Name of the Wind
Author: Patrick Rothfuss
ASIN: B0034P1031
Goodreads ID: kca://book/amzn1.gr.book.v1.xyz
```

> **Note:** The underlying `BookResult` struct contains additional fields (`work_id`, `work_legacy_id`, `book_legacy_id`, `book_image_url`, `book_isbn`, `book_isbn13`, `book_language`, `book_number_of_pages`, `author_id`, `author_legacy_id`, `author_profile_image_url`) that are not currently included in CLI output.

---

## What's next

The `process:file` background job ([`internal/jobs/process_file.go`](../internal/jobs/process_file.go)) extracts and stores `Title`, `ISBN`, `Description`, `Publisher`, `Language`, `PublicationDate`, embedded EPUB cover art, and links extracted `Author` names to book records. Planned future improvements include:

1. **More cover formats** — extend embedded cover extraction beyond EPUB (currently only EPUB files get an extracted `CoverImageURL`; the sidecar cover write already supports JPEG, PNG, WebP, and AVIF once a URL is present).

Use `cmd/cli` to import a single file and verify what Biblioteka extracts before a full library scan.

---

## Contributing

To add support for a new extracted field:

1. Add the field to `ExifToolOutput` in [`internal/exif/tsv.go`](../internal/exif/tsv.go).
2. Add the corresponding tag-name mapping in the `ParseTSV` function in the same file.
3. Add or extend tests in [`internal/exif/tsv_test.go`](../internal/exif/tsv_test.go) and [`internal/metadata/extractor_test.go`](../internal/metadata/extractor_test.go).
4. Update the [Extracted fields](#extracted-fields) table in this document.

To add support for a new file format:

1. Verify ExifTool supports the format and returns tags Biblioteka can map (title, author, ISBN, etc.).
2. Extend `ParseTSV` in [`internal/exif/tsv.go`](../internal/exif/tsv.go) if format-specific tag names differ.
3. If the format embeds cover art differently from EPUB, add extraction logic in [`internal/exif/cover.go`](../internal/exif/cover.go).
4. Add test fixtures and test cases in [`internal/metadata/extractor_test.go`](../internal/metadata/extractor_test.go).

### `internal/exif` package overview

| File | Responsibility |
|------|----------------|
| [`exif.go`](../internal/exif/exif.go) | Manages the ExifTool stay-open subprocess: start, communicate via stdin/stdout, graceful close with timeout, and dead-instance detection |
| [`tsv.go`](../internal/exif/tsv.go) | Parses ExifTool's tab-separated output into `ExifToolOutput`; defines `NormalizeISBN` |
| [`cover.go`](../internal/exif/cover.go) | Extracts EPUB cover images directly from the ZIP archive using the manifest reference returned by ExifTool |
| [`exif_write.go`](../internal/exif/exif_write.go) | Writes metadata back to files via the stay-open ExifTool process; used by sidecar/OPF workflows |
