<!-- disable-agentic-editing: true -->

# API Reference — Libraries

[← Back to API Reference](../api-reference.md)

## Libraries

Libraries represent collections of books at one or more file-system paths. Creating a library automatically enqueues a background job to scan each configured path and index the books found there.

### `GET /api/libraries` 🔒

List all libraries.

**Response body (`200`):** Array of library objects (see below).

---

### `POST /api/libraries` 🔒 **Admin**

Create a library.

**Request body:**

| Field               | Type     | Required | Description |
|---------------------|----------|----------|-------------|
| `name`              | string   | ✓        | Display name (must be unique) |
| `paths`             | string[] | ✓        | Absolute file-system paths; each must be an existing directory |
| `organization_type` | string   |          | File-organization mode: `"book_per_folder"` (default, `Author/Title/file`), `"book_per_file"` (`Author/file`), or `"none"` (leave files in place). |
| `monitored`         | boolean  |          | Whether to auto-import new files |

**Responses:** `201 Created` with the new library object, `403 Forbidden` if the caller is not an admin, or `409 Conflict` if the name is taken.

> **Note:** On successful creation, the server asynchronously enqueues a `scan:library` job that then fans out a `scan:path` job for each configured path. Updating (`PUT`) a library does **not** trigger an automatic re-scan.

**Library object:**

```json
{
  "id": "<id>",
  "name": "Fiction",
  "paths": ["/mnt/books/fiction"],
  "organization_type": "book_per_folder",
  "monitored": true,
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

---

### `GET /api/libraries/{id}` 🔒

Get a single library by ID.

**Response body (`200`):** Library object (same shape as the object in [`POST /api/libraries`](#post-apilibraries)).

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Library not found |

---

### `PUT /api/libraries/{id}` 🔒 **Admin**

Update a library. All fields are replaced (full update).

**Request body:** Same as `POST /api/libraries`.

**Response body (`200`):** Updated library object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON or validation error such as missing name, invalid/non-existent/empty paths) |
| `403` | Caller is not an admin |
| `404` | Library not found |
| `409` | A library with that name already exists |

---

### `DELETE /api/libraries/{id}` 🔒 **Admin**

Delete a library. Returns `204 No Content`.

> **Cascade:** Deleting a library also removes all `library_books` join entries for that library. Books themselves are **not** deleted — only their membership in this library. See [Cascade Deletion Summary](../database-schema.md#cascade-deletion-summary).

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Library not found |

---

### `GET /api/libraries/{id}/books` 🔒

List books that belong to a specific library, with pagination. Results are sorted by `title` ascending.

**Query parameters:**

| Parameter | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `limit`   | integer | `50`    | Maximum books to return (capped at `200`) |
| `offset`  | integer | `0`     | Number of books to skip |

**Response body (`200`):** Paginated books object (same shape as [`GET /api/books`](books.md#get-apibooks)).

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Library not found |
| `500` | Unexpected server error |

---

