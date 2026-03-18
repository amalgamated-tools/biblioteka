# Kobo E-Reader Sync

Biblioteka includes a native sync server compatible with the Kobo e-reader API, so your Kobo device can browse and download your entire library—and keep reading progress in sync—just like it would with the official Kobo store.

## How it works

When you add a Kobo sync token in Biblioteka, you receive a unique URL of the form:

```
https://<your-host>/kobo/<token>
```

Point your Kobo device's "Content Library" sync endpoint at this URL. The device authenticates using the token embedded in the URL (not a username/password), and Biblioteka serves the Kobo device API from that path. From the device's perspective it looks like a Kobo store — books appear in the library, cover images load, downloads work, and reading progress is saved back to the server.

> **Privacy note:** Anyone who obtains a token URL can access your library as that user. Treat it like an API key: keep it private and revoke it immediately if compromised.

---

## Setting up your Kobo device

### 1. Create a Kobo sync token

Open **Settings → Kobo Sync** in the Biblioteka web UI, enter a descriptive name (e.g. `"Kobo Libra 2"`), and click **Generate**. The generated token is shown **only once**—copy the full sync URL before closing the dialog.

You can also create a token via the API (see [Kobo Tokens API](#kobo-tokens-api) below).

### 2. Configure the Kobo device

Kobo devices do not expose a sync-server setting in the normal UI. You configure it with a small one-time script run from a connected computer, or by editing a config file on the device's internal storage:

**Method A – Kobo config file (simplest)**

1. Connect your Kobo to a computer via USB.
2. Open the `.kobo/Kobo/Kobo eReader.conf` file in a text editor.
3. Find (or add) the `[OneStoreServices]` section and set:

   ```ini
   [OneStoreServices]
   StoreURL=https://<your-host>/kobo/<token>/
   ```

4. Save the file and safely eject the Kobo.

**Method B – NickelMenu / KFMon plugin**

If you run a custom firmware plugin such as [NickelMenu](https://pgaskin.net/NickelMenu/) or have a debug menu enabled, you can set the store URL through the device's own menu without connecting to a computer.

### 3. Trigger a sync

On your Kobo, open the library and trigger a sync (usually **Sync** in the top-right menu or from the home screen). The device contacts `GET /kobo/<token>/v1/library/sync` to fetch your books. Subsequent syncs use an incremental sync token so only new or changed books are transferred.

---

## What syncs

| Feature | Supported |
|---------|-----------|
| Book list (library sync) | ✅ |
| Book metadata (title, authors, series, description, cover) | ✅ |
| Book downloads (EPUB, KEPUB, MOBI, PDF, AZW3) | ✅ |
| Cover images | ✅ |
| Reading progress (percent read, position, status) | ✅ |
| Kobo store purchases / wishlists | ❌ (not applicable for a self-hosted library) |

### Supported file formats

The sync endpoint exposes files with these Kobo format identifiers:

| Extension | Kobo format |
|-----------|-------------|
| `.epub` | `EPUB3` |
| `.kepub` | `KEPUB` |
| `.mobi` | `MOBI` |
| `.pdf` | `PDF` |
| `.azw3` | `AZW3` |

Books with no supported file format are excluded from the sync response.

### Reading progress

When you read a book on your Kobo, the device periodically pushes reading state updates (reading status, percent read, and bookmark location) to `PUT /kobo/<token>/v1/library/{bookID}/state`. Biblioteka stores this per-user in the `kobo_reading_states` table and includes updated states in subsequent sync responses so progress is preserved even after a factory reset.

Reading states have three statuses that map to Kobo's values:

| Status | Meaning |
|--------|---------|
| `ReadyToRead` | Book has not been opened |
| `Reading` | Book is in progress |
| `Finished` | Book has been marked finished |

---

## Managing sync tokens

Each Biblioteka user can create **multiple** Kobo sync tokens — one per device is recommended so you can revoke a single device without affecting others.

### Via the web UI

Open **Settings → Kobo Sync**. The tab lists all your existing tokens (names and creation dates only; raw tokens are never shown again after creation) and lets you generate or delete tokens.

### Via the API

See [Kobo Tokens API](#kobo-tokens-api) below for the full reference.

---

## Security model

- **Token-based authentication**: Each `/kobo/<token>/...` request is authenticated by looking up the SHA-256 hash of the token against the `kobo_tokens` table. The raw token is never stored.
- **User data isolation**: All sync queries are scoped to the user associated with the token. A device with one token cannot access another user's books.
- **JWT required for token management**: The token management API (`/api/kobo/tokens`) requires a valid JWT session — API keys are not accepted. This prevents an API key from being used to create or enumerate Kobo sync URLs.
- **One-time token display**: The raw token is returned once at creation time only. Biblioteka stores only the hash; if you lose the token URL you must delete it and create a new one.

---

## Kobo Tokens API

These endpoints manage Kobo sync tokens and require a **JWT** (not an API key). The `/api/kobo/tokens` and `/api/kobo/tokens/` paths are also listed in the [JWT-only endpoints](api-reference.md#jwt-only-endpoints) table.

### `GET /api/kobo/tokens` 🔒 **JWT only**

List all Kobo sync tokens for the authenticated user.

**Response `200 OK`:**

```json
[
  {
    "id": "<id>",
    "user_id": "<user_id>",
    "name": "Kobo Libra 2",
    "token_hash": "sha256hex...",
    "created_at": "2026-03-17T12:00:00Z"
  }
]
```

Returns an empty array `[]` if no tokens have been created yet.

---

### `POST /api/kobo/tokens` 🔒 **JWT only**

Create a new Kobo sync token. The raw token value is returned **only in this response** and is never retrievable again.

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
  "token_hash": "sha256hex...",
  "created_at": "2026-03-17T12:00:00Z",
  "token": "a3f8e1b2c4d5..."
}
```

The `token` field contains the raw token. Build the sync URL as:

```
https://<your-host>/kobo/<token>/
```

**Response headers:** `Cache-Control: no-store` and `Pragma: no-cache` are set to prevent the token from being cached by proxies or browsers.

**Error responses:**

| Status | Condition |
|--------|-----------|
| `400 Bad Request` | `name` is missing or empty, or exceeds 100 characters |
| `401 Unauthorized` | Missing or invalid JWT |

---

### `DELETE /api/kobo/tokens/{id}` 🔒 **JWT only**

Delete a Kobo sync token. The Kobo device using this token will receive `401` on its next sync attempt.

**Path parameters:** `{id}` — Kobo token resource ID.

**Response `204 No Content`** on success.

**Error responses:**

| Status | Condition |
|--------|-----------|
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | Token not found or not owned by the authenticated user |

---

## Device API endpoints

The following endpoints are served under `/kobo/<token>/` and are consumed directly by the Kobo firmware. They are documented here for completeness; you do not need to call them manually.

All requests to `/kobo/<token>/...` are authenticated by the middleware which strips the prefix and injects the user ID into the request context before dispatching to the sub-mux.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/kobo/<token>/v1/initialization` | Returns the Kobo API resource map pointing all endpoints back to this server |
| `POST` | `/kobo/<token>/v1/auth/device` | Returns dummy auth tokens (Kobo devices exchange tokens even for sideloaded content) |
| `POST` | `/kobo/<token>/v1/auth/refresh` | Returns dummy refreshed tokens |
| `POST` | `/kobo/<token>/v1/auth/exchange` | Returns dummy exchanged tokens |
| `GET` | `/kobo/<token>/v1/library/sync` | Incremental library sync (returns new/changed entitlements) |
| `GET` | `/kobo/<token>/v1/library/{bookID}/metadata` | Single-book metadata |
| `GET` | `/kobo/<token>/v1/library/{bookID}/state` | Current reading state for a book |
| `PUT` | `/kobo/<token>/v1/library/{bookID}/state` | Update reading state (progress, status, bookmark) |
| `GET` | `/kobo/<token>/download/{bookID}/{format}` | Stream book file to device |
| `GET` | `/kobo/<token>/covers/{bookID}/{width}/{height}/{quality}/{greyscale}/image.jpg` | Serve cover image |

### Sync token (pagination)

The sync endpoint uses a stateless pagination mechanism. The device sends `x-kobo-synctoken: <base64>` on each request; the server returns an updated token in the same header. When there are more results the server also sets `x-kobo-sync: continue` so the device immediately re-requests.

The sync token encodes three high-water marks:

| Field | Description |
|-------|-------------|
| `BooksLastModified` | Timestamp of the most recently seen book modification |
| `BooksLastID` | ID of the most recently seen book (for stable pagination when timestamps tie) |
| `ReadingStateLastModified` | Timestamp of the most recently seen reading state change |

---

## Audit log

Token creation and deletion are recorded in the [audit log](api-reference.md#get-apiaudit-logs--admin):

| Action | Object type | Metadata |
|--------|------------|----------|
| `kobo_token.created` | `kobo_token` | `{"name": "…"}` |
| `kobo_token.deleted` | `kobo_token` | `{"name": "…"}` |

---

## Troubleshooting

**Books are not appearing on my Kobo**

- Confirm the sync URL is set correctly including the trailing slash.
- Check that your books have at least one file in a supported format (EPUB, KEPUB, MOBI, PDF, AZW3). Books with only unsupported formats are excluded from sync.
- Trigger a manual sync from the Kobo menu.

**"Content not available" error on the device**

The sync URL may point to an invalid or deleted token. Generate a new token and update the device config file.

**Reading progress is not saved**

Reading state is pushed to the server when the device syncs. Make sure the Kobo has network connectivity and is regularly syncing (not in airplane mode).

**Cover images are broken**

Cover images are served from `/kobo/<token>/covers/{bookID}/...`. Check that the book has a cover image attached in Biblioteka and that the server is reachable at the configured URL.
