# Metadata Extraction

Biblioteka extracts metadata — title, author, ISBN, ASIN, description, publisher, language, publication date, subjects, and embedded cover art when available — from book files using [ExifTool](https://exiftool.org/).

| Supported formats | External dependency |
|-------------------|---------------------|
| `.epub`, `.mobi`, `.azw3`, `.pdf` | [ExifTool](https://exiftool.org/) must be on `PATH` |

The extractor is implemented in [`internal/metadata/extractor.go`](../internal/metadata/extractor.go) and exposed to end users via the standalone [`cmd/cli`](../cmd/cli/main.go) utility.

---

## ExifTool tag mapping

All formats are handled by [ExifTool](https://exiftool.org/) running as a stay-open subprocess managed by the custom [`internal/exif`](../internal/exif/) package. The package owns the ExifTool process lifecycle, TSV output parsing, and cover extraction for EPUB, MOBI, and AZW3 files; it replaces the previously used `go-exiftool` third-party library.

| ExifTool tag | `BookMetadata` field | Fallback |
|--------------|---------------------|----------|
| `Title` | `Title` | Filename stem |
| `Author`, `Creator` | `Author` | `""` (empty; no author record created) |
| `ISBN`, `Identifier` | `ISBN` | `""` (empty; `NormalizeISBN` strips `urn:isbn:`/`isbn:` prefixes and normalizes values to 10- or 13-character ISBN-like strings; it does not verify ISBN-10/13 check digits) |
| _(file extension)_ | `Format` | Lowercased extension (e.g. `"epub"`, `"pdf"`); persisted as `book_files.file_type`, not on `books` |
| `Description` | `Description` | `""` |
| `Publisher` | `Publisher` | `""` |
| `Language` | `Language` | `""` |
| `PublicationDate` | `PublicationDate` | `""` (ExifTool's `YYYY:MM:DD` format normalized to `YYYY-MM-DD`) |
| `MetaName`, `MetaContent`, `ManifestItemId`, `ManifestItemHref`, `ManifestItemMedia-type` | `CoverImageURL` (EPUB) | `""` when no embedded cover is found |
| _(binary record in MOBI/AZW3 file)_ | `CoverImageURL` (MOBI, AZW3) | `""` when no embedded cover is found |
| `ASIN` | `ASIN` (MOBI, AZW3, EPUB `dc:identifier` with scheme `AMAZON` or `MOBI-ASIN`, or value prefixed `urn:amazon`) | `""` — extracted but not persisted during import |
| `Subject` | `Subjects []string` (EPUB) | `[]` — values separated by `", "` (comma + space) are split, trimmed, and deduplicated; not persisted |

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

Applies pending migrations and exits (useful in init containers or CI schema-dump steps without binding to a port). Prints `Database migrations completed successfully` on success. The server runs the same migrations automatically on startup.

```bash
./biblioteka-cli db-migrate
```

---

### `process-file` — import a single book file

Extracts metadata from one file, stores a book and book_file record in the database, and creates an author record when one is found. Records are written directly to the database rather than going through the background job queue.

```bash
./biblioteka-cli process-file /path/to/book.epub
# Legacy shorthand (backwards-compatible):
./biblioteka-cli /path/to/book.epub
```

Prints `Successfully processed file: /path/to/book.epub` on success.

> **Note:** When ExifTool is not installed, imports still succeed but metadata is derived from the filename only.

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

### `calibre-import` — import a Calibre library

Reads a Calibre library's `metadata.db` and imports books, authors, series, and file records directly into the Biblioteka database. No Redis or worker is required — all writes happen inline.

```bash
./biblioteka-cli calibre-import /path/to/calibre/library
./biblioteka-cli calibre-import /path/to/calibre/library <library-id>
```

| Argument | Required | Description |
|---|---|---|
| `<calibre-library-path>` | Yes | Root directory of the Calibre library (must contain `metadata.db`) |
| `<library-id>` | No | UUID of an existing Biblioteka library to associate every imported book with |

The import is **idempotent**: books already indexed by file path are skipped, not duplicated. Per-book errors are counted without aborting the rest of the import.

**Example output:**

```
Calibre import complete:
  Total books:    312
  Imported:       308
  Skipped:        3
  Errors:         1
```

> **Note:** Calibre tags are not imported. Re-tag books manually in Biblioteka after import.

See the full [Migrate from Calibre](calibre.md) guide for identifier mapping, deduplication behavior, troubleshooting, cover backfill options, and a step-by-step workflow.

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

## AI enrichment

Biblioteka can enrich book metadata using a locally-hosted large language model (LLM). When configured, AI enrichment is triggered through the metadata API and enqueues a background job that sends the book's title, authors, and existing description to the LLM. The model returns suggested tags (which can include genres, themes, or mood-related labels), a reading level, and a generated catalog description. Results are stored as a **pending enrichment record** that you review before any changes are committed to the book.

> **Requires Redis and an LLM provider.** AI enrichment runs as a background job; a Redis worker must be running and an LLM provider must be configured by an administrator. See [Administration — LLM Configuration](administration.md#llm-configuration-runtime) for setup instructions and [Background Jobs — enrich:ai](background-jobs.md#enrichai) for job internals.

### Enrichment workflow

1. **Trigger** — send `POST /api/books/{id}/metadata/ai-fetch`. The server enqueues an `enrich:ai` job and returns immediately. The request is idempotent: if a pending enrichment already exists for the book, or the job is already running, it returns `202` without enqueueing a duplicate.
2. **Monitor progress** — subscribe to the SSE stream at `GET /api/books/{id}/metadata/events`. The same event stream used for Goodreads enrichment publishes `progress`, `complete`, and `error` events for AI enrichment jobs.
3. **Review** — fetch the pending result with `GET /api/books/{id}/metadata/ai`. The response includes suggested tags, a reading level, and an AI-generated description (all pending, not yet applied).
4. **Apply or reject** — choose one of:
   - `POST /api/books/{id}/metadata/ai-apply` — merges the AI-suggested tags into the book's existing tag set (union, no duplicates) and sets the description if the book has no description yet.
   - `POST /api/books/{id}/metadata/ai-reject` — marks the pending record as rejected without modifying the book.

### `AIEnrichment` object fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Opaque enrichment record ID |
| `book_id` | string\|null | ID of the associated book; `null` if the book has been deleted (the foreign-key constraint sets this to `null` on book deletion) |
| `status` | `"pending"` \| `"applied"` \| `"rejected"` | Current review state |
| `provider` | string | LLM provider that generated the enrichment (e.g. `"ollama"`) |
| `model` | string | Model name used (e.g. `"llama3"`) |
| `suggested_tags` | string[] | AI-suggested tag names |
| `reading_level` | string\|null | Suggested reading level. Expected values: `"children"`, `"young_adult"`, `"adult"`, `"academic"` (stored as free-form text; not server-validated) |
| `generated_description` | string\|null | AI-generated catalog description |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last-updated timestamp |

### Apply semantics

- **Tags** — the suggested tags are merged with the book's existing tags using a union (existing tags are never removed).
- **Description** — the generated description is applied only when the book's current description is blank. If the book already has a description it is left unchanged.
- **Blank tags** — suggested tag strings that are blank or whitespace-only are silently skipped during apply.

> **User isolation.** Only enrichment records owned by the authenticated user are returned or modifiable. Each user's pending enrichments are fully isolated from other users'.

For the full request/response schemas and error codes, see the [API Reference — Books: AI enrichment endpoints](api/books.md#get-apibooksidmetadataai).

---

## Contributing

To add support for a new extracted field:

1. Add the field to `ExifToolOutput` in [`internal/exif/types.go`](../internal/exif/types.go).
2. Add the corresponding tag-name mapping as a `case` in the `parseScalar` function in [`internal/exif/tsv.go`](../internal/exif/tsv.go).
3. Add or extend tests in [`internal/exif/tsv_test.go`](../internal/exif/tsv_test.go) and [`internal/metadata/extractor_test.go`](../internal/metadata/extractor_test.go).
4. Update the [ExifTool tag mapping](#exiftool-tag-mapping) table in this document.

To add support for a new file format:

1. Verify ExifTool supports the format and returns tags Biblioteka can map (title, author, ISBN, etc.).
2. Extend `ParseTSV` in [`internal/exif/tsv.go`](../internal/exif/tsv.go) if format-specific tag names differ.
3. Add a format case in `finishBook` in [`internal/exif/finish.go`](../internal/exif/finish.go). If the format embeds cover art in the binary file (like MOBI/AZW3), implement extraction in a new helper or in [`internal/exif/mobi_cover.go`](../internal/exif/mobi_cover.go). If cover art is in a ZIP archive (like EPUB), add logic to [`internal/exif/cover.go`](../internal/exif/cover.go).
4. Add test fixtures and test cases in [`internal/exif/tsv_test.go`](../internal/exif/tsv_test.go) and [`internal/metadata/extractor_test.go`](../internal/metadata/extractor_test.go).

### `internal/exif` package overview

| File | Responsibility |
|------|----------------|
| [`exif.go`](../internal/exif/exif.go) | Manages the ExifTool stay-open subprocess: start, communicate via stdin/stdout, graceful close with timeout, and dead-instance detection |
| [`types.go`](../internal/exif/types.go) | Defines `ExifToolOutput`, `Identifier`, `MetaTag`, `ManifestItem`, and related types |
| [`tsv.go`](../internal/exif/tsv.go) | Parses ExifTool's tab-separated output into `ExifToolOutput`; handles scalar, identifier, meta, and manifest record assembly |
| [`finish.go`](../internal/exif/finish.go) | Format-specific post-processing: `finishBook` dispatches to `finishEPUB` (cover manifest resolution) and `finishMOBI` (MOBI/AZW3 cover extraction) |
| [`cover.go`](../internal/exif/cover.go) | Extracts EPUB cover images directly from the ZIP archive using the manifest reference returned by ExifTool |
| [`mobi_cover.go`](../internal/exif/mobi_cover.go) | Extracts cover images from MOBI and AZW3 files using the `sblinch/mobi` library; returns a `data:` URL |
| [`isbn.go`](../internal/exif/isbn.go) | `NormalizeISBN` (strip prefixes, validate length and check digit) and `isASIN` helpers |
| [`exif_write.go`](../internal/exif/exif_write.go) | Writes metadata back to files via the stay-open ExifTool process; used by sidecar/OPF workflows |

### EPUB cover discovery

EPUB cover art is located via `finishEPUB` in [`internal/exif/finish.go`](../internal/exif/finish.go), which tries five strategies in priority order and stops at the first successful match:

| Priority | Strategy | Description | EPUB version |
|----------|----------|-------------|--------------|
| 1 | Manifest ID `"cover"` | During TSV parsing, a manifest item whose `id` attribute equals `"cover"` (case-insensitive) and whose media type is an image is recorded directly | EPUB 2 and 3 |
| 2 | `<meta name="cover">` | A `<meta>` tag with `name="cover"` points to a manifest item ID; that item is looked up by ID | EPUB 2 |
| 3 | `properties="cover-image"` | A manifest item carrying the `properties="cover-image"` attribute is used | EPUB 3 |
| 4 | ID/href heuristic | A manifest item whose `id` or `href` contains the substring `"cover"` (case-insensitive) and is an image | EPUB 2 and 3 |
| 5 | Single-image fallback | When the manifest contains exactly one image item and no other strategy matched, that image is used as the cover | EPUB 2 and 3 |

> **Note on EPUB 2 vs. EPUB 3:** EPUB 2 files use the `<meta name="cover">` element (Strategy 2) to identify the cover; EPUB 3 files use the `properties="cover-image"` manifest attribute (Strategy 3). Strategy 2 takes precedence over Strategy 3, so if both are present (which is valid in a hybrid or poorly-authored file), the EPUB 2 pointer wins. In `internal/testutils/epub.go`, fixture generation is controlled by `EPUBOptions.Version` (defaults to `"3.0"`), `EPUBOptions.EPUB3Cover`, and `EPUBOptions.Subjects`: use `Version: "2.0"` for EPUB 2 cover fixtures; use `Version: "3.0"` together with `EPUB3Cover: true` to emit the EPUB 3 `properties="cover-image"` marker for Strategy 3 tests; pass `Subjects: []string{"Fiction", "Mystery"}` to embed `<dc:subject>` elements for subject-extraction tests.
