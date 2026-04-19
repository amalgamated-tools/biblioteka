<!-- disable-agentic-editing: true -->

# API Reference — Kobo & KOSync

[← Back to API Reference](../api-reference.md)

## Kobo Tokens

Kobo sync tokens authenticate a Kobo e-reader device against the built-in Kobo device API served under `/kobo/<token>/`. Each token is scoped to a single user; multiple tokens can exist per user (one per device is recommended). See the [Kobo Sync guide](../kobo.md) for setup instructions and a full feature overview.

All three endpoints require a **JWT** — API keys are not accepted (see [JWT-only endpoints](../api-reference.md#jwt-only-endpoints)).

### `GET /api/kobo/tokens` 🔒 **JWT only**

List all Kobo sync tokens for the authenticated user.

**Response `200 OK`:**

```json
[
  {
    "id": "<id>",
    "user_id": "<user_id>",
    "name": "Kobo Libra 2",
    "created_at": "2026-03-17T12:00:00Z"
  }
]
```

Returns `[]` when no tokens exist.

**Kobo token object** fields:

| Field        | Type   | Description |
|--------------|--------|-------------|
| `id`         | string | Opaque token ID (used for deletion) |
| `user_id`    | string | ID of the owning user |
| `name`       | string | Human-readable label given at creation time |
| `created_at` | string | Timestamp when the token was created (ISO 8601) |

---

### `POST /api/kobo/tokens` 🔒 **JWT only**

Create a new Kobo sync token. The raw token is returned **only in this response** and is never retrievable again.

**Request body:**

| Field  | Type   | Required | Description                          |
|--------|--------|----------|--------------------------------------|
| `name` | string | ✓        | Human-readable label (max 100 chars) |

**Response `201 Created`:**

```json
{
  "id": "<id>",
  "user_id": "<user_id>",
  "name": "Kobo Libra 2",
  "created_at": "2026-03-17T12:00:00Z",
  "token": "a3f8e1b2c4d5..."
}
```

The `token` field is the raw value. Build the device sync URL as `https://<host>/kobo/<token>/`. The response also sets `Cache-Control: no-store` to prevent proxy or browser caching of the token.

**Error responses:**

| Status | Description |
|--------|-------------|
| `400 Bad Request` | `name` missing, empty, or exceeds 100 characters |
| `401 Unauthorized` | Missing or invalid JWT |

---

### `DELETE /api/kobo/tokens/{id}` 🔒 **JWT only**

Delete a Kobo sync token. The device using this token will receive `401` on its next sync.

**Path parameters:** `{id}` — Kobo token resource ID.

**Response:** `204 No Content`

**Error responses:**

| Status | Description |
|--------|-------------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | Token not found or not owned by the authenticated user |

---

## KOReader / KOSync

Biblioteka implements the [kosync](https://github.com/koreader/koreader-sync-server) protocol so that [KOReader](https://koreader.rocks/) can back up and synchronise reading positions to your self-hosted server. See the [KOReader Sync guide](../koreader.md) for setup instructions.

### Credential management (JWT-protected)

These endpoints require a **JWT** (not an API key) and manage the separate KOSync username and password used by KOReader.

#### `GET /api/kosync/credentials` 🔒 **JWT only**

Returns the current user's KOSync credentials (username and timestamps; the password hash is never returned).

**Response `200 OK`:**

```json
{
  "username": "mykosynuser",
  "created_at": "2026-03-17T12:00:00Z",
  "updated_at": "2026-03-17T12:00:00Z"
}
```

**Error responses:**

| Status | Description |
|--------|-------------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No KOSync credentials configured yet |

---

#### `PUT /api/kosync/credentials` 🔒 **JWT only**

Create or update the current user's KOSync credentials.

**Request body:**

| Field      | Type   | Required | Description |
|------------|--------|----------|-------------|
| `username` | string | ✓ | KOSync username (max 256 chars, case-insensitive, globally unique) |
| `password` | string | ✓ | KOSync password (min 8 chars) |

**Response `200 OK`:** Same shape as `GET /api/kosync/credentials`.

**Error responses:**

| Status | Description |
|--------|-------------|
| `400 Bad Request` | Missing/empty `username` or `password`, or fails validation |
| `401 Unauthorized` | Missing or invalid JWT |
| `409 Conflict` | Username already taken by another Biblioteka user |

---

#### `DELETE /api/kosync/credentials` 🔒 **JWT only**

Delete the current user's KOSync credentials. KOReader sync will return `401` after this.

**Response:** `204 No Content`

**Error responses:**

| Status | Description |
|--------|-------------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No KOSync credentials configured |

---

### Protocol endpoints (kosync-compatible)

These endpoints are called automatically by KOReader using `x-auth-user` and `x-auth-key` headers (not JWT or API key). They are rate-limited to 5 req/s per IP (burst 10).

#### `POST /api/user/create`

KOReader always attempts registration before authentication. Biblioteka always returns `409 Conflict` because account creation is managed through the web UI. KOReader treats `409` as "account already exists" and proceeds to the auth step.

---

#### `GET /api/user/auth`

Verify KOSync credentials. Returns `{"authorized":"OK"}` if the `x-auth-user` / `x-auth-key` headers are valid.

**Response `200 OK`:**

```json
{ "authorized": "OK" }
```

---

#### `PUT /api/syncs/progress`

Save or update reading progress for a document.

**Request body:**

| Field        | Type   | Required | Description |
|--------------|--------|----------|-------------|
| `document`   | string | ✓ | KOReader document identifier (no `/` characters, max 1024 chars) |
| `progress`   | string | ✓ | KOReader position string (max 4096 chars) |
| `percentage` | number | ✓ | Reading percentage in `[0, 1]` |
| `device`     | string |   | Device name (max 256 chars) |
| `device_id`  | string |   | Device identifier (max 256 chars) |

**Response `200 OK`:**

```json
{
  "document": "mybook-abc123",
  "progress": "1/3/4/5/6/7/8",
  "percentage": 0.42,
  "device": "Kindle Paperwhite",
  "device_id": "abc123",
  "timestamp": 1742220000
}
```

`timestamp` is a Unix epoch second.

---

#### `GET /api/syncs/progress/{document}`

Retrieve the latest reading progress for a document.

**Response `200 OK`:** Same shape as the `PUT` response above.

**Error responses:**

| Status | Description |
|--------|-------------|
| `401 Unauthorized` | Missing or invalid `x-auth-user` / `x-auth-key` |
| `404 Not Found` | No progress stored for this document |

---

