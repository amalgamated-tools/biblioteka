# Reading Groups

Reading groups let you collaborate with other Biblioteka users around shared reading lists and compare per-member reading progress. Create a group, invite your book club or reading circle, share lists for everyone to browse, and track how far each member has gotten through a book.

## Key concepts

| Concept | Description |
|---------|-------------|
| **Owner** | The user who created the group. Owners can rename the group, add and remove members, and delete the group. |
| **Member** | A user who has been added to the group by the owner. Members can view the group, browse shared lists, see reading progress, share their own lists, and leave the group. |
| **Shared list** | A reading list that a member has explicitly shared with the group. Only the list's owner can share or unshare it. |
| **Member visibility** | A user can only see or interact with a group they belong to. Non-members receive `404 Not Found` rather than `403 Forbidden` to avoid leaking group existence. |

---

## Creating a group

1. **Via the web UI** — open **Reading Groups** from the main navigation and click **New Group**.
2. **Via the API** — send a `POST /api/groups` request (see [API reference](api-reference.md#reading-groups)):

```http
POST /api/groups
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "name": "Tuesday Night Book Club",
  "description": "We meet every Tuesday at 7 pm."
}
```

The authenticated user becomes the group owner and is automatically added as a member. Group names are case-insensitively unique per owner.

---

## Managing members

### Adding a member

Only the group owner can add members. The target user must already have a Biblioteka account.

```http
POST /api/groups/{id}/members
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "user_id": "<target-user-id>"
}
```

Adding a user who is already a member is idempotent — it returns `204 No Content` without error.

### Removing a member

- The **owner** can remove any member.
- A **member** can remove themselves (leave the group).
- The **owner cannot leave their own group**. To disband a group entirely, use `DELETE /api/groups/{id}`.

```http
DELETE /api/groups/{id}/members/{memberID}
Authorization: Bearer <jwt>
```

---

## Sharing reading lists

Any member can share one of their personal reading lists with the group, making it visible to all members.

```http
POST /api/groups/{id}/lists
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "list_id": "<reading-list-id>"
}
```

**Requirements:**

- You must own the reading list (lists are scoped per user).
- You must be a member of the group.

Sharing a list that is already shared with the group is idempotent.

### Retrieving shared lists

```http
GET /api/groups/{id}/lists
Authorization: Bearer <jwt>
```

Returns all reading lists shared with the group. The authenticated user must be a group member.

### Unsharing a list

Only the list owner can remove a list from the group.

```http
DELETE /api/groups/{id}/lists/{listID}
Authorization: Bearer <jwt>
```

> Unsharing a list does not delete it; it only removes it from the group.

---

## Comparing reading progress

Use the progress endpoint to see how far each group member has read a specific book. Progress values are sourced from each member's **Kobo reading data** (recorded by a synced Kobo device or the KOReader Kobo plugin).

```http
GET /api/groups/{id}/progress?book_id=<book-id>
Authorization: Bearer <jwt>
```

**Response `200 OK`:**

```json
[
  {
    "user_id": "abc123...",
    "user_name": "Alice",
    "percentage": 0.72,
    "updated_at": "2026-04-10T19:45:00Z"
  },
  {
    "user_id": "def456...",
    "user_name": "Bob",
    "percentage": 0,
    "updated_at": null
  }
]
```

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | string | Member's user ID |
| `user_name` | string | Member's display name |
| `percentage` | number | Reading progress in `[0, 1]` (0 = not started or no data) |
| `updated_at` | string? | ISO 8601 timestamp of the last sync, or `null` if no data |

All group members appear in the response. Members with no recorded Kobo reading progress have `percentage: 0` and `updated_at: null`.

> **Kobo / KOReader requirement:** Progress is only recorded for books synced via a Kobo device or KOReader. See the [Kobo Sync guide](kobo.md) and [KOReader Sync guide](koreader.md) for setup instructions.

---

## Updating a group

Only the group owner can rename or update the description.

```http
PUT /api/groups/{id}
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "name": "Wednesday Night Book Club",
  "description": "Rescheduled to Wednesdays."
}
```

Setting `description` to `null` or omitting it clears the description field.

---

## Deleting a group

Only the owner can delete a group. Deleting a group removes all memberships and shared-list associations; the underlying reading lists and books are not affected.

```http
DELETE /api/groups/{id}
Authorization: Bearer <jwt>
```

Returns `204 No Content`.

---

## API summary

| Method | Path | Who can call | Description |
|--------|------|-------------|-------------|
| `GET` | `/api/groups` | Any member | List all groups the user belongs to |
| `POST` | `/api/groups` | Any authenticated user | Create a new group |
| `GET` | `/api/groups/{id}` | Members | Get group details |
| `PUT` | `/api/groups/{id}` | Owner | Update name / description |
| `DELETE` | `/api/groups/{id}` | Owner | Delete the group |
| `GET` | `/api/groups/{id}/members` | Members | List members |
| `POST` | `/api/groups/{id}/members` | Owner | Add a member |
| `DELETE` | `/api/groups/{id}/members/{memberID}` | Owner or self | Remove a member / leave |
| `GET` | `/api/groups/{id}/lists` | Members | List shared reading lists |
| `POST` | `/api/groups/{id}/lists` | Members (list owner) | Share a reading list |
| `DELETE` | `/api/groups/{id}/lists/{listID}` | Members (list owner) | Unshare a reading list |
| `GET` | `/api/groups/{id}/progress?book_id=` | Members | Compare per-member reading progress |

For request/response schemas and full error code tables, see the [API Reference — Reading Groups](api-reference.md#reading-groups).

---

## Error responses

All group endpoints return JSON errors in the standard format:

```json
{ "error": "human-readable message" }
```

Common status codes for group endpoints:

| Code | Condition |
|------|-----------|
| `400 Bad Request` | Missing or blank name, missing required field, or owner trying to leave their group |
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | Group not found, user is not a member, or lacks the required role |
| `409 Conflict` | You already own a group with that name |
| `500 Internal Server Error` | Unexpected server-side failure |
