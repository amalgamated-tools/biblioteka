# Administration Guide

This guide covers day-to-day administration of a Biblioteka instance: managing users, reviewing audit logs, monitoring background jobs, and maintaining libraries.

---

## First-Time Setup

The **first account** to sign up on a fresh instance is automatically granted admin privileges. There is no separate admin creation step — just navigate to your Biblioteka instance and register.

After signing up, complete the initial configuration:

1. **Set a strong JWT secret** — ensure `JWT_SECRET` in your environment is a long random value (e.g. `openssl rand -hex 32`). The default value is insecure.
2. **Enable secure cookies** — set `SECURE_COOKIES=true` if your instance is behind HTTPS (recommended for any non-local deployment).
3. **(Optional) Configure OIDC** — see the [Authentication guide](authentication.md) to set up SSO.
4. **Create libraries** — add at least one library with filesystem paths pointing to your book collection. Biblioteka will scan those paths for supported file types (`.epub`, `.mobi`, `.pdf`, `.azw3`).

---

## User Management

Admins can manage all user accounts via **Settings → Users** in the web UI or directly via the API.

### List users

```bash
curl http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer <admin-jwt>"
```

**Response:**

```json
[
  {
    "id": "<id>",
    "name": "Alice",
    "email": "alice@example.com",
    "is_admin": true,
    "oidc_linked": false,
    "created_at": "2026-03-14T02:00:00Z"
  },
  {
    "id": "<id>",
    "name": "Bob",
    "email": "bob@example.com",
    "is_admin": false,
    "oidc_linked": true,
    "created_at": "2026-03-15T10:00:00Z"
  }
]
```

### Grant or revoke admin access

```bash
# Grant admin
curl -X PUT http://localhost:8080/api/admin/users/<user-id> \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"is_admin": true}'

# Revoke admin
curl -X PUT http://localhost:8080/api/admin/users/<user-id> \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"is_admin": false}'
```

**Notes:**
- There is no user deletion API. Remove accounts directly in the database if needed (see [Deleting a user account](#deleting-a-user-account) below).
- An admin cannot revoke their own admin status via the API. Use a second admin account or edit the database directly.

### Deleting a user account

Biblioteka does not expose a user deletion endpoint. To remove an account, connect to the database directly and issue a `DELETE` statement against the `users` table.

#### Before deleting a user

Work through this checklist before issuing the `DELETE` statement:

1. **Review libraries the user previously created or managed.** Libraries are global resources and are not owned by a user record. Deleting the account will not reassign or delete any libraries. Before removing the user, confirm whether any shared libraries they set up still need to remain in place, and update or remove those libraries through the admin UI or API if necessary.

2. **Check for user-scoped background jobs before deletion.** Review the job queue in Redis or the Asynqmon dashboard (`/asynqmon/`) for any queued or in-progress jobs whose payload includes the user's ID, such as `enrich:ai` or `enrich:goodreads`. Allow those jobs to finish, or cancel them, before proceeding. `scan:library` jobs reference `library_id` and paths rather than `user_id`, so they do not by themselves create an orphaned `user_id` risk when deleting a user.

3. **Confirm audit log retention.** Rows in `audit_logs` that reference the deleted `user_id` are intentionally retained for accountability — there is no foreign key constraint on `audit_logs.user_id`. After deletion those rows will contain an orphaned `user_id` value, which is expected behavior. Ensure your compliance or audit policies accept this before proceeding.

**SQLite**

```bash
# Connect to the SQLite database (adjust path as needed)
sqlite3 /data/biblioteka.db
```

```sql
-- Enable foreign-key enforcement (required for cascades to fire in SQLite)
PRAGMA foreign_keys = ON;

-- Delete the user — all owned data cascades automatically (see below)
DELETE FROM users WHERE id = '<user-id>';
```

**PostgreSQL**

```sql
-- Foreign-key cascades fire automatically in PostgreSQL
DELETE FROM users WHERE id = '<user-id>';
```

#### What is deleted automatically (cascades)

The following rows are deleted along with the user record via `ON DELETE CASCADE` foreign keys to `users`:

| Table | Contents removed |
|-------|-----------------|
| `api_keys` | All API keys belonging to the user |
| `opds_credentials` | The user's OPDS Basic Auth credential |
| `kosync_credentials` | The user's KOSync credential |
| `kobo_tokens` | All Kobo sync tokens |
| `kobo_reading_states` | All Kobo reading progress records |
| `reading_progress` | All KOReader sync progress records |
| `reading_lists` and `reading_list_books` | All reading lists and their book entries |
| `reading_groups` | All reading groups owned by the user |
| `reading_group_members` | All reading group memberships for the user |
| `reading_group_lists` | All reading lists the user shared into groups |
| `passkey_credentials` and `passkey_challenges` | Any registered passkeys |
| `goodreads_metadata` | Any Goodreads enrichment data linked to the user |
| `book_annotations` | All annotations created by the user |
| `book_downloads` | All download-tracking events for the user |
| `ai_enrichments` | Any AI metadata enrichment requests |

> **Libraries are not deleted.** Libraries are global resources managed by admins. Deleting a user does not remove any libraries. Existing library–book associations (`library_books`) are also unaffected. **Books themselves are not deleted** — they remain in the catalog and retain their authors, series, and tags.

#### What is NOT deleted

- **Books, authors, series, and tags** — catalog records are global and are not removed.
- **Audit log entries** — `audit_logs` rows referencing the deleted `user_id` are retained for historical accountability. The `user_id` field on those rows becomes an orphaned reference, which is intentional: Biblioteka does not place a foreign key constraint on `audit_logs.user_id` so audit records survive account removal. This preserves the audit event and associated `user_id` even after the account no longer exists.

> **SQLite note:** The cascades above only fire when `PRAGMA foreign_keys = ON` is set for the connection. The application always enables this pragma, but if you connect via a separate client (e.g. `sqlite3` CLI or a GUI tool), you must set it yourself before issuing the `DELETE`.

### The `oidc_linked` field

`oidc_linked: true` means the account is linked to an OIDC/SSO provider. The user can log in via their SSO provider without a password. They may also retain a local password if the account was created locally before linking.

### Controlling self-registration

By default, anyone who can reach your Biblioteka instance can create an account. To prevent new self-service registrations (for example, in a single-user or invite-only deployment), set the `DISABLE_SIGNUP` environment variable:

```bash
DISABLE_SIGNUP=true
```

When enabled, `POST /api/auth/signup` returns `403 Forbidden` and the **Sign Up** tab is hidden in the web UI. The first admin account retains full access. The `GET /api/auth/signup/enabled` endpoint returns `{"enabled": false}` so clients can adapt their UI accordingly.

> **Tip:** If you are the sole user, deploy with signup enabled initially, create your first admin account, then set `DISABLE_SIGNUP=true` and redeploy to prevent further self-service registrations.

---

## Audit Logs

Biblioteka records an append-only audit trail of all significant actions. Use this to track who changed what and when.

### Viewing audit logs

```bash
# Most recent 50 entries
curl http://localhost:8080/api/audit-logs \
  -H "Authorization: Bearer <admin-jwt>"

# Paginate: skip the first 50, return the next 50
curl "http://localhost:8080/api/audit-logs?limit=50&offset=50" \
  -H "Authorization: Bearer <admin-jwt>"
```

**Response:**

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

Entries are returned newest-first. `limit` defaults to `50` (maximum `200`); `offset` skips that many entries.

### Action reference

| `action`               | `entity_type` | `metadata` fields                                | Trigger                                  |
|------------------------|---------------|--------------------------------------------------|------------------------------------------|
| `user.signed_up`       | `user`        | `email`, `name`                                  | New account via `POST /api/auth/signup`  |
| `user.admin_updated`   | `user`        | `is_admin`                                       | Admin toggle via `PUT /api/admin/users/{id}` |
| `user.profile_updated` | `user`        | `name`                                           | Display name changed via `PUT /api/auth/me` |
| `library.created`      | `library`     | `name`                                           | `POST /api/libraries`                   |
| `library.updated`      | `library`     | `name`                                           | `PUT /api/libraries/{id}`               |
| `library.deleted`      | `library`     | `name`                                           | `DELETE /api/libraries/{id}`            |
| `book.created`         | `book`        | `title`                                          | `POST /api/books`                       |
| `book.updated`         | `book`        | `title`                                          | `PUT /api/books/{id}`                   |
| `book.deleted`         | `book`        | `title`                                          | `DELETE /api/books/{id}`                |
| `author.created`       | `author`      | `name`                                           | `POST /api/authors`                     |
| `author.updated`       | `author`      | `name`                                           | `PUT /api/authors/{id}`                 |
| `author.deleted`       | `author`      | `name`                                           | `DELETE /api/authors/{id}`              |
| `series.created`       | `series`      | `name`                                           | `POST /api/series`                      |
| `series.updated`       | `series`      | `name`                                           | `PUT /api/series/{id}`                  |
| `series.deleted`       | `series`      | `name`                                           | `DELETE /api/series/{id}`               |
| `book_file.created`    | `book_file`   | `book_id`, `file_name`, `file_type`              | `POST /api/books/{id}/files`            |
| `book_file.deleted`    | `book_file`   | `book_id`, `file_name`, `file_type`              | `DELETE /api/book-files/{id}`           |
| `api_key.created`      | `api_key`     | `name`                                           | `POST /api/api-keys`                    |
| `api_key.deleted`      | `api_key`     | `name`                                           | `DELETE /api/api-keys/{id}`             |
| `opds_credential.updated` | `opds_credential` | `username`                               | `PUT /api/opds/credentials`             |
| `opds_credential.deleted` | `opds_credential` | `username`                               | `DELETE /api/opds/credentials`          |
| `kobo_token.created`   | `kobo_token`  | `name`                                           | `POST /api/kobo/tokens`                 |
| `kobo_token.deleted`   | `kobo_token`  | `name`                                           | `DELETE /api/kobo/tokens/{id}`          |
| `kosync_credential.updated` | `kosync_credential` | `username`                           | `PUT /api/kosync/credentials`           |
| `kosync_credential.deleted` | `kosync_credential` | `username`                           | `DELETE /api/kosync/credentials`        |
| `smtp.config_updated`  | `config`      | `host`, `from`                                   | `PUT /api/config/smtp`                  |
| `fts.rebuilt`          | `fts`         | —                                                | `POST /api/admin/search/reindex`        |
| `watch_folder.config_updated` | `config` | `path`, `library_id`                             | `PUT /api/config/watch-folder`          |
| `llm.config_updated`   | `config`      | —                                                | `PUT /api/config/llm`                   |

**Notes:** `user_id` is the actor who performed the action (`null` for system/background actions). Entries are append-only and never modified. Book files created by the background scanner do **not** currently produce an audit entry — only files created via the API are audited. Background imports run without an authenticated user context (there is no actor to attribute the action to), so they cannot be represented in the same audit model as user-initiated writes.

---

## Background Job Monitoring

Biblioteka uses Redis-backed background jobs to scan library paths and import book files. When Redis is configured, the [Asynqmon](https://github.com/hibiken/asynqmon) dashboard is available at `/asynqmon/`.

### Accessing the dashboard

Navigate to `http://<your-host>/asynqmon/` in a browser while signed in as an admin. The session cookie is sent automatically.

If accessing via an API client or a tool without cookie support:

```bash
curl http://localhost:8080/asynqmon/ \
  -H "Authorization: Bearer <admin-jwt>"
```

### What to look for

| Queue view | Meaning |
|------------|---------|
| **Pending** | Jobs waiting to be picked up by a worker |
| **Active** | Jobs currently being processed (up to 4 concurrently by default) |
| **Completed** | Successfully finished jobs (retained briefly for inspection) |
| **Failed** | Jobs that exhausted all retries (default: 5 attempts) |
| **Scheduled** | Jobs queued to run at a future time |

### Common failure causes

Before retrying a failed job, check the application logs to understand the root cause. The most common reasons a `process:file` job fails are:

- **ExifTool not on `PATH`** — metadata extraction requires ExifTool to be installed and accessible. Without ExifTool, the `process:file` job still succeeds but falls back to filename-derived metadata only (no title, author, or ISBN from file contents). A `DEBUG`-level log line mentioning `exiftool is not available on this system` confirms this. Install ExifTool and restart the server to enable full extraction. See [Metadata — Installing ExifTool](metadata.md#installing-exiftool) for platform-specific instructions. The Dockerfiles in this repository include ExifTool by default.
- **Filesystem permission errors** — the server process must be able to read every file it scans and write to library directories when file organization is enabled. A `permission denied` error in the logs indicates the process user lacks the required access. Check directory ownership and permission bits, and ensure the user running Biblioteka can read (and, if organizing files, write) the library paths.
- **Cross-filesystem move failures** — when file organization is enabled and the source and destination are on different filesystems, Biblioteka falls back to a copy-then-delete. If the destination filesystem is full or read-only the job will fail. See [File Organization](#file-organization) for details on how reorganization failures are handled.
- **Corrupt or unreadable files** — a file that ExifTool cannot parse (e.g. a truncated EPUB) logs a `WARN`-level `metadata extraction failed` entry and continues with filename-derived metadata. The `process:file` job does not fail; the file is still imported with a minimal record. Inspect `WARN`-level log entries to identify which files produced degraded metadata.

To find the error for a specific failed job, look up the job's `task_id` in the Asynqmon UI and then search the logs:

```bash
docker compose logs biblioteka | jq 'select(.task_id == "<task-id>")'
```

### Retrying failed jobs

From the Asynqmon UI, select a failed job and click **Retry**. The job re-enters the pending queue and will be processed shortly.

To retry all failed jobs in the default queue via the Asynqmon API:

```bash
curl -X POST "http://localhost:8080/asynqmon/api/queues/default/tasks:batch-run" \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"task_ids":["all"]}'
```

### Job pipeline

A library scan cascades through four job types:

```
scan:libraries (scheduled every 24h)
 └─▶ scan:library  (one per monitored library)
      └─▶ scan:path  (one per library path)
           └─▶ process:file  (one per supported file found)
```

See [Background Jobs](background-jobs.md) for full details on each job type, payloads, and how to add new jobs.

### Forcing a manual scan

Creating a library via the API immediately enqueues a `scan:library` job. Updating an existing library does **not** trigger an automatic re-scan — to force a re-scan, delete and recreate the library, or wait for the next scheduled 24-hour scan.

```bash
# Create a library — triggers an immediate scan
curl -X POST http://localhost:8080/api/libraries \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Fiction",
    "paths": ["/mnt/books/fiction"],
    "monitored": true
  }'
```

---

## Managing Libraries

Libraries are global collections of filesystem paths. Any authenticated user can view libraries; only **admins** can create, update, or delete them.

### Create a library

```bash
curl -X POST http://localhost:8080/api/libraries \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Science Fiction",
    "paths": ["/mnt/books/scifi"],
    "organization_type": "book_per_folder",
    "monitored": true
  }'
```

| Field               | Required | Default             | Description                                           |
|---------------------|----------|---------------------|-------------------------------------------------------|
| `name`              | ✓        | —                   | Unique library name                                   |
| `paths`             | —        | `[]`                | Filesystem paths to scan                              |
| `organization_type` | —        | `"book_per_folder"` | How the library is organised (see note below)         |
| `monitored`         | —        | `false`             | Include in scheduled 24-hour scans                    |

> **`organization_type`:** Supported values are `"book_per_folder"` (`Author/Title/file`), `"book_per_file"` (`Author/file`), and `"none"` (leave files in place).

> **Web UI scan feedback:** When you add a library through the web interface, the book list automatically switches to a "Scanning library..." state with a live spinner instead of showing the empty-state placeholder. Once the background scan finishes and books appear, the list refreshes automatically — no page reload required. A 5-minute safety timeout clears the scanning indicator if the UI misses the completion signal. See [BookList scan-aware polling](frontend.md#scan-aware-polling) for implementation details.

### Edit and delete libraries

Use `PUT /api/libraries/{id}` to update a library and `DELETE /api/libraries/{id}` to remove it. Both operations require admin privileges. Deleting a library removes only the library record and its book associations — the underlying book, author, series, and book file records are not deleted.

### Book file path validation

When a book file is registered via `POST /api/books/{id}/files`, the server resolves the submitted `file_path` to a canonical absolute path and checks it against all configured library roots. If the path falls outside every library root, the request is rejected with `400 Bad Request`:

```
file path is outside allowed library directories
```

This validation also applies at download time: if a previously registered file's stored path no longer falls within any library root (for example, after a library is removed), the download endpoints (`GET /download/{bookID}/{format}` and the OPDS download endpoint) return `403 Forbidden` instead of serving the file.

---

## Watch Folder

The watch folder lets Biblioteka automatically import any book file dropped into a designated directory. When enabled, the server runs a background scan every minute and enqueues a `process:file` job for each new file it finds.

> **Requires Redis and an active worker.** Watch folder scanning runs as a background job. The feature is only active when the background worker is running (server mode `all` or `worker`). Redis must be reachable; the server defaults `REDIS_URL` to `redis://localhost:6379` if not explicitly set.

### Configure the watch folder

Set the watch folder path and target library from **Settings → Watch Folder** in the web UI, or via the API:

```bash
# Set a watch folder
curl -X PUT http://localhost:8080/api/config/watch-folder \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/mnt/incoming",
    "library_id": "<library-uuid>"
  }'

# Clear the watch folder (disables scanning)
curl -X PUT http://localhost:8080/api/config/watch-folder \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"path": ""}'
```

**Requirements:**

- `path` must be an absolute path to an existing directory on the server's filesystem.
- `library_id` must be the ID of an existing library. Every imported file is associated with this library.
- Sending an empty `path` clears both `path` and `library_id`, disabling the watch folder.

### How it works

Every minute, the `scan:watch-folder` background job reads the configured path and library ID, then runs the same `ScanDirectory` pipeline used by the regular library scanner. Each supported file (`.epub`, `.mobi`, `.azw3`, `.pdf`) is enqueued as a `process:file` job. Already-indexed file paths are skipped — the scan is safe to run repeatedly.

If the library has a `book_per_folder` or `book_per_file` organization type, newly imported files are moved into the library's canonical directory structure. Otherwise they remain in the watch folder path.

> **Audit trail:** Changes to the watch folder configuration are recorded in the audit log as `watch_folder.config_updated`.

See [API Reference — Watch folder endpoints](api/config.md#get-apiconfigwatch-folder--admin--jwt-only) for the full endpoint shape and error codes.

---

## OIDC Configuration (Runtime)

Admins can configure OIDC at runtime without a server restart via **Settings → SSO** or the API:

```bash
# Get current OIDC config
curl http://localhost:8080/api/config/oidc \
  -H "Authorization: Bearer <admin-jwt>"

# Set OIDC config
curl -X PUT http://localhost:8080/api/config/oidc \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "issuer_url":    "https://sso.example.com/realms/my-realm",
    "client_id":     "biblioteka",
    "client_secret": "<secret>",
    "redirect_uri":  "https://books.example.com/api/auth/oidc/callback"
  }'
```

Before attempting provider discovery, the server validates the issuer URL against a set of SSRF-prevention rules. The following are rejected with `400 Bad Request`:

- Non-`https` schemes (only `https://` is accepted)
- URLs containing userinfo (e.g. `https://user:pass@issuer.example.com`)
- Literal private, loopback, link-local, or unique-local IP addresses (RFC 1918, `127.0.0.0/8`, `169.254.0.0/16`, `100.64.0.0/10` per RFC 6598, IPv6 loopback `::1`, link-local `fe80::/10`, and unique-local `fc00::/7`)
- Hostnames that resolve via DNS to any of the above address ranges (DNS failures are treated as pass — the OIDC discovery request itself will fail and the SSRF-safe HTTP client provides a second check at connection time)
- IPv6 zone identifiers in the host (e.g. `[fe80::1%lo0]`)

If validation passes, the server performs OIDC discovery. If discovery fails, the config is rejected with a `400` error.

All four settings (`issuer_url`, `client_id`, `client_secret`, `redirect_uri`) are saved atomically in a single database transaction. If the write fails, none of the settings are changed — the configuration is never left in a partially-updated state.

**Precedence:** Environment variables (`OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI`) always override database-stored settings. If environment variables are set, the runtime configuration UI will appear read-only.

---

## SMTP Configuration (Runtime)

Admins can configure SMTP at runtime without a server restart via **Settings → SMTP** or the API:

```bash
# Get current SMTP config (password is never returned)
curl http://localhost:8080/api/config/smtp \
  -H "Authorization: Bearer <admin-jwt>"

# Set SMTP config
curl -X PUT http://localhost:8080/api/config/smtp \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "host":     "smtp.example.com",
    "port":     "587",
    "username": "mailer@example.com",
    "password": "<password>",
    "from":     "biblioteka@example.com",
    "tls":      "starttls"
  }'

# Send a test email to verify the configuration
curl -X POST http://localhost:8080/api/config/smtp/test \
  -H "Authorization: Bearer <admin-jwt>"
```

The test endpoint sends a short verification email to the authenticated admin's registered email address. It returns `200 OK` with a `{"message":"Test email sent to <email>"}` body on success, or a `4xx`/`5xx` error with `{"error":"…"}` on failure.

All six SMTP settings (`host`, `port`, `username`, `password`, `from`, `tls`) are saved atomically in a single database transaction. If the write fails, none are changed — the configuration is never left in a partially-updated state.

**TLS modes:** `none` (plaintext), `starttls` (STARTTLS upgrade on port 587, default), or `tls` (implicit TLS on port 465).

**Precedence:** When the `SMTP_HOST` environment variable is set, all SMTP settings are read exclusively from environment variables (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`) and the database values are ignored. The runtime configuration UI will appear read-only. When `SMTP_HOST` is unset (the default), the values stored in the database via the API or Settings UI are used.

See [API reference — SMTP config endpoints](api/config.md#get-apiconfigsmtp--admin--jwt-only) for full request/response shapes.

### Emailing book files to readers

When SMTP is configured, any authenticated user can send a book file as an email attachment to any valid address from the library UI (useful for e-readers like Kindle). The mail icon on each book card opens the send dialog; books with multiple files show a file selector first. Files larger than 25 MB are rejected with `413 Request Entity Too Large`, and each successful send is recorded in the audit log as `book_file.emailed`.

If SMTP is not configured, the request is rejected with `400 Bad Request` and the UI disables the send button accordingly.

See [API reference — `POST /api/book-files/{id}/email`](api/books.md#post-apibook-filesidemail) for the full endpoint shape and error codes.

---

## LLM Configuration (Runtime)

Biblioteka can enrich book metadata using a locally-hosted large language model. When configured, the **AI Enrich** action on a book detail page enqueues a background job that sends the book's title, authors, and existing description to the LLM and stores the suggested genres, themes, mood, reading level, tags, and generated catalog description as a pending review record. The user can then apply or reject the suggestions without committing them automatically.

> **Requires Redis.** AI enrichment runs as a background job; a Redis worker must be running. See [Background Jobs](background-jobs.md#enrichai) for details on the `enrich:ai` job.

### Supported providers

Currently only [Ollama](https://ollama.com/) is supported. Run Ollama locally or on a server reachable from the Biblioteka host.

### Configuring LLM access

Admins can configure the LLM at runtime via **Settings → AI Enrichment** or the API:

```bash
# Get current LLM config
curl http://localhost:8080/api/config/llm \
  -H "Authorization: Bearer <admin-jwt>"

# Enable and configure Ollama
curl -X PUT http://localhost:8080/api/config/llm \
  -H "Authorization: Bearer <admin-jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider":  "ollama",
    "endpoint":  "http://localhost:11434",
    "model":     "llama3",
    "enabled":   true
  }'
```

| Field | Required when `enabled` | Description |
|-------|------------------------|-------------|
| `provider` | No (defaults to `"ollama"`) | LLM provider name. Currently only `"ollama"` is accepted. |
| `endpoint` | Yes | Base URL of the Ollama server (e.g. `"http://ollama.example.com:11434"`). Subject to SSRF validation — see below. |
| `model` | Yes | Ollama model name (e.g. `"llama3"`, `"mistral"`, `"gemma3"`). The model must already be pulled on the Ollama server. |
| `enabled` | — | `true` to activate AI enrichment; `false` to disable it. |

#### Endpoint validation (SSRF protection)

Before saving, the server validates the `endpoint` URL against the following rules. Violations are rejected with `400 Bad Request`:

- Only the `http` and `https` schemes are accepted.
- URLs containing userinfo (e.g. `http://user:pass@ollama.internal`) are rejected to prevent credential leakage.
- Literal private, loopback, link-local, unique-local, and other non-routable IP addresses are blocked, including RFC 1918 ranges, `0.0.0.0/8` ("this" network), `127.0.0.0/8`, `169.254.0.0/16`, `100.64.0.0/10` per RFC 6598, IPv6 unspecified `::/128`, IPv6 loopback `::1`, link-local `fe80::/10`, and unique-local `fc00::/7`.
- IPv6 literals with zone identifiers (e.g. `http://[fe80::1%lo0]:11434`) are rejected.
- If the host is a DNS name, it is resolved (with a short timeout) and any private or loopback result is also blocked. DNS failures are treated as a pass — the enrichment job will fail when it actually connects, and a second SSRF-safe dialer in the Ollama client provides an additional layer of defense.

> **Local Ollama note:** Because private IP addresses are blocked, `http://localhost:11434` and `http://127.0.0.1:11434` are rejected when set via the API or admin UI. For a local Ollama instance on the same host as Biblioteka, point the endpoint at a non-private, publicly routable hostname or address that the Biblioteka process can reach (for example, the host's public IP or a hostname that resolves to it). If Ollama is on a separate private-network host, access it via a reverse proxy reachable from a routable hostname.

A successful update is recorded in the audit log as `llm.config_updated`.

> **Restart required.** LLM configuration is read once at server startup. After saving a new configuration via the API, **restart the server** (or the worker process if running in split mode) for the change to take effect. The `PUT /api/config/llm` response always includes `"restart_required": true` as a reminder.

See [API reference — LLM config endpoints](api/config.md#get-apiconfigllm--admin--jwt-only) for the full request/response shapes.

---

## File Organization

Biblioteka can automatically move imported book files into an organized directory structure under each library root. This keeps your collection tidy and makes paths predictable.

File organization is configured per-library via the **File Organization** dropdown when creating or editing a library. The available modes are:

| Mode | Directory Structure | Description |
|------|-------------------|-------------|
| **Book Per Folder** (default) | `Author/Title/file` | Each book gets its own folder under the author |
| **Multiple Books Per Author** | `Author/files` | Books are placed directly in the author folder |
| **No Organization** | (unchanged) | Files are left where they are |

### How it works

When a library's `organization_type` is set to `book_per_folder` or `book_per_file`, the `process:file` job moves each imported file into the corresponding directory structure under the library root.

The author and title come from embedded file metadata when available, falling back to values parsed from the file's existing directory structure (see [Path-based metadata](background-jobs.md#path-based-metadata)).

**Behaviour details:**

- `book_per_folder` requires both author and title; `book_per_file` requires only an author. If the required fields are absent after metadata extraction, the file stays in place.
- Directory names are sanitized: path separators (`/`, `\`), control characters, colons, wildcards, and leading dots are removed. As a defense-in-depth measure, the computed target path is also verified to stay within the library root, guarding against path traversal via untrusted author/title metadata embedded in book files.
- The move uses `os.Rename` when source and destination are on the same filesystem. A copy-then-delete falls back for cross-filesystem moves; source file permissions and modification time are preserved.
- Empty source directories left behind after a move are removed automatically (up to but not including the library root).
- If a file already exists at the target path, the move is skipped and a warning is logged — existing files are never silently overwritten.
- If reorganization fails (e.g. permissions error), a warning is logged and the import completes at the original path.

### Path-parsing and series inference

Even when file organization is set to `none`, Biblioteka parses each file's path relative to the library root to extract author, title, and series from the directory structure. Trailing `(YYYY)` year tokens are stripped to keep titles clean (the year is not stored as `publication_date`). This path-derived metadata supplements (but does not override) embedded file metadata.

For full details on the supported directory layouts and precedence rules, see [Background Jobs — Path-based metadata](background-jobs.md#path-based-metadata).

### Sidecar files

Every time a book file is imported — regardless of whether file organization is enabled — Biblioteka writes two sidecar files in the same directory as the book file:

| File | Contents |
|------|----------|
| `cover.<ext>` | Cover image (JPEG, PNG, WebP, or AVIF) decoded from the embedded EPUB cover, when available. Skipped for non-EPUB formats and when no cover is present. |
| `metadata.opf` | OPF 2.0 Dublin Core metadata: title, author, identifier, language, publication date, publisher, and description. |

These writes are best-effort. Failures are logged at `WARN` level and do not prevent the book from being imported.

> **`book_per_file` libraries:** When `organization_type` is `book_per_file`, multiple books share the same author directory, so sidecar filenames are prefixed with the book's filename stem to avoid collisions. For example, a book file named `gatsby.epub` produces `gatsby.jpg` and `gatsby.opf` instead of `cover.jpg` and `metadata.opf`.

When file organization is enabled, sidecar files are placed in the final `<Author>/<Title>/` directory alongside the relocated book file. See [Background Jobs — Sidecar files](background-jobs.md#sidecar-files) for implementation details.

---

## Health Check

Use the health endpoint to verify the server is running:

```bash
curl -sf http://localhost:8080/api/health
# → {"status":"ok"}
```

This endpoint requires no authentication and is suitable for liveness/readiness probes.

---

## Search Index Maintenance (SQLite)

Biblioteka uses an [FTS5](https://www.sqlite.org/fts5.html) virtual table (`books_fts`) to accelerate full-text search on SQLite. The index is **maintained automatically** by database triggers on every insert, update, and delete. No routine maintenance is needed.

### Startup integrity check

At startup, Biblioteka runs an FTS5 integrity check. If corruption is detected (which can happen after running SQLite's `VACUUM` command, because `VACUUM` may remap the implicit rowids that the content-table index relies on), the server automatically rebuilds the index and logs a warning. Startup is never aborted due to an FTS failure, but startup may take longer while the rebuild runs. If the rebuild does not succeed, search requests may fail because the `books_fts` table remains broken.

### Manual rebuild

Admins can trigger a full rebuild via **Settings → Search Index** in the web UI or via the API.

This is useful after bulk imports (Calibre import, library scan) that may temporarily leave the search index stale, after running `VACUUM` on the database file outside of normal server operation, or whenever you suspect the index is out of sync.

**Web UI:** Navigate to **Settings → Search Index** and click **Rebuild Search Index**. A success banner confirms the request was accepted and the rebuild has started. Completion is indicated later by an `fts.rebuilt` entry in the audit log (see below).

**API:**

```bash
curl -i -X POST http://localhost:8080/api/admin/search/reindex \
  -H "Authorization: Bearer <admin-jwt>"
# HTTP/1.1 202 Accepted
# → {"message":"search index rebuild started"}
```

The endpoint returns `202 Accepted` immediately and runs the rebuild in the background. A successful rebuild emits an audit log entry with action `fts.rebuilt` (`entity_type: fts`).

> **PostgreSQL:** The pg_trgm GIN indexes used for search on PostgreSQL are maintained automatically by the database engine. On PostgreSQL instances, this endpoint returns `200 OK` with a message indicating no rebuild is required and does not emit an audit log entry.

---

## Log-Based Troubleshooting

Biblioteka writes structured JSON logs to stdout. Increase verbosity by setting `LOG_LEVEL=debug`:

```bash
LOG_LEVEL=debug LOG_FORMAT=text docker compose up
```

Useful patterns:

```bash
# Watch all ERROR-level entries
docker compose logs -f biblioteka | jq 'select(.level == "ERROR")'

# Trace a specific request by its ID
docker compose logs biblioteka | jq 'select(.request_id == "<id>")'

# See all background job activity
docker compose logs biblioteka | jq 'select(.msg | test("job|scan|process|file"))'
```

See the [Observability guide](observability.md) for the full log field reference.
