<!-- disable-agentic-editing: true -->

# API Reference — OPDS

[← Back to API Reference](../api-reference.md)

## OPDS Credentials

Each user can configure one set of OPDS credentials (a separate username and password) for use with OPDS reading apps. Credentials are stored as a bcrypt hash. See the [OPDS Catalog guide](opds.md) for the full feature overview.

### `GET /api/opds/credentials` 🔒 **JWT only**

Return the current user's OPDS credential.

**Response body (`200`):**

```json
{
  "username": "alice",
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Returns credential |
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No OPDS credential configured |

---

### `PUT /api/opds/credentials` 🔒 **JWT only**

Create or replace the current user's OPDS credential. If a credential already exists it is updated in-place; the username and hashed password are both replaced.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | ✓ | OPDS username (case-insensitive, trimmed) |
| `password` | string | ✓ | OPDS password (min 8 chars) |

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Returns the updated credential |
| `400 Bad Request` | Missing or invalid fields |
| `401 Unauthorized` | Missing or invalid JWT |
| `409 Conflict` | Username already taken by another user |

**Response body (`200`):** Same shape as the GET response above.

---

### `DELETE /api/opds/credentials` 🔒 **JWT only**

Delete the current user's OPDS credential. Any OPDS client using those credentials will subsequently receive `401 Unauthorized`.

**Responses:**

| Status | Description |
|--------|-------------|
| `204 No Content` | Credential deleted |
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No OPDS credential configured |

---

## OPDS Catalog

The OPDS catalog is served under `/opds` (not under `/api`). It uses **HTTP Basic Authentication** with the OPDS-specific credentials set via [the credentials endpoints above](#opds-credentials)—not the account JWT.

All responses are `application/atom+xml` Atom feeds compliant with [OPDS 1.2](https://specs.opds.io/opds-1.2). Even error responses are returned as Atom XML so that OPDS clients can parse them. See the [OPDS Catalog guide](opds.md) for setup instructions and client examples.

### `GET /opds` — Root catalog

Navigation feed listing all catalog sections (All Books, Recent Books, Authors, Series). Includes a link to the OpenSearch description document.

**Auth:** HTTP Basic (`WWW-Authenticate: Basic realm="Biblioteka OPDS"`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=navigation`

---

### `GET /opds/all` — All books

Acquisition feed of all books, paginated (50 per page).

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`

---

### `GET /opds/recent` — Recent books

Acquisition feed of books ordered by most recently added, paginated (50 per page).

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`

---

### `GET /opds/authors` — Authors list

Navigation feed of all authors, paginated (50 per page).

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=navigation`

---

### `GET /opds/authors/{id}` — Books by author

Acquisition feed of books by the specified author, paginated (50 per page).

**Path parameters:** `{id}` — author resource ID

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Acquisition feed |
| `404 Not Found` | Author not found |

---

### `GET /opds/series` — Series list

Navigation feed of all series, paginated (50 per page).

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=navigation`

---

### `GET /opds/series/{id}` — Books in series

Acquisition feed of books in the specified series, paginated (50 per page).

**Path parameters:** `{id}` — series resource ID

**Query parameters:** `page` (integer, default `1`)

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Acquisition feed |
| `404 Not Found` | Series not found |

---

### `GET /opds/search` — OpenSearch description

When called **without** the `q` parameter, returns an [OpenSearch description document](https://opensearch.org) that OPDS clients use to discover the search template.

**Response content-type:** `application/opensearchdescription+xml`

---

### `GET /opds/search?q={query}` — Search results

Acquisition feed of books matching the query, paginated (50 per page).

**Query parameters:**

| Parameter | Description |
|-----------|-------------|
| `q` | Search query (title, author, description) |
| `page` | Page number (default `1`) |

**Response content-type:** `application/atom+xml;profile=opds-catalog;kind=acquisition`

---

### `GET /opds/download/{file-id}` — Download book file

Streams a book file to the client with the correct `Content-Type` and `Content-Disposition: attachment` header.

**Path parameters:** `{file-id}` — book file resource ID (from the `rel="http://opds-spec.org/acquisition"` link in an acquisition feed entry)

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | File stream |
| `404 Not Found` | File not found |

---

### `GET /opds/covers/{bookID}` — Book cover image

Serve a book's cover image. When the stored `cover_image_url` is a `data:` URL (as set automatically for EPUB files during import), the endpoint decodes the base64 payload and streams the image bytes with the correct `Content-Type`. When it is a plain `https://` URL the client receives a `307` redirect. OPDS clients encounter this URL through the `rel="http://opds-spec.org/image"` link in acquisition feed entries and do not need to construct it manually.

**Path parameters:** `{bookID}` — book resource ID

**Auth:** HTTP Basic (same credentials as other `/opds` endpoints)

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Image bytes; `Content-Type` is set to the detected image MIME type (e.g. `image/jpeg`, `image/png`) |
| `307 Temporary Redirect` | Cover URL is a plain `https://` URL; client is redirected there. Non-HTTPS URLs (e.g. `http://`) stored in `cover_image_url` are rejected and return `404` instead |
| `404 Not Found` | Book not found, no cover image set, or stored `cover_image_url` is rejected because it is not HTTPS |
| `500 Internal Server Error` | Stored data URL is malformed or its payload is not a valid image |

See [Cover images](opds.md#cover-images) in the OPDS guide for MIME-type detection rules and details on the `data:` URL content-sniffing behaviour.

---

