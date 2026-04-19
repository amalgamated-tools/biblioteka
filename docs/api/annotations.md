<!-- disable-agentic-editing: true -->

# API Reference — Annotations

[← Back to API Reference](../api-reference.md)

## Annotations

Book annotations are notes you attach to books to record highlights, quotes, and observations. Each annotation stores free-form text, an optional [EPUB CFI](https://idpf.org/epub/linking/cfi/) position, and an optional reading-group association for sharing with group members.

All annotation endpoints require authentication (for example, `Authorization: Bearer <token-or-api-key>`; browser session cookie authentication also works where relevant).

> **Visibility asymmetry:** `GET /api/books/{id}/annotations` returns your own annotations *plus* any group-shared annotations for groups you belong to. `GET /api/annotations/{id}` is owner-only — it returns `404 Not Found` for annotations you do not own, even if they appear in the book list. This is intentional: the list shows shared context; individual access is restricted to the annotation owner.

---

### `GET /api/books/{id}/annotations` 🔒

Lists all annotations visible to the authenticated user for a given book: the user's own annotations plus annotations shared with any reading group the user belongs to.

**Response body (`200 OK`):** Array of annotation objects (empty array if none).

```json
[
  {
    "id": "01abc123...",
    "user_id": "01def456...",
    "book_id": "01ghi789...",
    "text": "This passage reframes the entire first act.",
    "cfi": "/4/2/4",
    "group_id": "01grp123...",
    "user_name": "alice",
    "created_at": "2026-04-18T12:00:00Z",
    "updated_at": "2026-04-18T12:00:00Z"
  }
]
```

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if no annotations are visible) |
| `401 Unauthorized` | Missing or invalid authentication |
| `500 Internal Server Error` | Database error |

---

### `POST /api/books/{id}/annotations` 🔒

Creates a new annotation on a book.

**Request body:**

```json
{
  "text": "This passage reframes the entire first act.",
  "cfi": "/4/2/4",
  "group_id": null
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | ✓ | Annotation content (must be non-blank) |
| `cfi` | string \| null | | EPUB Canonical Fragment Identifier — an EPUB position string (e.g. `/4/2/4`). Omit or set to `null` to leave unset. |
| `group_id` | string \| null | | Reading group ID to share the annotation with. You must be a member of the group. Omit or set to `null` for a private annotation. |

**Response body (`201 Created`):** The created annotation object (same shape as the list items above).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `201 Created` | Annotation created successfully |
| `400 Bad Request` | `text` is blank or the request body is invalid JSON |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | `group_id` refers to a group you are not a member of |
| `500 Internal Server Error` | Database error |

---

### `GET /api/annotations/{id}` 🔒

Returns a single annotation. You must own the annotation — annotations belonging to other users return `404 Not Found` even if they appeared in the book list.

**Response body (`200 OK`):** Annotation object (same shape as the list items above).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Missing or invalid annotation ID in the path |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

### `PUT /api/annotations/{id}` 🔒

Updates an annotation you own. This is a **full replacement**: every writable field is overwritten. Omitting an optional field (`cfi`, `group_id`) is equivalent to sending `null` and will clear that field. Include the current values if you do not intend to clear them.

**Request body:**

```json
{
  "text": "Updated note text.",
  "cfi": "/4/2/6",
  "group_id": "01grp123..."
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | ✓ | Updated annotation content (must be non-blank) |
| `cfi` | string \| null | | New CFI position, or `null` to clear it |
| `group_id` | string \| null | | New group association, or `null` to remove it. You must be a member of the specified group. |

**Response body (`200 OK`):** The updated annotation object.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Updated successfully |
| `400 Bad Request` | `text` is blank or the request body is invalid JSON |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | `group_id` refers to a group you are not a member of |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/annotations/{id}` 🔒

Permanently removes an annotation you own.

**Response:** `204 No Content`.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Deleted successfully |
| `400 Bad Request` | Missing or invalid annotation ID in the path |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

## Annotation object fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique annotation ID (opaque string) |
| `user_id` | string | ID of the user who created the annotation |
| `book_id` | string | ID of the annotated book |
| `text` | string | Annotation content |
| `cfi` | string | EPUB CFI position. Omitted from responses when not set (`omitempty`). |
| `group_id` | string | Reading group the annotation is shared with. Omitted from responses when not set (`omitempty`). |
| `user_name` | string | Display name of the annotation author (useful for identifying group-shared annotations) |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last-updated timestamp |
