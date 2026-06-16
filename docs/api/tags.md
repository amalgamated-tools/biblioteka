<!-- disable-agentic-editing: true -->

# API Reference — Tags

[← Back to API Reference](../api-reference.md)

## Tags

### `GET /api/tags` 🔒

List all tags. Results are sorted by `name` ascending.

**Response body (`200`):** Array of tag objects (see below).

---

### `POST /api/tags` 🔒

Create a tag.

**Request body:**

| Field  | Type   | Required | Description |
|--------|--------|----------|-------------|
| `name` | string | ✓        | Tag name (must be unique, case-insensitive) |

> **Name normalization:** Before storage, the server trims leading/trailing whitespace and collapses any internal whitespace run to a single space. Capitalization is preserved. For example, `"  science  fiction  "` is stored as `"science fiction"`. A name that is blank after normalization is rejected with `400`.

**Response:** `201 Created` with the tag object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `409` | A tag with that name already exists (comparison is case-insensitive) |
| `500` | Unexpected server error |

**Tag object:**

```json
{
  "id": "<id>",
  "name": "science fiction",
  "created_at": "2026-03-14T02:00:00Z",
  "updated_at": "2026-03-14T02:00:00Z"
}
```

---

### `GET /api/tags/{id}` 🔒

Get a single tag by ID.

**Response body (`200`):** Tag object (same shape as the object in [`POST /api/tags`](#post-apitags)).

**Errors:**

| Status | Meaning |
|--------|---------|
| `404` | Tag not found |

---

### `PUT /api/tags/{id}` 🔒

Update a tag (full update).

**Request body:** Same fields as `POST /api/tags`. The same name normalization (whitespace trimming and collapsing) applies.

**Response body (`200`):** Updated tag object.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (malformed JSON, missing name, or name is blank after normalization) |
| `404` | Tag not found |
| `409` | A tag with that name already exists (comparison is case-insensitive) |

---

### `DELETE /api/tags/{id}` 🔒 **Admin**

Delete a tag. Returns `204 No Content`.

> **Cascade:** Deleting a tag also removes all `book_tags` join entries for that tag via the `book_tags.tag_id` foreign key's `ON DELETE CASCADE` constraint. Books themselves are **not** deleted.

**Errors:**

| Status | Meaning |
|--------|---------|
| `403` | Caller is not an admin |
| `404` | Tag not found |

---

