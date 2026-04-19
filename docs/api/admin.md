<!-- disable-agentic-editing: true -->

# API Reference — Admin

[← Back to API Reference](../api-reference.md)

## Admin

### `GET /api/admin/users` 🔒 **Admin** · **JWT only**

List all registered users.

**Response body (`200`):** Array of user objects:

```json
[
  {
    "id": "<id>",
    "name": "Alice",
    "email": "alice@example.com",
    "is_admin": true,
    "oidc_linked": false,
    "created_at": "2026-03-14T02:00:00Z"
  }
]
```

| Field         | Type    | Description |
|---------------|---------|-------------|
| `id`          | string  | Opaque user ID |
| `name`        | string  | User's display name |
| `email`       | string  | User's email address |
| `is_admin`    | boolean | `true` when the user has admin privileges |
| `oidc_linked` | boolean | `true` when the account has an OIDC/SSO identity linked |
| `created_at`  | string  | Timestamp when the account was created (ISO 8601) |

---

### `PUT /api/admin/users/{id}` 🔒 **Admin** · **JWT only**

Grant or revoke admin privileges for a user. Admins cannot change their own admin status.

**Request body:**

| Field      | Type    | Required |
|------------|---------|----------|
| `is_admin` | boolean | ✓        |

**Response body (`200`):**

```json
{ "message": "admin status updated" }
```

**Error responses:**

| Status | Description |
|--------|-------------|
| `400 Bad Request` | Request body is invalid or missing required fields |
| `400 Bad Request` | Caller attempted to change their own admin status |
| `401 Unauthorized` | JWT is missing, malformed, or invalid, or the JWT is valid but the calling user's account no longer exists |
| `403 Forbidden` | Caller is not an admin |
| `404 Not Found` | Target user not found |
| `405 Method Not Allowed` | Method is not `PUT` |
| `500 Internal Server Error` | Database error or failed to update the user's admin status |

---

### `GET /api/audit-logs` 🔒 **Admin**

Return a paginated list of all audit log entries. Each entry records an action performed on an entity (e.g. book created, library deleted).

**Query parameters:**

| Parameter | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `limit`   | integer | `50`    | Maximum entries to return (capped at `200`) |
| `offset`  | integer | `0`     | Number of entries to skip (for pagination) |

**Response body (`200`):**

```json
{
  "entries": [
    {
      "id": "<id>",
      "user_id": "<user-id>",
      "action": "book.created",
      "entity_type": "book",
      "entity_id": "<book-id>",
      "metadata": { "title": "Dune" },
      "created_at": "2026-03-14T02:00:00Z"
    }
  ],
  "total": 142,
  "limit": 50,
  "offset": 0
}
```

| Field     | Type    | Description |
|-----------|---------|-------------|
| `entries` | array   | Page of audit log entries |
| `total`   | integer | Total number of entries (for pagination) |
| `limit`   | integer | Effective limit used |
| `offset`  | integer | Effective offset used |

**Audit log entry object** (`entries[]` elements):

| Field         | Type    | Description |
|---------------|---------|-------------|
| `id`          | string  | ID of this audit log entry |
| `user_id`     | string? | ID of the user who performed the action; `null` for system actions |
| `action`      | string  | Typically a dot-separated `<entity_type>.<verb>` string, but the prefix is not guaranteed to match `entity_type` (see table below for all known values) |
| `entity_type` | string  | Type of the entity affected (e.g. `book`, `library`) |
| `entity_id`   | string  | ID of the entity affected (e.g. book ID) |
| `metadata`    | object? | Optional extra context (e.g. entity name at time of action); omitted when empty |
| `created_at`  | string  | Timestamp when the action was recorded (ISO 8601) |

**Known `action` values:**

| `action`               | `entity_type` | Trigger |
|------------------------|---------------|---------|
| `library.created`      | `library`     | Library created via `POST /api/libraries` |
| `library.updated`      | `library`     | Library updated via `PUT /api/libraries/{id}` |
| `library.deleted`      | `library`     | Library deleted via `DELETE /api/libraries/{id}` |
| `book.created`         | `book`        | Book created via `POST /api/books` |
| `book.updated`         | `book`        | Book updated via `PUT /api/books/{id}` |
| `book.deleted`         | `book`        | Book deleted via `DELETE /api/books/{id}` |
| `book.uploaded`        | `book_upload` | Book file uploaded via `POST /api/books/upload` (`entity_type` is `book_upload`; do not infer it from the `action` prefix) |
| `author.created`       | `author`      | Author created via `POST /api/authors` |
| `author.updated`       | `author`      | Author updated via `PUT /api/authors/{id}` |
| `author.deleted`       | `author`      | Author deleted via `DELETE /api/authors/{id}` |
| `series.created`       | `series`      | Series created via `POST /api/series` |
| `series.updated`       | `series`      | Series updated via `PUT /api/series/{id}` |
| `series.deleted`       | `series`      | Series deleted via `DELETE /api/series/{id}` |
| `book_file.created`    | `book_file`   | File attached via `POST /api/books/{id}/files` |
| `book_file.deleted`    | `book_file`   | File deleted via `DELETE /api/book-files/{id}` |
| `book_file.emailed`    | `book_file`   | File emailed via `POST /api/book-files/{id}/email` |
| `api_key.created`      | `api_key`     | API key created via `POST /api/api-keys` |
| `api_key.deleted`      | `api_key`     | API key revoked via `DELETE /api/api-keys/{id}` |
| `opds_credential.updated` | `opds_credential` | OPDS credentials set via `PUT /api/opds/credentials` |
| `opds_credential.deleted` | `opds_credential` | OPDS credentials removed via `DELETE /api/opds/credentials` |
| `kobo_token.created`   | `kobo_token`  | Kobo sync token created via `POST /api/kobo/tokens` |
| `kobo_token.deleted`   | `kobo_token`  | Kobo sync token revoked via `DELETE /api/kobo/tokens/{id}` |
| `kosync_credential.updated` | `kosync_credential` | KOSync credentials set or updated via `PUT /api/kosync/credentials` |
| `kosync_credential.deleted` | `kosync_credential` | KOSync credentials removed via `DELETE /api/kosync/credentials` |
| `user.signed_up`       | `user`        | New account created via `POST /api/auth/signup` |
| `user.password_changed` | `user`       | Password changed via `PUT /api/auth/password` |
| `user.admin_updated`   | `user`        | Admin status changed via `PUT /api/admin/users/{id}` |
| `user.profile_updated` | `user`        | Display name changed via `PUT /api/auth/me` |
| `smtp.config_updated`  | `config`      | SMTP settings saved via `PUT /api/config/smtp` |
| `watch_folder.config_updated` | `config` | Watch folder settings saved via `PUT /api/config/watch-folder` |
| `metadata.fetch_requested` | `book`   | Metadata enrichment job enqueued via `POST /api/books/{id}/metadata/fetch` |
| `metadata.applied`     | `book`        | Pending metadata applied to a book via `POST /api/books/{id}/metadata/apply` |
| `metadata.rejected`    | `book`        | Pending metadata discarded via `POST /api/books/{id}/metadata/reject` |
| `ai_enrichment.fetch_requested` | `book` | AI enrichment job enqueued via `POST /api/books/{id}/metadata/ai-fetch` |
| `ai_enrichment.applied` | `book`       | Pending AI enrichment applied via `POST /api/books/{id}/metadata/ai-apply` |
| `ai_enrichment.rejected` | `book`      | Pending AI enrichment discarded via `POST /api/books/{id}/metadata/ai-reject` |
| `llm.config_updated`   | `config`      | LLM settings saved via `PUT /api/config/llm` |

| Status | Description |
|--------|-------------|
| `200 OK` | Paginated audit log entries |
| `400 Bad Request` | Invalid `limit` or `offset` value |
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Caller is not an admin |
| `405 Method Not Allowed` | Non-GET request |
| `500 Internal Server Error` | Database error |

---

