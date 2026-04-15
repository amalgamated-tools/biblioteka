# Migrating from Calibre

Biblioteka includes a `calibre-import` CLI command that reads a Calibre library from disk and copies books, authors, series, and file records directly into your Biblioteka database. The import is **idempotent for stable file paths** — re-running it against the same library never creates duplicates as long as the Calibre book paths have not changed.

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| Calibre library on disk | Must contain a `metadata.db` file in the library root |
| Biblioteka database configured | Uses the same environment variables as the server (see [Deployment](deployment.md)) |
| No Redis / worker required | The importer writes directly to the database — no background queue is used |

> **Tip:** Run `./biblioteka-cli db-migrate` before your first import if the server has never started and the schema has not been initialized.

---

## Quick start

```bash
# Build the CLI (if not already built)
go build -o biblioteka-cli ./cmd/cli

# Import a Calibre library
./biblioteka-cli calibre-import /path/to/calibre/library

# Import and associate every book with an existing Biblioteka library
./biblioteka-cli calibre-import /path/to/calibre/library <library-id>
```

**Example output:**

```
Calibre import complete:
  Total books:    312
  Imported:       308
  Skipped:        3
  Errors:         1
```

---

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<calibre-library-path>` | Yes | Path to the Calibre library root directory. The command looks for `metadata.db` in this directory. |
| `<library-id>` | No | UUID of an existing Biblioteka [library](administration.md#libraries) to associate every imported book with. When omitted, books are imported without a library association. |

The path is resolved to an absolute path before use. The command exits with an error if:

- The path does not exist or is not a directory.
- `metadata.db` is not found inside the directory.
- The optional `<library-id>` is provided but does not match an existing library.

---

## What gets imported

### Books

Each Calibre book becomes a Biblioteka book record. The following fields are mapped:

| Calibre field | Biblioteka field | Notes |
|---------------|-----------------|-------|
| `title` | `title` | Required; always set |
| `pubdate` | `publication_date` | Skipped when Calibre's sentinel date (`0101-01-01`) is detected |
| `publisher` | `publisher` | Empty when no publisher is set in Calibre |
| `comments.text` | `description` | Empty when no comment is set |
| Language code | `language` | ISO 639 code; empty when not set. Older Calibre databases without a `languages` table are handled gracefully |
| Identifiers (ISBN) | `isbn10`, `isbn13` | `isbn13` takes priority over `isbn10`, which takes priority over `isbn`. Values are normalized and checked for ISBN format/length only |
| Identifier `asin` or `mobi-asin` | `asin` | |
| Identifier `goodreads` or `goodreads-id` | `goodreads_id` | |
| Identifier `google`, `google-id`, or `googlebooks` | `google_books_id` | |
| Identifier `hardcover` or `hardcover-id` | `hardcover_id` | |

### Authors

Authors are resolved via `FindOrCreateAuthor`: if an author with the same name already exists in Biblioteka, the existing record is reused. Otherwise a new author record is created. Author names are imported in display order as stored in Calibre.

### Series

Series are resolved via `FindOrCreateSeries` using the same find-or-create logic. The series position (`series_index` in Calibre) is preserved.

### File formats

Every Calibre format (EPUB, MOBI, AZW3, PDF, etc.) stored in the library becomes a `book_file` record. The file path is constructed as:

```
<calibre-library-path>/<book-path>/<filename>.<ext>
```

File size is read from Calibre's `data.uncompressed_size` column. Note that the files themselves are not moved or copied — only the metadata records are written to the Biblioteka database.

---

## What is NOT imported

| Data | Reason |
|------|--------|
| **Tags** | Not currently imported by `calibre-import`. Re-tag books manually in Biblioteka after import |
| **Ratings** | Biblioteka does not have a ratings field |
| **Custom columns** | Calibre custom columns have no equivalent in Biblioteka |
| **Reading progress** | No equivalent in Biblioteka |
| **Cover images** | Covers are not read from the Calibre library during import. Running `process-file` on the same already-imported EPUB paths will be skipped once those files are indexed, so it is not a supported post-import cover backfill workflow |

> Calibre tags are not imported. After the import completes, use the [Tags API](api-reference.md) or the Biblioteka UI to apply tags to your books.

### Cover backfill after import

If you need covers after a Calibre import, use one of these supported workflows:

1. **Goodreads metadata enrichment:** trigger [`POST /api/books/{id}/metadata/fetch`](api-reference.md#post-apibooksidmetadatafetch-) for each imported book, then review and apply the pending metadata (UI or [`POST /api/books/{id}/metadata/apply`](api-reference.md#post-apibooksidmetadataapply-)). This can populate `cover_image_url` when Goodreads has a cover URL.
2. **Manual cover update:** update each book via [`PUT /api/books/{id}`](api-reference.md#put-apibooksid-) and set `cover_image_url` in the request body.

---

## Deduplication and idempotency

The importer checks whether a book is already indexed by looking up each format's file path in the `book_files` table:

- If **any** format file path already exists in the database, the entire book is skipped (counted as **Skipped**).
- Books with **no file formats** are also skipped — they cannot be deduplicated by path and would be re-imported on every run.

This means the command is safe to run multiple times. Only genuinely new books are imported on subsequent runs.

> **Path stability:** Deduplication is based on the absolute file path of each
> book format. If you move or reorganize your Calibre library to a new
> location after the first import, the original paths will no longer match and
> affected books will be re-imported as new records (the old records remain).
> Run the import from the same path each time, or clean up duplicate records
> manually afterward.

---

## Error handling

Per-book errors are **logged and counted** without aborting the import of remaining books. The final output reports how many books encountered errors.

Common causes of per-book errors:

| Symptom | Likely cause |
|---------|--------------|
| `create book` error | Database constraint violation (e.g. duplicate title with same identifiers) |
| `all N file record(s) failed to create` | Database error while creating `book_files` rows (e.g. constraint or uniqueness conflict) |

**Non-fatal warnings** (do not increment `Errors`):

| Symptom | What happens |
|---------|--------------|
| `failed to find or create author` | The author link is skipped, but the book is still counted as **Imported** |

If `Errors` is non-zero, check the application logs before re-running:

```bash
# If running via Docker Compose, filter for errors and warnings (covers import failures)
docker compose logs --no-log-prefix biblioteka | jq 'select(.level == "ERROR" or .level == "WARN")'
```

For **transient errors** (e.g. a momentary database lock), re-running the command is sufficient — already-imported books are skipped and only failed books are retried. For **permanent errors** (e.g. a database constraint violation), re-running will produce the same failure; investigate the logged error message to determine whether the conflicting record needs to be removed or the Calibre data corrected before retrying.

---

## Associating imported books with a library

Pass a Biblioteka library UUID as the second argument to associate every successfully imported book with that library:

```bash
./biblioteka-cli calibre-import /mnt/calibre my-library-uuid
```

The library must exist before running the import. Create it via the [Administration UI](administration.md#libraries) or the API. If the library ID is invalid the entire import is aborted before any books are written.

Books that fail to link to the library (e.g. due to a transient database error) are logged as warnings. The book record itself is still created — only the library association is missing.

---

## Full example workflow

```bash
# 1. Build the CLI
go build -o biblioteka-cli ./cmd/cli

# 2. (First run only) Initialize the schema
./biblioteka-cli db-migrate

# 3. Create a library in the UI or via the API and copy its UUID

# 4. Run the Calibre import
./biblioteka-cli calibre-import /home/user/Calibre\ Library my-library-uuid

# 5. Review the summary and re-run if there were errors
#    (already-imported books are skipped automatically)

# 6. Re-tag books in Biblioteka as needed — Calibre tags are not imported
```

---

## See also

- [CLI tool reference](metadata.md#cli-tool) — `process-file`, `scan-directory`, and other commands
- [Administration — Libraries](administration.md#libraries) — creating and managing library records
- [Background Jobs — `enrich:goodreads`](background-jobs.md#enrichgoodreads) — how Goodreads metadata (including covers) is fetched
- [Background Jobs](background-jobs.md) — how `scan-directory` and `process:file` work when a worker is running
- [API Reference — `POST /api/books/{id}/metadata/fetch`](api-reference.md#post-apibooksidmetadatafetch-) — enqueue Goodreads metadata fetch jobs
- [API Reference — `PUT /api/books/{id}`](api-reference.md#put-apibooksid-) — manually update `cover_image_url`
- [API Reference](api-reference.md) — managing books, authors, series, and tags via the REST API
