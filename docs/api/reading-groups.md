<!-- disable-agentic-editing: true -->

# API Reference — Reading Groups

[← Back to API Reference](../api-reference.md)

## Reading Groups

Reading groups let users collaborate around shared reading lists and compare reading progress. Each group has one **owner** (the user who created it) and any number of **members**. All reading group endpoints require authentication.

> **Member visibility:** A user can only see or interact with a group they belong to. Non-members receive `404 Not Found` rather than `403 Forbidden` to avoid leaking group existence.

---

### `GET /api/groups` 🔒

Returns all reading groups the authenticated user belongs to (as owner or member), ordered by name.

**Response:** `200 OK`

```json
[
  {
    "id": "g1h2i3j4k5l6...",
    "owner_id": "u1a2b3c4d5e6...",
    "name": "Sci-Fi Book Club",
    "description": "Monthly science fiction reads",
    "member_count": 5,
    "created_at": "2026-01-10T09:00:00Z",
    "updated_at": "2026-03-15T12:00:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique group ID (opaque string) |
| `owner_id` | string | User ID of the group owner |
| `name` | string | Group name |
| `description` | string \| null | Optional free-text description |
| `member_count` | integer | Current number of members (including the owner) |
| `created_at` | string | ISO 8601 creation timestamp |
| `updated_at` | string | ISO 8601 last-updated timestamp |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if the user belongs to no groups) |
| `401 Unauthorized` | Missing or invalid authentication |
| `500 Internal Server Error` | Database error |

---

### `POST /api/groups` 🔒

Creates a new reading group. The authenticated user becomes the owner and is automatically added as a member.

**Request body:**

```json
{
  "name": "Sci-Fi Book Club",
  "description": "Monthly science fiction reads"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Group name (must not be blank) |
| `description` | string \| null | | Optional description |

**Response:** `201 Created` — the created group object (same shape as the list items above).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `201 Created` | Group created successfully |
| `400 Bad Request` | Missing or blank `name` |
| `401 Unauthorized` | Missing or invalid authentication |
| `409 Conflict` | You already own a group with that name |
| `500 Internal Server Error` | Database error |

---

### `GET /api/groups/{id}` 🔒

Returns a single reading group by ID. The authenticated user must be a member of the group.

**Response:** `200 OK` — the group object (same shape as the list items above).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Missing or empty group ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not a member |
| `500 Internal Server Error` | Database error |

---

### `PUT /api/groups/{id}` 🔒

Updates a group's name and description. Only the group owner may call this endpoint. Omitting `description` (or setting it to `null`) clears the field.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | New group name (must not be blank) |
| `description` | string \| null | | New description; omitted or `null` clears the field |

**Response:** `200 OK` — the updated group object.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Updated successfully |
| `400 Bad Request` | Missing or blank `name` |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not the owner |
| `409 Conflict` | You already own a group with that name |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/groups/{id}` 🔒

Deletes a reading group and all its memberships and shared list associations. Only the group owner may call this endpoint. Returns `204 No Content`.

> Deleting a group does not delete the underlying reading lists or books.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Deleted successfully |
| `400 Bad Request` | Missing or empty group ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not the owner |
| `500 Internal Server Error` | Database error |

---

### `GET /api/groups/{id}/members` 🔒

Returns all members of a reading group. The authenticated user must be a member of the group.

**Response:** `200 OK`

```json
[
  {
    "group_id": "g1h2i3j4k5l6...",
    "user_id": "u1a2b3c4d5e6...",
    "user_name": "Alice",
    "role": "owner",
    "joined_at": "2026-01-10T09:00:00Z"
  },
  {
    "group_id": "g1h2i3j4k5l6...",
    "user_id": "u7m8n9o0p1q2...",
    "user_name": "Bob",
    "role": "member",
    "joined_at": "2026-02-01T08:30:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `group_id` | string | Group ID |
| `user_id` | string | Member's user ID |
| `user_name` | string | Member's display name |
| `role` | `"owner"` \| `"member"` | Role within the group |
| `joined_at` | string | ISO 8601 timestamp when the user joined |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success |
| `400 Bad Request` | Missing or empty group ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not a member |
| `500 Internal Server Error` | Database error |

---

### `POST /api/groups/{id}/members` 🔒

Adds a user to a reading group. Only the group owner may call this endpoint. The operation is idempotent — adding a user who is already a member succeeds without error. Returns `204 No Content`.

**Request body:**

```json
{ "user_id": "the-user-id" }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `user_id` | string | ✓ | ID of the user to add |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Member added (or was already a member) |
| `400 Bad Request` | Missing `user_id` |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found, user is not the owner, or target user does not exist |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/groups/{id}/members/{memberID}` 🔒

Removes a member from a reading group. The group owner may remove any member; regular members may only remove themselves (leave the group). Returns `204 No Content`.

> The owner cannot leave their own group. To disband a group, use `DELETE /api/groups/{id}` instead.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | Member removed successfully |
| `400 Bad Request` | Missing ID path segment, or owner attempted to remove themselves |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or the requester lacks permission |
| `500 Internal Server Error` | Database error |

---

### `GET /api/groups/{id}/lists` 🔒

Returns all reading lists shared with a reading group. The authenticated user must be a member of the group.

**Response:** `200 OK` — an array of reading list objects (same shape as `GET /api/reading-lists`).

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (empty array if no lists are shared) |
| `400 Bad Request` | Missing or empty group ID |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not a member |
| `500 Internal Server Error` | Database error |

---

### `POST /api/groups/{id}/lists` 🔒

Shares a reading list with a group. The authenticated user must own the reading list **and** be a member of the group. The operation is idempotent — sharing a list that is already shared with the group succeeds without error. Returns `204 No Content`.

**Request body:**

```json
{ "list_id": "the-reading-list-id" }
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `list_id` | string | ✓ | ID of the reading list to share |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | List shared successfully |
| `400 Bad Request` | Missing `list_id` |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group or reading list not found, or user does not own the list / is not a group member |
| `500 Internal Server Error` | Database error |

---

### `DELETE /api/groups/{id}/lists/{listID}` 🔒

Removes a reading list from a group. The authenticated user must own the reading list. Returns `204 No Content`.

**Status codes:**

| Status | Meaning |
|--------|---------|
| `204 No Content` | List unshared successfully |
| `400 Bad Request` | Missing ID path segment |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group or reading list not found, or user does not own the list |
| `500 Internal Server Error` | Database error |

---

### `GET /api/groups/{id}/progress` 🔒

Returns each group member's reading progress for a specific book. The authenticated user must be a member of the group. Progress values come from the member's Kobo reading data.

**Query parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `book_id` | string | ✓ | ID of the book to query progress for |

**Response:** `200 OK`

```json
[
  {
    "user_id": "u1a2b3c4d5e6...",
    "user_name": "Alice",
    "percentage": 0.72,
    "updated_at": "2026-04-15T20:30:00Z"
  },
  {
    "user_id": "u7m8n9o0p1q2...",
    "user_name": "Bob",
    "percentage": 0.10,
    "updated_at": "2026-04-10T18:00:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | string | Member's user ID |
| `user_name` | string | Member's display name |
| `percentage` | number | Reading progress as a decimal between `0` and `1` (e.g. `0.72` = 72%) |
| `updated_at` | string \| null | ISO 8601 timestamp of the last progress sync; `null` if the member has no recorded progress for this book |

**Status codes:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Success (all group members are returned; members without recorded progress have `percentage: 0` and `updated_at: null`) |
| `400 Bad Request` | Missing or empty group ID, or missing `book_id` query parameter |
| `401 Unauthorized` | Missing or invalid authentication |
| `404 Not Found` | Group not found or user is not a member |
| `500 Internal Server Error` | Database error |

---

