# Book Annotations

Annotations are notes you attach to books to record highlights, quotes, and observations. Each annotation stores free-form text, an optional EPUB position (CFI), and an optional group association for sharing within a [reading group](api/reading-groups.md).

## Key concepts

| Concept | Description |
|---------|-------------|
| **Text** | Required free-form note content. Cannot be blank. |
| **CFI** | Optional [Canonical Fragment Identifier](https://idpf.org/epub/linking/cfi/) — an EPUB position string that pinpoints a location within the file (e.g. `/4/2/4`). Use it to anchor a note to a specific passage. |
| **Group annotation** | An annotation associated with a reading group (`group_id` set). The list endpoint returns group annotations alongside your own so all members can see shared notes. Individual read/update/delete access remains restricted to the annotation owner. |
| **Visibility** | `GET /api/books/{id}/annotations` returns your own annotations plus any group annotations for groups you belong to. `GET /api/annotations/{id}` is owner-only. |

---

## Creating an annotation

```http
POST /api/books/{id}/annotations
Authorization: Bearer <token-or-api-key>
Content-Type: application/json

{
  "text": "This passage reframes the entire first act.",
  "cfi": "/4/2/4",
  "group_id": null
}
```

- `text` is required and must be non-blank.
- `cfi` and `group_id` are optional; omit or set to `null` to leave them unset.
- To share an annotation with a reading group, set `group_id` to the group's ID. You must be a member of that group.

**Response `201 Created`:**

```json
{
  "id": "4f8c2a1d9b6e4c3f8a7d1e2b5c6f9a0d",
  "user_id": "9a1b2c3d4e5f6780a1b2c3d4e5f67890",
  "book_id": "0f1e2d3c4b5a69788796a5b4c3d2e1f0",
  "text": "This passage reframes the entire first act.",
  "cfi": "/4/2/4",
  "user_name": "alice",
  "created_at": "2026-04-18T12:00:00Z",
  "updated_at": "2026-04-18T12:00:00Z"
}
```

---

## Listing annotations for a book

```http
GET /api/books/{id}/annotations
Authorization: Bearer <token-or-api-key>
```

Returns all annotations visible to you for the given book: your own annotations plus annotations shared with any reading group you belong to.

**Response `200 OK`:** Array of annotation objects (empty array if none).

> **Note:** This endpoint may return annotations you do not own (group-shared annotations from other members). You can identify the owner via the `user_id` and `user_name` fields. You cannot update or delete annotations you do not own.

---

## Retrieving a single annotation

```http
GET /api/annotations/{id}
Authorization: Bearer <token-or-api-key>
```

Returns the annotation if you own it. Returns `404 Not Found` for annotations owned by other users, even if they appear in the book's annotation list.

---

## Updating an annotation

```http
PUT /api/annotations/{id}
Authorization: Bearer <token-or-api-key>
Content-Type: application/json

{
  "text": "Updated note text.",
  "cfi": "/4/2/6",
  "group_id": "a1b2c3d4e5f678901234567890abcdef"
}
```

- You must own the annotation.
- `text` is required and must be non-blank.
- `cfi` and `group_id` are optional. **PUT is a full replacement**: omitting either field is equivalent to sending `null` and will clear it. Include the current values if you do not intend to clear them.
- To add or change a group association, set `group_id` to a group ID you belong to. To remove the group association, set `group_id` to `null`.

**Response `200 OK`:** The updated annotation object.

---

## Deleting an annotation

```http
DELETE /api/annotations/{id}
Authorization: Bearer <token-or-api-key>
```

Permanently removes the annotation. You must own it.

**Response `204 No Content`.**

---

## API reference

All annotation endpoints require authentication (`Authorization: Bearer <token-or-api-key>`). See [Authentication](api.md#authentication) for accepted credential types (JWT, API key, or session cookie).

### `GET /api/books/{id}/annotations` 🔒

Lists annotations visible to the authenticated user for a book (own + group-shared).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if no annotations) |
| `401 Unauthorized` | Missing or invalid authentication |
| `500 Internal Server Error` | Database error |

---

### `POST /api/books/{id}/annotations` 🔒

Creates a new annotation on a book.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | ✓ | Annotation content (must be non-blank) |
| `cfi` | string \| null | — | EPUB CFI position string |
| `group_id` | string \| null | — | Reading group ID to share with (must be a group you belong to) |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `201 Created` | Annotation created |
| `400 Bad Request` | Blank `text` or invalid JSON |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | `group_id` refers to a group you are not a member of |
| `500 Internal Server Error` | Database error |

---

### `GET /api/annotations/{id}` 🔒

Returns a single annotation owned by the authenticated user.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

### `PUT /api/annotations/{id}` 🔒

Updates an annotation owned by the authenticated user.

**Request body:** Same fields as `POST /api/books/{id}/annotations`.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Blank `text` or invalid JSON |
| `401 Unauthorized` | Missing or invalid authentication |
| `403 Forbidden` | `group_id` refers to a group you are not a member of |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/annotations/{id}` 🔒

Deletes an annotation owned by the authenticated user.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Deleted |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Annotation not found or not owned by you |
| `500 Internal Server Error` | Database error |

---

## Annotation object fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique annotation ID |
| `user_id` | string | ID of the user who created the annotation |
| `book_id` | string | ID of the annotated book |
| `text` | string | Annotation content |
| `cfi` | string (optional) | EPUB CFI position. Omitted from responses when not set. |
| `group_id` | string (optional) | Reading group the annotation is shared with. Omitted from responses when not set. |
| `user_name` | string | Display name of the annotation author |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last-updated timestamp |
