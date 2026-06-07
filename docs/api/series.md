<!-- disable-agentic-editing: true -->

# API Reference — Series

[← Back to API Reference](../api-reference.md)

## Series

### `GET /api/series` 🔒

List all series. Results are sorted by `name` ascending.

**Response body (`200`):** Array of series objects (see below).

---

### `POST /api/series` 🔒

Create a series.

**Request body:**

| Field             | Type   | Required | Description |
|-------------------|--------|----------|-------------|
| `name`            | string | ✓        | Series name (must be unique, case-insensitive) |
| `goodreads_id`    | string |          | Goodreads series ID |
| `hardcover_id`    | string |          | Hardcover series ID |
| `google_books_id` | string |          | Google Books series ID |

> **Name normalization:** Before storage, the server trims leading/trailing whitespace and collapses any internal whitespace run to a single space. Capitalization is preserved. For example, `"  The  Lord  of  the  Rings  "` is stored as `"The Lord of the Rings"`. A name that is blank after normalization is rejected with `400`.

**Response:** `201 Created` with the series object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `409` | A series with that name already exists (comparison is case-insensitive) |
| `500` | Unexpected server error |

**Series object:**

```json
{
  "id": "<id>",
  "name": "The Lord of the Rings",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null,
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

---

### `GET /api/series/{id}` 🔒

Get a single series by ID.

**Response body (`200`):** Series object (same shape as the object in [`POST /api/series`](#post-apiseries)).

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Series not found |

---

### `PUT /api/series/{id}` 🔒

Update a series (full update).

**Request body:** Same fields as `POST /api/series`. The same name normalization (whitespace trimming and collapsing) applies.

**Response body (`200`):** Updated series object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `404` | Series not found |
| `409` | A series with that name already exists (comparison is case-insensitive) |

---

### `DELETE /api/series/{id}` 🔒 **Admin**

Delete a series. Returns `204 No Content`.

> **Cascade:** Deleting a series also removes all `book_series` join entries for that series. Books themselves are **not** deleted. See [Cascade Deletion Summary](../database-schema.md#cascade-deletion-summary).

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Series not found |

---

### `GET /api/series/{id}/books` 🔒

List all books in a series, with pagination. Results are ordered by series position ascending, then by `title` ascending. Books with no assigned position appear last on PostgreSQL and first on SQLite.

**Path parameters:** `{id}` — Series resource ID.

**Query parameters:**

| Parameter | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `limit`   | integer | `50`    | Maximum books to return (capped at `200`) |
| `offset`  | integer | `0`     | Number of books to skip |

**Response body (`200`):** Paginated books object (same envelope and book summary shape as [`GET /api/books`](books.md#get-apibooks)).

If no associated books are found (`total: 0`), Biblioteka first checks whether the series ID exists. If the series does not exist, the response is `404 Not Found`; otherwise, the response is `200 OK` with an empty `books` array and `total: 0`.

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Series not found when no books are associated with the given ID and the series does not exist |
| `500` | Unexpected server error |

---

