# KOReader Reading Progress Sync

Biblioteka exposes a [kosync](https://github.com/koreader/koreader-sync-server)-compatible API so that [KOReader](https://koreader.rocks/) can synchronize reading positions across devices and back them up to your self-hosted server—no third-party cloud service required.

## How it works

KOReader's **Progress sync** plugin connects to a kosync-compatible server using a dedicated username and password (separate from your main Biblioteka account). When you open, read, or close a book, KOReader pushes a reading position string (the `document` identifier it generates) to `PUT /api/syncs/progress` and retrieves it again on other devices via `GET /api/syncs/progress/{document}`.

Biblioteka stores one progress record per `(user, document)` pair in the `reading_progress` table. Progress is per-Biblioteka-user and completely isolated from other users.

> **Note:** The `document` identifier is chosen by KOReader from the book's file hash or path. It is an opaque string to Biblioteka — the server stores and retrieves it exactly as KOReader sends it.

---

## Setting up KOReader sync

### 1. Create KOSync credentials

Create your KOSync credentials via the API (see [KOSync Credentials API](#kosync-credentials-api) below). There is no Settings UI tab for KOReader Sync — credentials are managed exclusively through the API.

> **Important:** KOSync credentials are separate from your Biblioteka login. Choose a unique username (it must not already be taken by another Biblioteka user) and a strong password.

### 2. Configure KOReader

1. In KOReader, open the **Progress sync** plugin (☰ **Menu → Tools → Progress sync**).
2. Tap **Custom sync server** and enter:

   | Setting | Value |
   |---------|-------|
   | **Custom sync server** | `https://<your-host>` |
   | **Username** | The KOSync username you created in step 1 |
   | **Password** | The KOSync password you created in step 1 |

3. Tap **Register** — Biblioteka always returns `409 Conflict` here because account creation is managed through the API (not via a web form). KOReader treats `409` as "account already exists" and automatically proceeds to the login step.
4. Tap **Login** to authenticate. KOReader will confirm with a ✓ tick if successful.

### 3. Sync reading progress

Once configured, KOReader automatically syncs progress when you open and close books. You can also trigger a manual sync from the Progress sync plugin menu.

---

## Security model

- **Separate credentials**: KOSync username and password are stored independently of your Biblioteka login, so compromising one does not affect the other.
- **bcrypt storage**: The server stores `bcrypt(md5_hex(password))`. KOReader sends the hex-encoded MD5 digest of the password as the `x-auth-key` header (this is the kosync wire protocol). Biblioteka never stores the raw password or the MD5 value—only the bcrypt hash.
- **HTTPS required**: The kosync protocol transmits the hex-encoded MD5 of your password as the `x-auth-key` authentication header on every request. The MD5 value itself is the effective credential — an attacker who can observe or intercept HTTP traffic can replay it to authenticate as you. **You must terminate TLS at a reverse proxy and only access the kosync protocol endpoints over HTTPS in production.** Accessing the sync endpoints over plain HTTP exposes your KOSync credential to interception. See [Deployment → Reverse Proxy Setup](deployment.md#reverse-proxy-setup) for TLS termination guidance.
- **User data isolation**: All reading progress queries are scoped to the authenticated user. One user cannot read or overwrite another user's progress.
- **Rate limiting**: The `/api/user/auth`, `/api/user/create`, and `/api/syncs/progress` endpoints share the same per-IP token-bucket rate limiter used for login (5 requests/second, burst of 10) to mitigate brute-force attacks.
- **Timing-safe auth**: When a username is not found, the server still performs a dummy bcrypt comparison to prevent timing-based username enumeration.

---

## What is synced

| Data | Synced |
|------|--------|
| Reading position (KOReader `progress` string) | ✅ |
| Reading percentage | ✅ |
| Device name | ✅ |
| Device ID | ✅ |
| Bookmarks and annotations | ❌ (not part of the kosync protocol) |
| Book files | ❌ (use [OPDS](opds.md) or direct download for that) |

> **Reading Groups note:** Reading progress synced via KOSync is stored in the `reading_progress` table and is **not** included in [Reading Group](reading-groups.md) progress comparisons. Reading Group progress is sourced exclusively from Kobo native sync data (`kobo_reading_states`). A KOReader user who has synced their position via KOSync but has **no** Kobo-native reading state will appear as `0%` progress in any group progress view, regardless of how far they have read. (A member who also has Kobo-native sync data for the same book will display their Kobo progress instead.) If your reading circle includes members who use KOReader without a Kobo device, they will always show as unstarted in group comparisons. See [Reading Groups](reading-groups.md) for details on how group progress is calculated.

---

## KOSync Credentials API

These endpoints manage KOSync credentials and require a **JWT** (not an API key). They are also listed in the [JWT-only endpoints](api-reference.md#jwt-only-endpoints) table in the API reference.

### `GET /api/kosync/credentials` 🔒 **JWT only**

Returns the current user's KOSync credentials (username and timestamps only — the password hash is never returned).

**Response `200 OK`:**

```json
{
  "username": "mykosynuser",
  "created_at": "2026-03-17T12:00:00Z",
  "updated_at": "2026-03-17T12:00:00Z"
}
```

**Error responses:**

| Status | Condition |
|--------|-----------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No KOSync credentials configured yet |

---

### `PUT /api/kosync/credentials` 🔒 **JWT only**

Create or update the current user's KOSync credentials. Call this endpoint to set up sync for the first time, or to change your KOSync username or password.

**Request body:**

| Field      | Type   | Required | Description                             |
|------------|--------|----------|-----------------------------------------|
| `username` | string | ✓        | KOSync username (max 256 chars, case-insensitive, must be globally unique) |
| `password` | string | ✓        | KOSync password (min 8 chars)           |

**Response `200 OK`:**

```json
{
  "username": "mykosynuser",
  "created_at": "2026-03-17T12:00:00Z",
  "updated_at": "2026-03-18T09:30:00Z"
}
```

**Error responses:**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | `username` or `password` missing, empty, or fails validation |
| `401 Unauthorized` | Missing or invalid JWT |
| `409 Conflict` | Username already taken by another Biblioteka user |

---

### `DELETE /api/kosync/credentials` 🔒 **JWT only**

Delete the current user's KOSync credentials. Subsequent sync attempts from KOReader will return `401 Unauthorized`.

**Response `204 No Content`** on success.

**Error responses:**

| Status | Condition |
|--------|-----------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | No KOSync credentials configured |

---

## Protocol endpoints

The following endpoints implement the KOReader kosync protocol. They are consumed automatically by KOReader and do not require manual interaction.

Authentication uses the `x-auth-user` (username) and `x-auth-key` (hex-encoded MD5 of password) request headers—not JWT or API key.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/user/create` | Always returns `409 Conflict` — account creation is managed through the Biblioteka web UI |
| `GET` | `/api/user/auth` | Verifies KOSync credentials; returns `{"authorized":"OK"}` on success |
| `PUT` | `/api/syncs/progress` | Save or update reading progress for a document |
| `GET` | `/api/syncs/progress/{document}` | Retrieve the latest progress for a document |

### `PUT /api/syncs/progress`

**Request body:**

| Field        | Type   | Required | Description |
|--------------|--------|----------|-------------|
| `document`   | string | ✓ | KOReader document identifier (must not contain `/`) |
| `progress`   | string | ✓ | KOReader position string (e.g. `"1/3/4/5/6/7/8"`) |
| `percentage` | number | ✓ | Reading percentage in the range `[0, 1]` |
| `device`     | string |   | Device name (e.g. `"Kindle Paperwhite"`) |
| `device_id`  | string |   | Device identifier |

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

> `timestamp` is a Unix epoch second.

### `GET /api/syncs/progress/{document}`

**Response `200 OK`:** Same shape as the `PUT` response above.

**Error responses:**

| Status | Condition |
|--------|-----------|
| `404 Not Found` | No progress stored for this document |

---

## Audit log

KOSync credential changes are recorded in the [audit log](api/admin.md#get-apiaudit-logs--admin):

| Action | Object type | Metadata |
|--------|------------|----------|
| `kosync_credential.updated` | `kosync_credential` | `{"username": "…"}` |
| `kosync_credential.deleted` | `kosync_credential` | `{"username": "…"}` |

---

## Troubleshooting

**KOReader shows "Login failed"**

- Confirm the custom sync server URL is correct (`https://<your-host>`) — no trailing slash or path.
- Verify the username and password by calling `GET /api/kosync/credentials` with a Biblioteka JWT. See [KOSync Credentials API](#kosync-credentials-api).
- Try deleting and recreating your credentials, then reconnecting KOReader.

**Progress is not syncing across devices**

- Ensure both KOReader devices are configured to use the same custom sync server and the same KOSync credentials.
- Trigger a manual sync from the Progress sync plugin.
- The `document` identifier is an opaque string generated by KOReader — typically an MD5 hash of the file contents, or an encoded/normalized value derived from the file's path (never the raw path string itself, which is why it contains no `/` characters). If the same book produces a different identifier on each device — because the files have different contents (e.g. different editions or encoding), or because the files are stored at different paths on the two devices — KOReader will treat them as separate documents and maintain independent progress entries. Ensure the same edition of each book is present on both devices at consistent paths to keep identifiers in sync.

**I forgot my KOSync password**

You cannot retrieve a KOSync password. Use `PUT /api/kosync/credentials` (see [KOSync Credentials API](#kosync-credentials-api)) to update your credentials with a new password, then update the password in KOReader's Progress sync settings.
