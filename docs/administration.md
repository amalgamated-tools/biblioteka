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
- There is no user deletion API. Remove accounts directly in the database if needed.
- An admin cannot revoke their own admin status via the API. Use a second admin account or edit the database directly.

### The `oidc_linked` field

`oidc_linked: true` means the account is linked to an OIDC/SSO provider. The user can log in via their SSO provider without a password. They may also retain a local password if the account was created locally before linking.

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

**Pagination:**
- `limit` defaults to `50`, maximum `200`.
- `offset` is the number of entries to skip.
- Entries are returned newest-first.

### Action reference

| `action`               | `entity_type` | `metadata` fields                                | Trigger                                  |
|------------------------|---------------|--------------------------------------------------|------------------------------------------|
| `user.signed_up`       | `user`        | `email`, `name`                                  | New account via `POST /api/auth/signup`  |
| `user.admin_updated`   | `user`        | `is_admin`                                       | Admin toggle via `PUT /api/admin/users/{id}` |
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

**Notes:**
- `user_id` in an audit log entry is the ID of the user who performed the action. It is `null` for system-initiated actions (e.g. background job operations).
- Audit log entries are never modified or deleted. They represent a historical record.
- Book files created automatically by the background scanner do **not** currently produce an `audit_log` entry — only files created via the API are audited.

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

> **`organization_type`:** The only recognised value is `"book_per_folder"` (the default). This describes the expected directory layout: each book occupies its own `<Author>/<Title>/` subdirectory under a library path. Biblioteka uses this layout for [path-based metadata extraction](background-jobs.md#path-based-metadata) (deriving author, title, and series from directory names) and, when file reorganization is enabled, for moving imported files into this structure automatically (see [Enabling file organization](#enabling-file-organization) below).

### Edit and delete libraries

Use `PUT /api/libraries/{id}` to update a library and `DELETE /api/libraries/{id}` to remove it. Both operations require admin privileges. Deleting a library removes only the library record and its book associations — the underlying book, author, series, and book file records are not deleted.

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

The server immediately tests the issuer URL by performing OIDC discovery before saving. If discovery fails, the config is rejected with a `400` error.

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

All six SMTP fields are saved in a single database transaction. If the database write fails partway through, the entire update is rolled back and the previous configuration remains unchanged.

**TLS modes:** `none` (plaintext), `starttls` (STARTTLS upgrade on port 587, default), or `tls` (implicit TLS on port 465).

All six settings (`host`, `port`, `username`, `password`, `from`, `tls`) are saved atomically in a single database transaction. If the write fails, none of the settings are changed — the configuration is never left in a partially-updated state.

**Precedence:** When the `SMTP_HOST` environment variable is set, all SMTP settings are read exclusively from environment variables (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`) and the database values are ignored. The runtime configuration UI will appear read-only. When `SMTP_HOST` is unset (the default), the values stored in the database via the API or Settings UI are used.

See [API reference — SMTP config endpoints](api-reference.md#get-apiconfigsmtp--admin) for full request/response shapes.

---

## File Organization

Biblioteka can automatically move imported book files into a canonical `Author/Title/` directory structure under each library root. This keeps your collection tidy and makes paths predictable.

### Enabling file organization

> **Note:** There is currently no HTTP API endpoint for toggling `organize_files`. The setting is read directly from the `settings` database table. Enable it by inserting or updating the row directly:

**SQLite:**

```bash
sqlite3 /path/to/biblioteka.db \
  "INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('organize_files', 'true', datetime('now'));"
```

**PostgreSQL:**

```sql
INSERT INTO settings (key, value, updated_at)
VALUES ('organize_files', 'true', NOW())
ON CONFLICT (key) DO UPDATE SET value = 'true', updated_at = NOW();
```

To disable it, set the value to `'false'` (or any value other than `'true'`):

**SQLite:**

```bash
sqlite3 /path/to/biblioteka.db \
  "UPDATE settings SET value = 'false', updated_at = datetime('now') WHERE key = 'organize_files';"
```

**PostgreSQL:**

```sql
UPDATE settings SET value = 'false', updated_at = NOW() WHERE key = 'organize_files';
```

Changes take effect the next time a `process:file` job runs — no server restart is required.

### How it works

When `organize_files` is `"true"` and a `process:file` job has a `library_root` in its payload, the handler moves each imported file to:

```
<library_root>/<Author>/<Title>/<filename>
```

The author and title come from embedded file metadata when available, falling back to values parsed from the file's existing directory structure (see [Path-based metadata](background-jobs.md#path-based-metadata)).

**Behaviour details:**

- Directory names are sanitized: path separators (`/`, `\`), control characters, colons, wildcards, and leading dots are removed.
- The move uses `os.Rename` when source and destination are on the same filesystem. A copy-then-delete falls back for cross-filesystem moves; source file permissions and modification time are preserved.
- Empty source directories left behind after a move are removed automatically (up to but not including the library root).
- If a file already exists at the target path, the handler skips the move and logs a warning — it never silently overwrites existing files.
- If reorganization fails for any reason, the handler logs a warning and continues processing the file at its original path. The import still completes; only the file location is affected.

### Path-parsing and series inference

Even when `organize_files` is disabled, Biblioteka parses each file's path relative to the library root to extract author, title, and series from the directory structure. Trailing `(YYYY)` year tokens are stripped to keep titles clean (the year is not stored as `publication_date`). This path-derived metadata supplements (but does not override) embedded file metadata.

For full details on the supported directory layouts and precedence rules, see [Background Jobs — Path-based metadata](background-jobs.md#path-based-metadata).

### Sidecar files

Every time a book file is imported — regardless of whether file organization is enabled — Biblioteka writes two sidecar files in the same directory as the book file:

| File | Contents |
|------|----------|
| `cover.<ext>` | Cover image (JPEG, PNG, WebP, or AVIF) decoded from the embedded EPUB cover, when available. Skipped for non-EPUB formats and when no cover is present. |
| `metadata.opf` | OPF 2.0 Dublin Core metadata: title, author, identifier, language, publication date, publisher, and description. |

These writes are best-effort. Failures are logged at `WARN` level and do not prevent the book from being imported.

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
