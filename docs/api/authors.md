<!-- disable-agentic-editing: true -->

# API Reference — Authors

[← Back to API Reference](../api-reference.md)

## Authors

### `GET /api/authors` 🔒

List all authors. Results are sorted by `name` ascending.

**Response body (`200`):** Array of author objects (see below).

---

### `POST /api/authors` 🔒

Create an author.

**Request body:**

| Field            | Type   | Required | Description |
|------------------|--------|----------|-------------|
| `name`           | string | ✓        | Author name (must be unique, case-insensitive — `"Jane Austen"` and `"jane austen"` conflict) |
| `goodreads_id`   | string |          | Goodreads author ID |
| `hardcover_id`   | string |          | Hardcover author ID |
| `google_books_id`| string |          | Google Books author ID |
| `image_url`      | string |          | Author photo URL |

> **Name normalization:** Before storage, the server trims leading/trailing whitespace and collapses any internal whitespace run to a single space. Capitalization is preserved. For example, `"  J.R.R.  Tolkien  "` is stored as `"J.R.R. Tolkien"`. A name that is blank after normalization is rejected with `400`.

**Response:** `201 Created` with the author object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `409` | An author with that name already exists (comparison is case-insensitive) |
| `500` | Unexpected server error |

**Author object:**

```json
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
```

---

### `GET /api/authors/{id}` 🔒

Get a single author by ID.

**Response body (`200`):** Author object (same shape as the object in [`POST /api/authors`](#post-apiauthors)).

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Author not found |

---

### `PUT /api/authors/{id}` 🔒

Update an author (full update).

**Request body:** Same fields as `POST /api/authors`. The same name normalization (whitespace trimming and collapsing) applies.

**Response body (`200`):** Updated author object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `404` | Author not found |
| `409` | An author with that name already exists (comparison is case-insensitive) |

---

### `DELETE /api/authors/{id}` 🔒 **Admin**

Delete an author. Returns `204 No Content`.

> **Cascade:** Deleting an author also removes all `book_authors` join entries for that author. Books themselves are **not** deleted. See [Cascade Deletion Summary](../database-schema.md#cascade-deletion-summary).

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Author not found |

---

### `GET /api/authors/{id}/books` 🔒

List all books associated with an author, with pagination. Results are sorted by `title` ascending.

**Path parameters:** `{id}` — Author resource ID.

**Query parameters:**

| Parameter | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `limit`   | integer | `50`    | Maximum books to return (capped at `200`) |
| `offset`  | integer | `0`     | Number of books to skip |

**Response body (`200`):** Paginated books object (same envelope and book summary shape as [`GET /api/books`](books.md#get-apibooks)).

If no associated books are found (`total: 0`), Biblioteka first checks whether the author ID exists. If the author does not exist, the response is `404 Not Found`; otherwise, the response is `200 OK` with an empty `books` array and `total: 0`.

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Author not found when no books are associated with the given ID and the author does not exist |
| `500` | Unexpected server error |

---

