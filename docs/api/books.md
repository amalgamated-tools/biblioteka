<!-- disable-agentic-editing: true -->

# API Reference — Books

[← Back to API Reference](../api-reference.md)

## Books

### `GET /api/books` 🔒

List books (summary objects — no nested authors, series, tags, or files), with pagination. Results are sorted by `title` ascending.

When the `query` parameter is provided, its value is trimmed first. If the trimmed value is non-empty, the endpoint searches `title` and `description` and returns only matching books, still paginated. The search implementation depends on the configured database backend:

- **SQLite:** Uses an FTS5 virtual table (`books_fts`). Each whitespace-separated token is matched as a prefix — for example, `"found"` also matches `"Foundation"`. Multi-token queries require all tokens to match somewhere in the combined title+description document (implicit AND), so `"frank dune"` returns books where `title` contains "Frank" and `description` contains "Dune" (or any other cross-field token split). Tokens made entirely of punctuation or special characters are silently dropped; if no valid tokens remain, zero results are returned.
- **PostgreSQL:** Uses `ILIKE` with GIN trigram indexes (`pg_trgm`). This is a case-insensitive literal substring match — `%query%`. LIKE special characters (`%`, `_`, `\`) in the query are escaped and treated as literals — they do not act as wildcards. Does not support prefix expansion across the combined document.

If `query` is omitted or trims to an empty string (for example, `query=%20%20`), all books are returned.

**Query parameters:**

| Parameter | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `query`   | string  | _(none)_ | Search term(s) across `title` and `description`. SQLite uses FTS5 (prefix-matching, cross-field AND); PostgreSQL uses `ILIKE` (literal-substring; LIKE special characters are escaped). The value is trimmed; only non-empty trimmed values trigger search. If omitted or blank after trimming, all books are returned. |
| `limit`   | integer | `50`    | Maximum books to return (capped at `200`) |
| `offset`  | integer | `0`     | Number of books to skip |

**Response body (`200`):** Paginated books object.

| Field    | Type    | Description |
|----------|---------|-------------|
| `books`  | array   | Array of book summary objects for this page |
| `total`  | integer | Total number of books across all pages |
| `limit`  | integer | Effective limit used |
| `offset` | integer | Effective offset used |

```json
{
  "books": [
    {
      "id": "<id>",
      "title": "Pride and Prejudice",
      "description": null,
      "asin": null,
      "isbn10": null,
      "isbn13": "978-0-14-143951-8",
      "goodreads_id": null,
      "hardcover_id": null,
      "google_books_id": null,
      "publication_date": "1813-01-28",
      "publisher": "Penguin Classics",
      "language": "en",
      "cover_image_url": null,
      "created_at": "2026-03-14T02:00:00Z",
      "updated_at": "2026-03-14T02:00:00Z"
    }
  ],
  "total": 142,
  "limit": 50,
  "offset": 0
}
```

**Book summary object** fields:

| Field              | Type    | Description |
|--------------------|---------|-------------|
| `id`               | string  | Opaque resource ID |
| `title`            | string  | Book title |
| `description`      | string\|null | Synopsis |
| `asin`             | string\|null | Amazon ASIN |
| `isbn10`           | string\|null | ISBN-10 |
| `isbn13`           | string\|null | ISBN-13 |
| `goodreads_id`     | string\|null | Goodreads book ID |
| `hardcover_id`     | string\|null | Hardcover book ID |
| `google_books_id`  | string\|null | Google Books ID |
| `publication_date` | string\|null | ISO date (`YYYY-MM-DD`) |
| `publisher`        | string\|null | Publisher name |
| `language`         | string\|null | BCP 47 language tag |
| `cover_image_url`  | string\|null | Cover art URL or `data:image/...;base64,...` string. For EPUB books imported with an embedded cover, this field is automatically set to a base64-encoded `data:` URL; the encoded value can be up to 20 MB. For other formats or manually set covers it is a plain `https://` URL. Prefer the [OPDS cover endpoint](opds.md#get-opds-covers-bookid--book-cover-image) to serve cover images rather than decoding this field in application code. |
| `created_at`       | string  | ISO 8601 creation timestamp |
| `updated_at`       | string  | ISO 8601 last-updated timestamp |

---

### `POST /api/books` 🔒

Create a book.

**Request body:**

| Field              | Type    | Required | Description |
|--------------------|---------|----------|-------------|
| `title`            | string  | ✓        | Book title |
| `description`      | string  |          | Synopsis or description |
| `asin`             | string  |          | Amazon ASIN |
| `isbn10`           | string  |          | ISBN-10 |
| `isbn13`           | string  |          | ISBN-13 |
| `goodreads_id`     | string  |          | Goodreads book ID |
| `hardcover_id`     | string  |          | Hardcover book ID |
| `google_books_id`  | string  |          | Google Books ID |
| `publication_date` | string  |          | ISO date (`YYYY-MM-DD`) |
| `publisher`        | string  |          | Publisher name |
| `language`         | string  |          | BCP 47 language tag (e.g., `"en"`) |
| `cover_image_url`  | string  |          | Cover art URL or a `data:image/...;base64,...` string. Accepts both a plain `https://` URL and a base64-encoded `data:` URL. For EPUB books, the import pipeline sets this field automatically from the embedded cover; for all other formats, set it manually. The 20 MB decoded-size cap applies on read — see [book summary object](#get-apibooks) for details. |

**Response:** `201 Created` with the full book object (includes `authors`, `series`, `tags`, `files` arrays).

---

### `GET /api/books/{id}` 🔒

Get a single book with its full details: authors, series entries, tags, and associated files.

**Book detail object:**

```json
{
  "id": "<id>",
  "title": "Pride and Prejudice",
  "description": null,
  "asin": null,
  "isbn10": null,
  "isbn13": "978-0-14-143951-8",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null,
  "publication_date": "1813-01-28",
  "publisher": "Penguin Classics",
  "language": "en",
  "cover_image_url": null,
  "authors": [
    {
      "id": "<id>",
      "name": "Jane Austen",
      "goodreads_id": null,
      "hardcover_id": null,
      "google_books_id": null,
      "image_url": null,
      "created_at": "2026-03-14T02:00:00Z",
      "updated_at": "2026-03-14T02:00:00Z"
    }
  ],
  "series": [
    {
      "series": {
        "id": "<id>",
        "name": "Discworld",
        "goodreads_id": null,
        "hardcover_id": null,
        "google_books_id": null,
        "created_at": "2026-03-14T02:00:00Z",
        "updated_at": "2026-03-14T02:00:00Z"
      },
      "position": 1
    }
  ],
  "tags": [
    {
      "id": "<id>",
      "name": "science fiction",
      "created_at": "2026-03-14T02:00:00Z",
      "updated_at": "2026-03-14T02:00:00Z"
    }
  ],
  "files": [],
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

The flat book fields (`id`, `title`, `description`, etc.) are described in the [book summary object](#get-apibooks) table. The nested arrays are described below.

**`authors[]`** — each element is an [author object](authors.md#post-apiauthors).

**`series[]`** — each element is a series entry object:

| Field              | Type    | Description |
|--------------------|---------|-------------|
| `series`           | object  | Series object (id, name, external IDs, timestamps) |
| `series.id`        | string  | Opaque series ID |
| `series.name`      | string  | Series name |
| `series.goodreads_id` | string? | Goodreads series ID; `null` when absent |
| `series.hardcover_id` | string? | Hardcover series ID; `null` when absent |
| `series.google_books_id` | string? | Google Books series ID; `null` when absent |
| `series.created_at` | string | Creation timestamp (ISO 8601) |
| `series.updated_at` | string | Last update timestamp (ISO 8601) |
| `position`         | number? | Position of this book in the series (e.g. `1` for book one); `null` when unset |

**`tags[]`** — each element is a [tag object](tags.md#post-apitags). Tags embedded here mirror the response from [`GET /api/books/{id}/tags`](#get-apibooksidtags) and are included for convenience so a single request returns the complete book record without extra round-trips.

**`files[]`** — each element is a [book file object](#get-apibook-filesid).

---

### `PUT /api/books/{id}` 🔒

Update a book's metadata. This is a **full replacement** — every field not included in the request body is cleared. Fetch the current book with `GET /api/books/{id}` first to avoid accidentally clearing fields.

**Request body:** Same fields as [`POST /api/books`](#post-apibooks). `title` is the only required field.

**Response:** `200 OK` with the updated book detail object (same shape as `GET /api/books/{id}`).

**Error responses:**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | `title` is missing or empty |
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | Book with the given `{id}` does not exist |
| `500 Internal Server Error` | Unexpected server error |

> **Cover images:** To update `cover_image_url` to a plain URL, include the full URL in the request. To preserve the existing cover, include the current `cover_image_url` value (even if it is a large `data:` URL) in the request body. To remove the cover, omit `cover_image_url` or set it to `null`.

---

### `DELETE /api/books/{id}` 🔒 **Admin**

Delete a book. Returns `204 No Content`.

> **Cascade:** Deleting a book also removes all associated `book_files`, `book_authors`, `book_series`, and `library_books` records. See [Cascade Deletion Summary](../database-schema.md#cascade-deletion-summary).

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Book not found |

---

### `POST /api/books/upload` 🔒

Upload a book file to a library. The file is staged on disk and processed **asynchronously** by a background worker that extracts metadata, organizes the file into the library's directory layout, and creates a book record. The endpoint returns `202 Accepted` immediately — the book will appear in the library once processing completes.

> **Requires:** Background processing must be configured. If background processing is not configured, or the upload job cannot be enqueued for background processing, the endpoint returns `503 Service Unavailable`.

**Content type:** `multipart/form-data`

**Supported file types:** `.epub`, `.mobi`, `.azw3`, `.pdf`

**Maximum upload size:** 500 MB

**Form fields:**

| Field         | Type   | Required | Description |
|---------------|--------|----------|-------------|
| `file`        | file   | ✓        | The book file to upload |
| `library_id`  | string | ✓        | ID of the target library |
| `title`       | string |          | Title override — takes precedence over metadata extracted from the file |
| `author`      | string |          | Author override — takes precedence over extracted metadata |
| `description` | string |          | Description override |
| `isbn`        | string |          | ISBN override (ISBN-10 or ISBN-13); validated immediately and returns `400` if invalid |
| `language`    | string |          | Language override |
| `publisher`   | string |          | Publisher override |

**Response body (`202 Accepted`):**

```json
{
  "message": "file accepted for processing",
  "file_name": "my-book.epub",
  "file_type": "epub",
  "library_id": "<library-id>"
}
```

| Field        | Type   | Description |
|--------------|--------|-------------|
| `message`    | string | Human-readable status message |
| `file_name`  | string | Basename of the uploaded file name, with any path components removed |
| `file_type`  | string | Lowercased file extension from the uploaded filename: `epub`, `mobi`, `azw3`, or `pdf` |
| `library_id` | string | ID of the target library |

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Missing `library_id` or `file`; unsupported file type; or invalid ISBN |
| `401` | Missing or invalid authentication token |
| `404` | Library with the given `library_id` not found |
| `413` | File exceeds the 500 MB limit |
| `500` | Server error while staging the file or querying library configuration |
| `503` | Background processing is not configured, or the job queue is unavailable |

**Example curl:**

```bash
curl -X POST https://your-server/api/books/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@/path/to/book.epub" \
  -F "library_id=<library-id>" \
  -F "title=My Book" \
  -F "author=Jane Smith"
```

---

### `GET /api/books/{id}/metadata` 🔒

Return the most recent **pending** (unreviewed) metadata candidate for a book. Metadata candidates are fetched from Goodreads by the background enrichment job triggered via [`POST /api/books/{id}/metadata/fetch`](#post-apibooksidmetadatafetch).

**Path parameter:** `{id}` — book ID.

**Response body (`200`):**

```json
{
  "id": "d1e2f3...",
  "book_id": "a1b2c3...",
  "status": "pending",
  "source": "goodreads",
  "title": "Dune",
  "description": "A science fiction epic set on the desert planet Arrakis.",
  "asin": "B00B7NWYBW",
  "isbn10": "0441013597",
  "isbn13": "9780441013593",
  "goodreads_id": "234225",
  "hardcover_id": null,
  "google_books_id": null,
  "publication_date": "1965-08-01",
  "publisher": "Chilton Books",
  "language": "en",
  "cover_image_url": "https://i.gr-assets.com/images/S/...",
  "author_name": "Frank Herbert",
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Opaque metadata record ID |
| `book_id` | string? | ID of the parent book; `null` if the book was deleted |
| `status` | string | Always `"pending"` for this endpoint (`"applied"` or `"rejected"` records are not returned) |
| `source` | string | Always `"goodreads"` |
| `title` | string? | Proposed book title |
| `description` | string? | Book description / blurb |
| `asin` | string? | Amazon Standard Identification Number |
| `isbn10` | string? | ISBN-10 |
| `isbn13` | string? | ISBN-13 |
| `goodreads_id` | string? | Goodreads book ID |
| `hardcover_id` | string? | Hardcover book ID |
| `google_books_id` | string? | Google Books volume ID |
| `publication_date` | string? | Publication date (ISO 8601 date string, e.g. `"1965-08-01"`) |
| `publisher` | string? | Publisher name |
| `language` | string? | ISO 639-1 language code (e.g. `"en"`) |
| `cover_image_url` | string? | URL of the cover image |
| `author_name` | string? | Author name as provided by the metadata source |
| `created_at` | string | Timestamp when the record was created (ISO 8601) |
| `updated_at` | string | Timestamp when the record was last updated (ISO 8601) |

> **User isolation:** Only metadata candidates owned by the authenticated user are returned.

| Status | Description |
|--------|-------------|
| `200 OK` | Pending metadata candidate |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found, or no pending metadata candidate exists for this user |
| `500 Internal Server Error` | Database error |

---

### `POST /api/books/{id}/metadata/fetch` 🔒

Enqueue a background job that searches Goodreads for metadata matching this book. When the job completes, the result appears as a pending candidate retrievable via [`GET /api/books/{id}/metadata`](#get-apibooksidmetadata). Progress can be streamed in real time via [`GET /api/books/{id}/metadata/events`](#get-apibooksidmetadataevents).

This endpoint is **idempotent**: if a pending candidate already exists for the user, or if the job is already running, the server returns `202` without enqueueing a duplicate job.

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response body (`202`):**

```json
{ "task_id": "abc123...", "status": "enqueued" }
```

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Background task ID; **omitted** when `status` is `"already_exists"` or `"already_running"` |
| `status` | string | `"enqueued"` — job queued; `"already_exists"` — pending candidate already exists; `"already_running"` — job is already in the queue |

> A successful enqueue is recorded in the audit log as `metadata.fetch_requested`.

| Status | Description |
|--------|-------------|
| `202 Accepted` | Job enqueued or already in progress |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found |
| `500 Internal Server Error` | Failed to enqueue job or check existing state |
| `503 Service Unavailable` | Background worker not configured |

---

### `GET /api/books/{id}/metadata/events` 🔒

Open a [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) (SSE) stream that delivers real-time progress updates for the active metadata enrichment job. The recommended workflow is:

1. Open this SSE connection.
2. After the connection is established, POST to `/api/books/{id}/metadata/fetch` to start the job.
3. Listen for events on the SSE stream; close the connection on a terminal event.

**Path parameter:** `{id}` — book ID.

**Response headers:**

| Header | Value |
|--------|-------|
| `Content-Type` | `text/event-stream` |
| `Cache-Control` | `no-cache` |
| `Connection` | `keep-alive` |

Each SSE message is a JSON object in the `data:` field:

```
data: {"event":"progress","source":"goodreads","message":"Searching Goodreads..."}

data: {"event":"complete","source":"goodreads","metadata_id":"d1e2f3..."}
```

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | `"progress"` — intermediate update; `"complete"` — job succeeded; `"error"` — job failed; `"not_found"` — no Goodreads match found |
| `source` | string | Always `"goodreads"` |
| `message` | string | Human-readable status message (present on `progress`, `error`, and `not_found`) |
| `step` | string | Optional machine-readable progress step (typically present on `progress` events), for example `searching_isbn13` or `searching_title` |
| `metadata_id` | string | ID of the newly created metadata candidate (present on `complete` only) |

The stream also sends `: heartbeat` comment lines every 15 seconds to keep the connection alive through proxies. The write deadline is reset on every heartbeat, so the connection remains open until a terminal event (`complete`, `error`, `not_found`) is sent or the client disconnects; proxies or other infrastructure may impose additional timeouts.

| Status | Description |
|--------|-------------|
| `200 OK` | SSE stream opened |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found |
| `500 Internal Server Error` | Response writer does not support streaming |
| `503 Service Unavailable` | Event streaming not configured (Redis pub/sub not available) |

---

### `POST /api/books/{id}/metadata/apply` 🔒

Apply the pending metadata candidate to the book. All non-null, non-empty fields from the candidate overwrite the corresponding book fields (falling back to the existing book values for any blank or absent metadata field). Authors are **not** updated by this endpoint — manage author associations via [`PUT /api/books/{id}/authors`](#put-apibooksidauthors).

After a successful apply, the candidate's `status` changes to `"applied"` and the pending record no longer appears in [`GET /api/books/{id}/metadata`](#get-apibooksidmetadata).

> **Note:** The **Apply All** button in the Metadata panel uses this endpoint directly. For a field-by-field workflow, copy individual fields into the edit form, save via `PUT /api/books/{id}`, and call `POST /api/books/{id}/metadata/reject` to dismiss the candidate. For programmatic or CLI-driven one-shot applies, call this endpoint directly.

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response body (`200`):** Updated book object (same schema as the items in [`GET /api/books`](#get-apibooks) — core fields only, without `authors`, `series`, `tags`, or `files`).

> A successful apply is recorded in the audit log as `metadata.applied`.

| Status | Description |
|--------|-------------|
| `200 OK` | Metadata applied; updated book returned |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found, or no pending metadata candidate for this user |
| `500 Internal Server Error` | Failed to update the book or mark the candidate as applied |

---

### `POST /api/books/{id}/metadata/reject` 🔒

Discard the pending metadata candidate without modifying the book. The candidate's `status` changes to `"rejected"` and it no longer appears in [`GET /api/books/{id}/metadata`](#get-apibooksidmetadata).

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response:** `204 No Content`

> A successful rejection is recorded in the audit log as `metadata.rejected`.

| Status | Description |
|--------|-------------|
| `204 No Content` | Candidate rejected |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found, or no pending metadata candidate for this user |
| `500 Internal Server Error` | Database error |

---

### `GET /api/books/{id}/metadata/ai` 🔒

Return the most recent pending AI enrichment candidate for a book. AI enrichment candidates are created by the background job triggered via [`POST /api/books/{id}/metadata/ai-fetch`](#post-apibooksidmetadataai-fetch).

**Path parameter:** `{id}` — book ID.

**Response body (`200`):**

```json
{
  "id": "a1b2c3...",
  "book_id": "b1c2d3...",
  "status": "pending",
  "provider": "ollama",
  "model": "llama3.2",
  "suggested_tags": ["Science Fiction", "Space Opera", "Classic"],
  "reading_level": "adult",
  "generated_description": "A sweeping interstellar epic...",
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Opaque enrichment record ID |
| `book_id` | string? | ID of the parent book; `null` if the book was deleted |
| `status` | string | Always `"pending"` for this endpoint (`"applied"` or `"rejected"` records are not returned) |
| `provider` | string | LLM provider that generated this enrichment (e.g. `"ollama"`) |
| `model` | string | Model name used for generation (e.g. `"llama3.2"`) |
| `suggested_tags` | string[] | Tags suggested by the model; never `null` (empty array when none) |
| `reading_level` | string? | One of `"children"`, `"young_adult"`, `"adult"`, `"academic"`; `null` when not determined |
| `generated_description` | string? | 2–3 sentence catalog description; `null` when the model did not return one |
| `created_at` | string | Timestamp when the record was created (ISO 8601) |
| `updated_at` | string | Timestamp when the record was last updated (ISO 8601) |

> **User isolation:** Only enrichment records owned by the authenticated user are returned.

| Status | Description |
|--------|-------------|
| `200 OK` | Pending AI enrichment candidate |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | No pending AI enrichment exists for this book and user |
| `500 Internal Server Error` | Database error |

---

### `POST /api/books/{id}/metadata/ai-fetch` 🔒

Enqueue a background `enrich:ai` job that calls the configured LLM provider to generate metadata for this book. When the job completes, the result appears as a pending candidate retrievable via [`GET /api/books/{id}/metadata/ai`](#get-apibooksidmetadataai). Progress events are published to the same SSE stream used by Goodreads enrichment ([`GET /api/books/{id}/metadata/events`](#get-apibooksidmetadataevents)).

This endpoint is **idempotent**: if a pending AI enrichment already exists for the user, or if the job is already running, the server returns `202` without enqueueing a duplicate job.

**Requires** a configured and enabled LLM provider (see [`PUT /api/config/llm`](config.md#put-apiconfigllm--admin--jwt-only)). Returns `503` when the LLM provider or background worker is not available.

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response body (`202`):**

```json
{ "task_id": "abc123...", "status": "enqueued" }
```

| Field | Type | Description |
|-------|------|-------------|
| `task_id` | string | Background task ID; **omitted** when `status` is `"already_exists"` or `"already_running"` |
| `status` | string | `"enqueued"` — job queued; `"already_exists"` — pending candidate already exists; `"already_running"` — job is already in the queue |

> A successful enqueue is recorded in the audit log as `ai_enrichment.fetch_requested`.

| Status | Description |
|--------|-------------|
| `202 Accepted` | Job enqueued or already in progress |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found |
| `500 Internal Server Error` | Failed to enqueue job or check existing state |
| `503 Service Unavailable` | LLM provider not configured, or background worker not available |

---

### `POST /api/books/{id}/metadata/ai-apply` 🔒

Apply the pending AI enrichment candidate to the book. The apply logic:

- **Tags**: union-merges `suggested_tags` with the book's existing tags. New tags are created via `FindOrCreate` if they do not yet exist; no existing tags are removed.
- **Description**: sets `description` only when the book has no existing description and the enrichment has a `generated_description`.

After a successful apply, the candidate's `status` changes to `"applied"` and it no longer appears in [`GET /api/books/{id}/metadata/ai`](#get-apibooksidmetadataai).

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response body (`200`):** The updated AI enrichment record (same schema as [`GET /api/books/{id}/metadata/ai`](#get-apibooksidmetadataai), with `status` set to `"applied"`).

> A successful apply is recorded in the audit log as `ai_enrichment.applied`.

| Status | Description |
|--------|-------------|
| `200 OK` | Enrichment applied; updated record returned |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Book not found, or no pending AI enrichment for this user |
| `409 Conflict` | Enrichment is no longer pending (already applied or rejected) |
| `500 Internal Server Error` | Failed to find or create tags, update book, or mark enrichment as applied |

---

### `POST /api/books/{id}/metadata/ai-reject` 🔒

Discard the pending AI enrichment candidate without modifying the book. The candidate's `status` changes to `"rejected"`.

**Path parameter:** `{id}` — book ID.

**Request body:** none

**Response:** `204 No Content`

> A successful rejection is recorded in the audit log as `ai_enrichment.rejected`.

| Status | Description |
|--------|-------------|
| `204 No Content` | Enrichment rejected |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | No pending AI enrichment for this book and user |
| `409 Conflict` | Enrichment is no longer pending (already applied or rejected) |
| `500 Internal Server Error` | Database error |

---

### `GET /api/books/{id}/authors` 🔒

List the authors linked to a book. Results are sorted by `name` ascending.

**Response body (`200`):** Array of [author objects](authors.md#post-apiauthors).

---

### `PUT /api/books/{id}/authors` 🔒

Replace the author list for a book.

**Request body:**

```json
{ "author_ids": ["<id>", "<id2>"] }
```

**Response body (`200`):** Updated array of [author objects](authors.md#post-apiauthors) — identical to the response from `GET /api/books/{id}/authors`.

---

### `GET /api/books/{id}/series` 🔒

List the series entries linked to a book. Results are sorted by series `name` ascending.

**Response body (`200`):** Array of series entry objects.

**Series entry object:**

| Field      | Type         | Description |
|------------|--------------|-------------|
| `series`   | object       | The [series object](series.md#post-apiseries) |
| `position` | number\|null | Position of this book within the series (e.g. `1`, `2.5`); `null` when unordered |

**Example:**

```json
[
  {
    "series": {
      "id": "<id>",
      "name": "Discworld",
      "goodreads_id": null,
      "hardcover_id": null,
      "google_books_id": null,
      "created_at": "2026-03-14T02:00:00Z",
      "updated_at": "2026-03-14T02:00:00Z"
    },
    "position": 1
  }
]
```

---

### `PUT /api/books/{id}/series` 🔒

Replace the series entries for a book.

**Request body:**

```json
{
  "entries": [
    { "series_id": "<id>", "position": 1 }
  ]
}
```

**Response body (`200`):** Updated array of series entry objects — identical to the response from `GET /api/books/{id}/series`.

---

### `GET /api/books/{id}/tags` 🔒

List the tags linked to a book. Results are sorted by `name` ascending.

**Response body (`200`):** Array of [tag objects](tags.md#post-apitags).

---

### `PUT /api/books/{id}/tags` 🔒

Replace the tag list for a book.

**Request body:**

```json
{ "tag_ids": ["<id>", "<id2>"] }
```

Pass an empty array (`[]`) to remove all tags from the book.

**Response body (`200`):** Updated array of [tag objects](tags.md#post-apitags) — identical to the response from `GET /api/books/{id}/tags`.

---

### `GET /api/books/{id}/files` 🔒

List all files associated with a book.

**Response body (`200`):** Array of [book-file objects](#get-apibook-filesid).

---

### `POST /api/books/{id}/files` 🔒

Attach a file record to a book.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file_type` | string | ✓ | Format identifier: `epub`, `mobi`, `pdf`, or `azw3` |
| `file_name` | string | ✓ | File name on disk |
| `file_path` | string | ✓ | Absolute path to the file |
| `file_size` | integer | | File size in bytes |
| `file_hash` | string | | Content hash (e.g. `sha256:abc123…`) |

**Response:** `201 Created` with the new book-file object (same shape as `GET /api/book-files/{id}`).

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Missing `file_type`, `file_name`, or `file_path` |
| `404` | Book with the given `{id}` not found |
| `500` | Unexpected server error |

---


## Book Files

Book files are created automatically when the background scanner processes a library path.

### `GET /api/book-files/{id}` 🔒

Get a single book file by ID.

**Book file object:**

```json
{
  "id": "<id>",
  "book_id": "<book_id>",
  "file_type": "epub",
  "file_name": "pride-and-prejudice.epub",
  "file_size": 524288,
  "file_hash": "sha256:abc123...",
  "file_path": "/mnt/books/fiction/Austen/Pride and Prejudice.epub",
  "download_count": 3,
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

| Field            | Type     | Description |
|------------------|----------|-------------|
| `id`             | string   | Opaque book-file ID |
| `book_id`        | string   | ID of the parent book |
| `file_type`      | string   | Format identifier: `epub`, `mobi`, `pdf`, or `azw3` |
| `file_name`      | string   | Filename on disk (basename only) |
| `file_size`      | integer  | File size in bytes |
| `file_hash`      | string?  | Content hash (e.g. `sha256:abc123…`); `null` when not recorded |
| `file_path`      | string   | Absolute path to the file on the server's filesystem |
| `download_count` | integer  | Number of download requests initiated for this file |
| `created_at`     | string   | Timestamp when the record was created (ISO 8601) |
| `updated_at`     | string   | Timestamp when the record was last updated (ISO 8601) |

---

### `GET /api/book-files/{id}/download` 🔒

Download the raw file content for a book file. The response is the binary file content with an appropriate `Content-Type` header. The `download_count` for the file is incremented best-effort at the start of each download request; a counter failure does not block the response. The endpoint supports HTTP range requests (`Range` header); clients may receive `206 Partial Content` for partial or resumed downloads.

**Path parameter:** `{id}` — book file ID.

**Response `200 OK`:**

| Header | Value |
|--------|-------|
| `Content-Type` | MIME type matching the file format (e.g. `application/epub+zip` for EPUB, `application/pdf` for PDF) |
| `Content-Disposition` | `attachment; filename="<file_name>"` |

**Error responses:**

| Status | Meaning |
|--------|---------|
| `403 Forbidden` | File path is outside all configured library roots |
| `404 Not Found` | Book file record not found, or file no longer exists on disk |
| `500 Internal Server Error` | Unexpected server error |

---

### `POST /api/book-files/{id}/email` 🔒

Send a book file as an email attachment to a specified address. Requires SMTP to be configured in the server settings. The file must be ≤ 25 MiB; larger files are rejected with `413`.

**Path parameter:** `{id}` — book file ID.

**Request body:**

| Field | Type   | Required | Description |
|-------|--------|----------|-------------|
| `to`  | string | ✓        | Recipient email address (RFC 5322 format) |

**Response `200 OK`:**

```json
{ "message": "Email sent successfully" }
```

**Error responses:**

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | Missing or invalid `to` address; SMTP not configured or misconfigured |
| `403 Forbidden` | File path is outside all configured library roots |
| `404 Not Found` | Book file record not found, or file no longer exists on disk |
| `413 Request Entity Too Large` | File exceeds the 25 MiB attachment limit |
| `500 Internal Server Error` | Unexpected server error (path validation, SMTP validation, message build, or file read failure) |
| `502 Bad Gateway` | SMTP server rejected or failed to deliver the message |

> A successful send is recorded in the audit log as `book_file.emailed`.

---

### `DELETE /api/book-files/{id}` 🔒 **Admin**

Delete a book file record (does not delete the file from disk). Returns `204 No Content`.

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Book file not found |

---

