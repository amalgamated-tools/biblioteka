<!-- disable-agentic-editing: true -->

# API Reference — Reading Lists

[← Back to API Reference](../api-reference.md)

## Reading Lists

Reading lists let each user curate named, ordered collections of books (shelves). All reading list endpoints require authentication. Lists are scoped to the authenticated user — users never see each other's lists.

### `GET /api/reading-lists` 🔒

Returns all reading lists owned by the authenticated user, ordered by name.

**Response:** `200 OK`

```json
[
  {
    "id": "a1b2c3d4e5f6...",
    "name": "To Read",
    "description": "Books I want to read this year",
    "book_count": 12,
    "created_at": "2026-04-01T10:00:00Z",
    "updated_at": "2026-04-10T14:30:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique reading list ID (opaque string; do not rely on format) |
| `name` | string | Normalized list name |
| `description` | string \| null | Optional free-text description |
| `book_count` | integer | Number of books currently in the list |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last-updated timestamp |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if no lists exist) |
| `401 Unauthorized` | Missing or invalid authentication |
| `500 Internal Server Error` | Database error |

---

### `POST /api/reading-lists` 🔒

Creates a new reading list for the authenticated user.

**Request body:**

```json
{
  "name": "To Read",
  "description": "Optional description"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | List name (normalized; must be unique per user) |
| `description` | string \| null | | Optional description |

**Response:** `201 Created` — the created reading list object (same shape as the list items above).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `201 Created` | List created successfully |
| `400 Bad Request` | Missing or blank `name` |
| `401 Unauthorized` | Missing or invalid authentication |
| `409 Conflict` | A reading list with that name already exists for this user |
| `500 Internal Server Error` | Database error |

---

### `GET /api/reading-lists/{id}` 🔒

Returns a single reading list by ID. The list must be owned by the authenticated user.

**Response:** `200 OK` — the reading list object.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Missing or empty list ID path segment |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | List not found or not owned by the user |
| `500 Internal Server Error` | Database error |

---

### `PUT /api/reading-lists/{id}` 🔒

Updates a reading list. The `name` field is required; omitting `description` (or setting it to `null`) clears the field.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | New list name (must not be blank) |
| `description` | string \| null | | New description; omitted or `null` clears the field |

```json
{
  "name": "Currently Reading",
  "description": "Updated description"
}
```

**Response:** `200 OK` — the updated reading list object.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Updated successfully |
| `400 Bad Request` | Invalid or empty reading list ID, or missing/blank `name` |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | List not found or not owned by the user |
| `409 Conflict` | Another list with that name already exists for this user |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/reading-lists/{id}` 🔒

Deletes a reading list and all its book associations. Returns `204 No Content`.

> Deleting a list does not delete the underlying books.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Deleted successfully |
| `400 Bad Request` | Missing or empty list ID path segment |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | List not found or not owned by the user |
| `500 Internal Server Error` | Database error |

---

### `GET /api/reading-lists/{id}/books` 🔒

Returns a paginated list of books in a reading list, ordered by position then add time.

**Query parameters:**

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `limit` | integer | `50` | `200` | Maximum number of books to return |
| `offset` | integer | `0` | — | Zero-based offset for pagination |

**Response:** `200 OK`

```json
{
  "books": [ /* array of book summary objects */ ],
  "total": 42,
  "limit": 50,
  "offset": 0
}
```

See [Books](books.md) for the book summary object shape. The `total` field reflects the total number of books in the list (not the current page size).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Invalid or empty reading list ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | List not found or not owned by the user |
| `500 Internal Server Error` | Database error |

---

### `POST /api/reading-lists/{id}/books` 🔒

Adds a book to a reading list. The operation is idempotent — adding a book that is already in the list succeeds without error. Returns `204 No Content`.

**Request body:**

```json
{ "book_id": "the-book-id" }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `book_id` | string | ✓ | ID of the book to add |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Book added (or already present) |
| `400 Bad Request` | Invalid or empty reading list ID, or missing `book_id` |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Reading list or book not found |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/reading-lists/{id}/books/{bookId}` 🔒

Removes a book from a reading list. The operation is idempotent — removing a book that is not in the list succeeds without error. Returns `204 No Content`.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Book removed (or was not present) |
| `400 Bad Request` | Invalid or empty reading list ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Reading list not found or not owned by the user |
| `500 Internal Server Error` | Database error |

---

### `GET /api/books/{id}/reading-lists` 🔒

Returns all reading lists owned by the authenticated user that contain the specified book, ordered by name. Useful for displaying which lists a book belongs to on the book detail page.

**Response:** `200 OK` — an array of reading list objects (same shape as `GET /api/reading-lists`).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if the book is in no lists) |
| `401 Unauthorized` | Missing or invalid authentication |
| `500 Internal Server Error` | Database error |

---

