<!-- disable-agentic-editing: true -->

# Database Schema

Biblioteka uses [dbmate](https://github.com/amacneil/dbmate) migrations, which run automatically on server startup. Migrations are stored under `db/migrations/sqlite/` and `db/migrations/postgres/`. The schema is identical across both dialects except for primary key generation (SQLite uses `lower(hex(randomblob(16)))`; PostgreSQL uses `gen_random_uuid()`).

**ID format:** With SQLite, every `id` column is a 32-character lowercase hex string (e.g. `"f47ac10b58cc4372a567b409e2087bc1"`). With PostgreSQL, IDs are UUID strings (e.g. `"550e8400-e29b-41d4-a716-446655440000"`). Treat IDs as opaque strings — do not rely on their format.

---

## Access model

Biblioteka uses a **shared catalog** model. Books, libraries, authors, and series are global resources visible to every authenticated user. Reading lists and reading progress are per-user private entities — each user sees and manages only their own.

| Entity | Scope | Notes |
|--------|-------|-------|
| `books`, `authors`, `series`, `book_files` | Global — all users | Any authenticated user can read, create, update, and delete |
| `libraries` | Global — all users | Paths scanned and books indexed are shared |
| `api_keys` | Per-user | Each key is owned by the user who created it; scoped to that user's permissions |
| `opds_credentials` | Per-user | One credential set per user; used only for OPDS Basic Auth |
| `kobo_tokens` | Per-user | One or more sync tokens per user; each token authenticates a single Kobo device |
| `kobo_reading_states` | Per-user | Reading progress reported by Kobo devices; one record per user–book pair |
| `reading_lists`, `reading_list_books` | Per-user | User-curated ordered lists of books; one user cannot see another's lists |
| `reading_progress` | Per-user | KOReader sync progress; one record per user–document pair |
| `audit_logs` | Global record — admin read | Logs record which user performed each action |

---

## Entity Relationship Overview

```
users ────────────────────────────────────────────────┐
  │                                                    │ (audit trail)
  ├──── api_keys                  settings             ▼
  ├──── opds_credentials                       audit_logs
  ├──── kobo_tokens
  ├──── goodreads_metadata
  ├──── reading_progress
  ├──── reading_lists ──── reading_list_books ───┐
  └──── kobo_reading_states ──────────────── books ──── book_authors ──── authors
                                               │
libraries ──── library_books ─────────────────┤
                                               ├──── book_series ──── series
                                               │
                                               └──── book_files
```

---

## Tables

### `users`

Stores registered user accounts.

| Column          | Type    | Nullable | Default     | Description                                      |
|-----------------|---------|----------|-------------|--------------------------------------------------|
| `id`            | TEXT    | NOT NULL | auto-gen    | Primary key                                      |
| `name`          | TEXT    | NOT NULL | —           | Display name                                     |
| `email`         | TEXT    | NOT NULL | —           | Email address (unique, case-insensitive)          |
| `password_hash` | TEXT    | NOT NULL | —           | bcrypt hash; set to a placeholder for OIDC-only users |
| `oidc_subject`  | TEXT    | NULL     | NULL        | OIDC `sub` claim; `NULL` if no SSO provider linked |
| `is_admin`      | INTEGER | NOT NULL | `0`         | `1` = admin, `0` = regular user                 |
| `created_at`    | DATETIME| NOT NULL | `now()`     | Account creation time                            |

**Indexes:**
- `UNIQUE(email)` — case-insensitive
- `UNIQUE(oidc_subject)` WHERE NOT NULL — each OIDC identity maps to at most one account

**Notes:**
- The first user to sign up is automatically granted `is_admin = 1`.
- OIDC-only users (created via SSO login without a prior password account) have a non-empty `password_hash` that cannot match any real password.

---

### `settings`

Key-value store for runtime configuration. Used for OIDC provider settings and SMTP mail configuration.

| Column       | Type    | Nullable | Default  | Description             |
|--------------|---------|----------|----------|-------------------------|
| `key`        | TEXT    | NOT NULL | —        | Primary key; setting name |
| `value`      | TEXT    | NOT NULL | —        | Setting value           |
| `updated_at` | DATETIME| NOT NULL | `now()`  | Last update time        |

**Known keys:**

*OIDC settings* (see [Authentication guide](authentication.md)):


| Key                    | Description                              |
|------------------------|------------------------------------------|
| `oidc_issuer_url`      | OIDC provider issuer URL                 |
| `oidc_client_id`       | OIDC application client ID               |
| `oidc_client_secret`   | OIDC application client secret           |
| `oidc_redirect_uri`    | OAuth2 redirect URI for the callback     |

*SMTP settings* (see [API reference — `GET /api/config/smtp`](api/config.md#get-apiconfigsmtp--admin--jwt-only)):

| Key               | Description                                                      |
|-------------------|------------------------------------------------------------------|
| `smtp_host`       | SMTP server hostname or IP address                               |
| `smtp_port`       | SMTP server port (defaults to `587` when absent)                 |
| `smtp_username`   | SMTP authentication username; empty for unauthenticated relay    |
| `smtp_password`   | SMTP authentication password (stored as plaintext; never returned by the API) |
| `smtp_from`       | Envelope `From` address for outgoing mail                        |
| `smtp_tls`        | TLS mode: `none`, `starttls` (default), or `tls`                 |

*Application settings:*

(No application-level keys remain — file organization is now configured per-library via the `organization_type` column on the `libraries` table.)

**Notes:**
- Environment variables (`OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `SMTP_HOST`, etc.) take precedence over values stored in this table.
- When `SMTP_HOST` is set as an environment variable, **all** SMTP settings are read from environment variables and any database-stored SMTP values are ignored.
- This table is intentionally minimal — it is not a general-purpose configuration store.

---

### `libraries`

Named collections of filesystem paths to scan for book files.

| Column              | Type    | Nullable | Default            | Description                                      |
|---------------------|---------|----------|--------------------|--------------------------------------------------|
| `id`                | TEXT    | NOT NULL | auto-gen           | Primary key                                      |
| `name`              | TEXT    | NOT NULL | —                  | Library name (unique)                            |
| `paths`             | TEXT    | NOT NULL | `'[]'`             | JSON array of absolute filesystem paths          |
| `organization_type` | TEXT    | NOT NULL | `'book_per_folder'`| How the library is organised (see below)         |
| `monitored`         | INTEGER | NOT NULL | `0`                | `1` = included in scheduled scans               |
| `created_at`        | DATETIME| NOT NULL | `now()`            | Creation time                                    |
| `updated_at`        | DATETIME| NOT NULL | `now()`            | Last update time                                 |

**`organization_type` values:**

| Value             | Meaning                                                   |
|-------------------|-----------------------------------------------------------|
| `book_per_folder` | Files are moved into `Author/Title/file` (default)        |
| `book_per_file`   | Files are moved into `Author/file` (flat per-author)      |
| `none`            | Files are left where they are                             |

**Notes:**
- `paths` is stored as a JSON array string (e.g. `'["/mnt/books", "/mnt/audiobooks"]'`).
- Libraries are global (not scoped per user). All authenticated users can read library metadata.
- Creating or updating a library with at least one path immediately enqueues a `scan:library` background job.

---

### `authors`

Metadata about book authors, shared across all libraries.

| Column           | Type    | Nullable | Default   | Description                                   |
|------------------|---------|----------|-----------|-----------------------------------------------|
| `id`             | TEXT    | NOT NULL | auto-gen  | Primary key                                   |
| `name`           | TEXT    | NOT NULL | —         | Author display name (unique, case-insensitive)|
| `goodreads_id`   | TEXT    | NULL     | NULL      | Goodreads author ID                           |
| `hardcover_id`   | TEXT    | NULL     | NULL      | Hardcover author ID                           |
| `google_books_id`| TEXT    | NULL     | NULL      | Google Books author ID                        |
| `image_url`      | TEXT    | NULL     | NULL      | URL to author photo                           |
| `created_at`     | DATETIME| NOT NULL | `now()`   | Creation time                                 |
| `updated_at`     | DATETIME| NOT NULL | `now()`   | Last update time                              |

**Indexes:**
- `UNIQUE(LOWER(name))` (`idx_authors_name_ci`) — case-insensitive uniqueness; `"Jane Austen"` and `"jane austen"` are treated as the same author

**Write normalization:** The application layer normalizes `name` before every insert or update: leading/trailing whitespace is trimmed and any internal whitespace run is collapsed to a single space. Capitalization is preserved. A name that is blank after normalization is rejected.

---

### `series`

Metadata about book series, shared across all libraries.

| Column           | Type    | Nullable | Default   | Description                   |
|------------------|---------|----------|-----------|-------------------------------|
| `id`             | TEXT    | NOT NULL | auto-gen  | Primary key                   |
| `name`           | TEXT    | NOT NULL | —         | Series name (unique, case-insensitive) |
| `goodreads_id`   | TEXT    | NULL     | NULL      | Goodreads series ID           |
| `hardcover_id`   | TEXT    | NULL     | NULL      | Hardcover series ID           |
| `google_books_id`| TEXT    | NULL     | NULL      | Google Books series ID        |
| `created_at`     | DATETIME| NOT NULL | `now()`   | Creation time                 |
| `updated_at`     | DATETIME| NOT NULL | `now()`   | Last update time              |

**Indexes:**
- `UNIQUE(LOWER(name))` (`idx_series_name_ci`) — case-insensitive uniqueness; `"The Chronicles of Narnia"` and `"the chronicles of narnia"` are treated as the same series. Series records created by `process:file` (via path parsing) reuse existing records rather than creating duplicates.

---

### `books`

Core book metadata. All fields except `title` are optional.

| Column            | Type    | Nullable | Default  | Description                            |
|-------------------|---------|----------|----------|----------------------------------------|
| `id`              | TEXT    | NOT NULL | auto-gen | Primary key                            |
| `title`           | TEXT    | NOT NULL | —        | Book title                             |
| `description`     | TEXT    | NULL     | NULL     | Synopsis or blurb                      |
| `asin`            | TEXT    | NULL     | NULL     | Amazon ASIN                            |
| `isbn10`          | TEXT    | NULL     | NULL     | ISBN-10                                |
| `isbn13`          | TEXT    | NULL     | NULL     | ISBN-13                                |
| `goodreads_id`    | TEXT    | NULL     | NULL     | Goodreads book ID                      |
| `hardcover_id`    | TEXT    | NULL     | NULL     | Hardcover book ID                      |
| `google_books_id` | TEXT    | NULL     | NULL     | Google Books volume ID                 |
| `publication_date`| TEXT    | NULL     | NULL     | ISO 8601 date string (e.g. `2026-03-15`) |
| `publisher`       | TEXT    | NULL     | NULL     | Publisher name                         |
| `language`        | TEXT    | NULL     | NULL     | BCP 47 language tag (e.g. `"en"`)     |
| `cover_image_url` | TEXT    | NULL     | NULL     | URL to cover image                     |
| `created_at`      | DATETIME| NOT NULL | `now()`  | Creation time                          |
| `updated_at`      | DATETIME| NOT NULL | `now()`  | Last update time                       |

**Indexes:**
- `idx_books_title` — index on `(title)` (SQLite) / composite index on `(title, id)` (PostgreSQL), covering `ORDER BY title ASC, rowid ASC` on SQLite and `ORDER BY title ASC, id ASC` on PostgreSQL in paginated list and search queries; used by `ListBooksPaginated`, `ListBooksByAuthorPaginated`, `ListBooksBySeriesPaginated`, and `SearchBooks`.
- `idx_books_created_at_id` — composite index on `(created_at DESC, id DESC)` (SQLite and PostgreSQL), covering `ORDER BY created_at DESC, id DESC` used by `ListRecentBooks`; on SQLite, the `id` tiebreaker avoids the temp B-tree sort step, and on PostgreSQL the same `ORDER BY` is already matched by the composite index without an extra sort (migration `20260428205224`).
- `idx_books_updated_at_id` — composite index on `(updated_at, id)` for efficient cursor-based pagination; used by the Kobo library sync endpoint to order and page through books by modification time.

**Full-text / trigram search indexes (added in migration `20260412000000`):**

- **SQLite:** `books_fts` — FTS5 content virtual table backed by the `books` table (columns: `title`, `description`). Three `AFTER INSERT/DELETE/UPDATE` triggers keep the index in sync automatically. `SearchBooks` queries this index using prefix-matching phrases (e.g. `"found"*`); the `sanitizeFTS5Query` function in `fts_sanitize.go` converts the user's query into a safe FTS5 `MATCH` expression. Running `VACUUM` on the database while FTS5 is enabled can corrupt the implicit `rowid` mapping; rebuild the index afterward with `INSERT INTO books_fts(books_fts) VALUES ('rebuild')` if needed.
- **PostgreSQL:** `idx_books_title_trgm` and `idx_books_description_trgm` — GIN trigram indexes using the `pg_trgm` extension (enabled automatically by the migration). These accelerate the `ILIKE '%query%'` clause used by `SearchBooks` on PostgreSQL, converting what would otherwise be a full sequential scan into an index lookup. LIKE special characters in the query are escaped before matching.

**Notes:**
- Books are global (not scoped per user).
- When a book file is first discovered by the background scanner, a book record is created automatically with the filename (minus extension) as the `title`. Other fields can be filled in via the API.

---

### `library_books` (join table)

Associates books with libraries (many-to-many).

| Column       | Type    | Nullable | Default  | Description                          |
|--------------|---------|----------|----------|--------------------------------------|
| `library_id` | TEXT    | NOT NULL | —        | FK → `libraries.id` (CASCADE DELETE) |
| `book_id`    | TEXT    | NOT NULL | —        | FK → `books.id` (CASCADE DELETE)    |
| `created_at` | DATETIME| NOT NULL | `now()`  | When the book was added to the library |

**Primary key:** `(library_id, book_id)`

---

### `book_authors` (join table)

Associates books with authors (many-to-many).

| Column      | Type    | Nullable | Default  | Description                         |
|-------------|---------|----------|----------|-------------------------------------|
| `book_id`   | TEXT    | NOT NULL | —        | FK → `books.id` (CASCADE DELETE)   |
| `author_id` | TEXT    | NOT NULL | —        | FK → `authors.id` (CASCADE DELETE) |
| `created_at`| DATETIME| NOT NULL | `now()`  | When the association was created    |

**Primary key:** `(book_id, author_id)`

---

### `book_series` (join table)

Associates books with series entries, including optional position within the series.

| Column      | Type    | Nullable | Default  | Description                          |
|-------------|---------|----------|----------|--------------------------------------|
| `book_id`   | TEXT    | NOT NULL | —        | FK → `books.id` (CASCADE DELETE)    |
| `series_id` | TEXT    | NOT NULL | —        | FK → `series.id` (CASCADE DELETE)   |
| `position`  | REAL    | NULL     | NULL     | Position in series (e.g. `1`, `2.5`) |
| `created_at`| DATETIME| NOT NULL | `now()`  | When the association was created     |

**Primary key:** `(book_id, series_id)`

**Notes:**
- `position` is a floating-point value to support fractional positions (e.g. `0.5` for a prequel, `2.5` for a novella between books 2 and 3).

---

### `tags`

Free-form labels that can be applied to books to aid organization and discovery. Tag names are normalized (trimmed and collapsed to single spaces), original casing is preserved for storage/display, and uniqueness is enforced case-insensitively.

| Column      | Type     | Nullable | Default  | Description                                   |
|-------------|----------|----------|----------|-----------------------------------------------|
| `id`        | TEXT     | NOT NULL | auto-gen | Primary key                                   |
| `name`      | TEXT     | NOT NULL | —        | Display name; unique after normalization       |
| `created_at`| DATETIME | NOT NULL | `now()`  | When the tag was created                      |
| `updated_at`| DATETIME | NOT NULL | `now()`  | When the tag was last updated                 |

**Indexes:**
- `UNIQUE(LOWER(name))` (`idx_tags_name`) — case-insensitive uniqueness enforced at the database level

**Notes:**
- Names are normalized before storage (`NormalizeTagName`); duplicate normalized names are rejected with `ErrTagNameExists`.
- Tags are global (not per-user); any authenticated user can view and apply them.
- Deleting a tag cascades and removes all `book_tags` rows that reference it.
- See [API Reference — Tags](api/tags.md) for the REST API.

---

### `book_tags` (join table)

Associates books with tags. Each row links one book to one tag.

| Column    | Type | Nullable | Default | Description                                |
|-----------|------|----------|---------|--------------------------------------------|
| `book_id` | TEXT | NOT NULL | —       | FK → `books.id` ON DELETE CASCADE          |
| `tag_id`  | TEXT | NOT NULL | —       | FK → `tags.id` ON DELETE CASCADE           |

**Primary key:** `(book_id, tag_id)`

**Indexes:**
- `idx_book_tags_tag_id` — fast lookup of books for a given tag

**Notes:**
- Setting a book's tags replaces the entire set in a single transaction (DELETE then INSERT).
- Deleting a book cascades and removes its `book_tags` rows.
- Deleting a tag cascades and removes its `book_tags` rows.

---

### `book_files`

Individual physical files (EPUB, MOBI, PDF, AZW3) linked to a book record.

| Column      | Type    | Nullable | Default  | Description                                         |
|-------------|---------|----------|----------|-----------------------------------------------------|
| `id`        | TEXT    | NOT NULL | auto-gen | Primary key                                         |
| `book_id`   | TEXT    | NOT NULL | —        | FK → `books.id` (CASCADE DELETE)                   |
| `file_type` | TEXT    | NOT NULL | —        | Format identifier: `epub`, `mobi`, `pdf`, `azw3`   |
| `file_name` | TEXT    | NOT NULL | —        | Filename on disk (e.g. `"dune.epub"`)               |
| `file_size` | INTEGER | NOT NULL | —        | File size in bytes                                  |
| `file_hash` | TEXT    | NULL     | NULL     | Content hash (e.g. `"sha256:abc123…"`)             |
| `file_path`      | TEXT    | NOT NULL | —        | Absolute path to the file on the server filesystem        |
| `download_count` | INTEGER (SQLite) / BIGINT (PostgreSQL) | NOT NULL | `0`      | Number of times this file has been downloaded |
| `created_at`     | DATETIME| NOT NULL | `now()`  | Creation time                                             |
| `updated_at`     | DATETIME| NOT NULL | `now()`  | Last update time                                          |

**Notes:**
- Supported `file_type` values (matched by the scanner): `epub`, `mobi`, `pdf`, `azw3`.
- Deleting a `book_file` record does **not** delete the file from disk.
- A book can have multiple files of the same type (e.g. two different EPUB editions).

**Indexes:**
- `UNIQUE(file_path)` (`idx_book_files_file_path`) — each physical file path is indexed at most once. The `process:file` handler relies on this constraint to prevent duplicate `book_file` rows when a file is encountered again after Redis state is lost or when the same path is scanned from multiple library configurations.

---

### `books_fts` (SQLite virtual table)

> **SQLite only.** This virtual table does not exist in the PostgreSQL schema; full-text search on PostgreSQL is handled by the `pg_trgm` GIN indexes described above.

An FTS5 content table that provides full-text search over `books.title` and `books.description`. It is created by migration `20260412000000_add_books_fts.sql`.

```sql
CREATE VIRTUAL TABLE books_fts USING fts5(
    title,
    description,
    content=books,
    content_rowid=rowid
);
```

`content=books` means FTS5 reads the indexed text from the `books` table rather than storing a copy, avoiding data duplication. `content_rowid=rowid` ties each FTS entry to the corresponding `books` row via SQLite's implicit `rowid`.

**Sync triggers:**

Three triggers keep `books_fts` in sync with the `books` table automatically:

| Trigger | Event | Behavior |
|---------|-------|----------|
| `books_fts_ai` | `AFTER INSERT ON books` | Inserts a new FTS entry for the added row |
| `books_fts_ad` | `AFTER DELETE ON books` | Marks the deleted row's FTS entry as deleted |
| `books_fts_au` | `AFTER UPDATE ON books WHEN title or description changed` | Deletes the old FTS entry and inserts a new one; skips updates that do not touch `title` or `description` |

**Query behavior:**

`SearchBooks` passes the user query through `sanitizeFTS5Query` before issuing the FTS `MATCH` expression. Each whitespace-separated token is wrapped in FTS5 phrase syntax with a trailing `*` wildcard for prefix matching (e.g. `"found"*` matches any word that starts with `found`, such as `foundation` — the match is case-insensitive because FTS5's `unicode61` tokenizer folds to lowercase). Multiple tokens are evaluated as an implicit AND — all tokens must match somewhere in the combined document (different tokens may match in different columns). Tokens that contain no letter or digit are skipped entirely; if no valid tokens remain, `SearchBooks` returns zero results without executing the FTS query.

**VACUUM warning:**

`books_fts` is tied to SQLite's implicit `rowid`, not the `books.id` TEXT primary key. Running `VACUUM` (or enabling `auto_vacuum`) can silently reassign rowids, corrupting the FTS index by mapping entries to wrong rows. By default SQLite does not enable `auto_vacuum`. If `VACUUM` is ever run manually or via a maintenance routine, rebuild the index afterwards:

```sql
INSERT INTO books_fts(books_fts) VALUES ('rebuild');
```

**Code:**
- `internal/db/fts_sanitize.go` — `sanitizeFTS5Query` and `containsWordChar` helpers
- `internal/db/book_queries.go` — `SearchBooks` (constructs the `MATCH` expression and queries `books_fts`)

---

### `api_keys`

Long-lived credentials for programmatic API access. Each key belongs to one user and inherits that user's permissions.

| Column         | Type    | Nullable | Default  | Description                                                       |
|----------------|---------|----------|----------|-------------------------------------------------------------------|
| `id`           | TEXT    | NOT NULL | auto-gen | Primary key                                                       |
| `user_id`      | TEXT    | NOT NULL | —        | FK → `users.id` (CASCADE DELETE)                                 |
| `name`         | TEXT    | NOT NULL | —        | Human-readable label chosen at creation (e.g. `"CI Pipeline"`)   |
| `key_hash`     | TEXT    | NOT NULL | —        | SHA-256 hash of the full key; the plaintext key is never stored  |
| `key_prefix`   | TEXT    | NOT NULL | —        | `bib_` prefix + first 12 hex chars of the key (e.g. `bib_a3f2c8e1d074`), stored in plaintext for UI display |
| `last_used_at` | DATETIME| NULL     | NULL     | Lazily updated (at most once per 5 minutes) when the key is used |
| `created_at`   | DATETIME| NOT NULL | `now()`  | Creation time                                                     |

**Indexes:**
- `UNIQUE(key_hash)` — fast constant-time lookup on each request
- `idx_api_keys_user_created_at` on `(user_id, created_at DESC, id DESC)` — list all keys for a user, sorted by newest first, without a temp sort pass

**Notes:**
- The full API key (`bib_` + 40 hex chars) is shown **once** at creation and is not recoverable afterward. The `key_hash` persists for lookup, and the non-secret `key_prefix` persists for UI identification.
- When a user is deleted, all their API keys are deleted via CASCADE.
- See the [Authentication guide — API Keys](authentication.md#api-keys) for usage details.

---

### `passkey_credentials`

Stores registered WebAuthn passkey credentials for a user. Each credential corresponds to a physical authenticator (hardware key, platform biometric, etc.). A user may register multiple passkeys.

| Column            | Type     | Nullable | Default  | Description                                                                  |
|-------------------|----------|----------|----------|------------------------------------------------------------------------------|
| `id`              | TEXT     | NOT NULL | auto-gen | Primary key                                                                  |
| `user_id`         | TEXT     | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE                                            |
| `name`            | TEXT     | NOT NULL | —        | User-chosen display label for the key (e.g. `"YubiKey 5"`)                  |
| `credential_id`   | TEXT     | NOT NULL | —        | Unique WebAuthn credential ID (base64url-encoded raw bytes); UNIQUE          |
| `credential_data` | TEXT     | NOT NULL | —        | Serialized `webauthn.Credential` JSON blob used during authentication        |
| `aaguid`          | TEXT     | NOT NULL | `''`     | Authenticator Attestation GUID; identifies the authenticator model           |
| `created_at`      | DATETIME | NOT NULL | `now()`  | When the credential was registered                                           |

**Indexes:**
- `UNIQUE(credential_id)` (`idx_passkey_credentials_credential_id`) — fast lookup during authentication assertions
- `idx_passkey_credentials_user_created_at` on `(user_id, created_at DESC, id DESC)` — list all credentials for a user, sorted by newest first, without a temp sort pass

**Notes:**
- `credential_data` is an opaque blob read and written exclusively by the `go-webauthn/webauthn` library; do not parse it directly.
- `credential_id` and `credential_data` are not exposed through the REST API; only `id`, `user_id`, `name`, `aaguid`, and `created_at` are returned to clients.
- When a user is deleted, all their passkey credentials are deleted via CASCADE.
- See [Authentication — Passkeys](authentication.md#passkeys-webauthn) for the user-facing feature overview.

---

### `passkey_challenges`

Temporary WebAuthn challenge sessions created during registration and authentication ceremonies. Each row is a short-lived blob that must be consumed (read and deleted atomically) before `expires_at`.

| Column         | Type     | Nullable | Default  | Description                                                    |
|----------------|----------|----------|----------|----------------------------------------------------------------|
| `id`           | TEXT     | NOT NULL | auto-gen | Primary key; also used as the challenge reference in the ceremony |
| `user_id`      | TEXT     | NULL     | NULL     | FK → `users.id` ON DELETE CASCADE; NULL for passwordless discovery flows |
| `session_data` | TEXT     | NOT NULL | —        | Serialized `webauthn.SessionData` JSON blob                    |
| `expires_at`   | DATETIME | NOT NULL | —        | Hard expiry; challenges older than this are invalid and are purged by a background job |
| `created_at`   | DATETIME | NOT NULL | `now()`  | When the challenge was created                                 |

**Indexes:**
- `idx_passkey_challenges_expires_at` — fast sweep of expired challenges by the cleanup job
- `idx_passkey_challenges_user_id` — fast lookup of in-progress challenge for a user

**Notes:**
- Challenges are consumed atomically via `GetAndDeletePasskeyChallenge` — reading a challenge also deletes it, preventing replay attacks.
- `user_id` is nullable to support discoverable-credential (usernameless) flows where the user identity is asserted by the authenticator, not the client.
- A background job (`DeleteExpiredPasskeyChallenges`) periodically purges rows where `expires_at` is in the past.

---

### `opds_credentials`

Per-user credentials used to authenticate OPDS reading apps via HTTP Basic Auth. Each user may have at most one set of OPDS credentials.

| Column          | Type    | Nullable | Default  | Description                                                      |
|-----------------|---------|----------|----------|------------------------------------------------------------------|
| `id`            | TEXT    | NOT NULL | auto-gen | Primary key                                                      |
| `user_id`       | TEXT    | NOT NULL | —        | FK → `users.id` (CASCADE DELETE); UNIQUE — one credential per user |
| `username`      | TEXT    | NOT NULL | —        | Case-insensitive OPDS username (UNIQUE)                          |
| `password_hash` | TEXT    | NOT NULL | —        | bcrypt hash of the OPDS password                                 |
| `created_at`    | DATETIME| NOT NULL | `now()`  | Creation time                                                    |
| `updated_at`    | DATETIME| NOT NULL | `now()`  | Last update time                                                 |

**Indexes:**
- `UNIQUE(user_id)` — one credential set per user
- `UNIQUE(username)` — case-insensitive; enforced via `COLLATE NOCASE`
- `idx_opds_credentials_username` — fast lookup during Basic Auth validation
- `idx_opds_credentials_user_id` — fast lookup by user

**Notes:**
- OPDS credentials are completely separate from the main account password and JWT-based authentication.
- When a user is deleted, their OPDS credentials are deleted via CASCADE.
- See the [OPDS Catalog guide](opds.md) for the full feature overview.

---

### `audit_logs`

Append-only record of create, update, and delete actions performed on entities.

| Column        | Type    | Nullable | Default  | Description                                             |
|---------------|---------|----------|----------|---------------------------------------------------------|
| `id`          | TEXT    | NOT NULL | auto-gen | Primary key                                             |
| `user_id`     | TEXT    | NULL     | NULL     | FK → `users.id` (no CASCADE); `NULL` for system actions |
| `action`      | TEXT    | NOT NULL | —        | Dot-separated action string (e.g. `"book.created"`)    |
| `entity_type` | TEXT    | NOT NULL | —        | Type of affected entity (e.g. `"book"`, `"library"`)   |
| `entity_id`   | TEXT    | NOT NULL | —        | ID of the affected entity                               |
| `metadata`    | TEXT    | NULL     | NULL     | JSON object with extra context (e.g. `{"title":"Dune"}`) |
| `created_at`  | DATETIME| NOT NULL | `now()`  | When the action occurred                                |

**Indexes:**
- `idx_audit_logs_created_at` — supports ordered pagination (newest first)
- `idx_audit_logs_entity` — fast lookup by `(entity_type, entity_id)`
- `idx_audit_logs_user_id` — fast lookup by `user_id`

**Notes:**
- This table is append-only. Rows are never updated or deleted.
- `user_id` is not a hard foreign key — if a user is deleted, their audit log entries are retained with the original `user_id` value.
- See the [Audit Logs API reference](api/admin.md#get-apiaudit-logs--admin) for the full list of `action` values.

---

### `kobo_tokens`

Named sync tokens that authenticate a Kobo e-reader device. Each token grants access to one user's library via the `/kobo/<token>/` device API.

| Column       | Type    | Nullable | Default  | Description                                              |
|--------------|---------|----------|----------|----------------------------------------------------------|
| `id`         | TEXT    | NOT NULL | auto-gen | Primary key                                              |
| `user_id`    | TEXT    | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE                        |
| `name`       | TEXT    | NOT NULL | —        | Human-readable label (max 100 chars)                    |
| `token_hash` | TEXT    | NOT NULL | —        | SHA-256 hex digest of the raw token                                                                |
| `created_at` | DATETIME| NOT NULL | `now()`  | When the token was created                               |

**Indexes:**
- `idx_kobo_tokens_user_created_at` on `(user_id, created_at DESC, id DESC)` — list all tokens for a user, sorted by newest first, without a temp sort pass
- `idx_kobo_tokens_token_hash` (unique) — fast lookup during device authentication

**Notes:**
- The raw token is a 32-byte cryptographically random value encoded as 64 hex characters.
- Only the SHA-256 hash is stored. If a token URL is lost, the user must delete it and create a new one.
- Deleting a user cascades and removes all their Kobo tokens.
- See [Kobo Tokens API](kobo.md#kobo-tokens-api) for management endpoints.

---

### `kobo_reading_states`

Tracks the reading progress a Kobo device has reported for each user–book pair. Updated each time the device syncs reading state.

| Column            | Type    | Nullable | Default       | Description                                          |
|-------------------|---------|----------|---------------|------------------------------------------------------|
| `id`              | TEXT    | NOT NULL | auto-gen      | Primary key                                          |
| `user_id`         | TEXT    | NOT NULL | —             | FK → `users.id` ON DELETE CASCADE                    |
| `book_id`         | TEXT    | NOT NULL | —             | FK → `books.id` ON DELETE CASCADE                    |
| `status`          | TEXT    | NOT NULL | `ReadyToRead` | `ReadyToRead`, `Reading`, or `Finished`              |
| `percent_read`    | REAL    | NULL     | NULL          | Reading progress as a fraction (0.0–1.0)             |
| `location_value`  | TEXT    | NULL     | NULL          | Kobo bookmark location value (CFI or similar)        |
| `location_type`   | TEXT    | NULL     | NULL          | Kobo bookmark location type                          |
| `location_source` | TEXT    | NULL     | NULL          | Kobo bookmark location source                        |
| `created_at`      | DATETIME| NOT NULL | `now()`       | When the reading state was first created             |
| `updated_at`      | DATETIME| NOT NULL | `now()`       | When the reading state was last updated              |

**Indexes:**
- `idx_kobo_reading_states_user_book` (unique) — enforces one state per user–book pair; used by the upsert
- `idx_kobo_reading_states_user_updated` — efficient time-range queries during library sync

**Notes:**
- The `(user_id, book_id)` pair is unique; updates use `INSERT … ON CONFLICT DO UPDATE`.
- Reading states are included in library sync responses when they were modified after the device's last sync token timestamp.
- Deleting a user or book cascades and removes their reading states.

---

### `kosync_credentials`

Stores KOReader [kosync](https://github.com/koreader/koreader-sync-server)-compatible credentials for each Biblioteka user. Each user may have at most one KOSync credential set. These credentials are used exclusively by the KOReader Progress sync plugin and are independent of the user's main Biblioteka login.

| Column          | Type     | Nullable | Default    | Description                                                      |
|-----------------|----------|----------|------------|------------------------------------------------------------------|
| `id`            | TEXT     | NOT NULL | auto-gen   | Primary key                                                      |
| `user_id`       | TEXT     | NOT NULL | —          | FK → `users.id` ON DELETE CASCADE (unique — one credential per user) |
| `username`      | TEXT     | NOT NULL | —          | KOSync username chosen by the user (case-insensitive, globally unique) |
| `password_hash` | TEXT     | NOT NULL | —          | `bcrypt(md5_hex(password))` — never the raw password             |
| `created_at`    | DATETIME | NOT NULL | `now()`    | When the credential was created                                  |
| `updated_at`    | DATETIME | NOT NULL | `now()`    | When the credential was last changed                             |

**Indexes:**
- `idx_kosync_credentials_username` (unique) — enforces globally unique usernames (case-insensitive)

**Notes:**
- The password is stored as `bcrypt(md5_hex(password))`. KOReader transmits the hex-encoded MD5 of the user's password as the `x-auth-key` header; by pre-hashing with MD5, Biblioteka can verify it directly against the stored bcrypt hash without storing the plaintext MD5.
- Deleting a user cascades and removes their KOSync credentials.

---

### `reading_progress`

Stores KOReader reading progress for each user–document pair. The `document` field is the opaque identifier KOReader generates from a book's file hash or path.

| Column       | Type     | Nullable | Default  | Description                                                      |
|--------------|----------|----------|----------|------------------------------------------------------------------|
| `id`         | TEXT     | NOT NULL | auto-gen | Primary key                                                      |
| `user_id`    | TEXT     | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE                                |
| `document`   | TEXT     | NOT NULL | —        | Opaque KOReader document identifier (file hash or path)          |
| `progress`   | TEXT     | NOT NULL | —        | KOReader position string (e.g. `"1/3/4/5/6/7/8"`)               |
| `percentage` | REAL     | NOT NULL | `0`      | Reading percentage in the range `[0, 1]`                         |
| `device`     | TEXT     | NULL     | NULL     | Name of the device that last updated this record (optional)      |
| `device_id`  | TEXT     | NULL     | NULL     | Identifier of the device that last updated this record (optional)|
| `created_at` | DATETIME | NOT NULL | `now()`  | When the progress record was first created                       |
| `updated_at` | DATETIME | NOT NULL | `now()`  | When the progress record was last updated                        |

**Indexes:**
- `idx_reading_progress_user_document` (unique) — enforces one record per user–document pair; used by the upsert
- `idx_reading_progress_user_updated_at` on `(user_id, updated_at DESC)` — fast user-scoped lookups ordered by recency; used by reading-activity queries and streak computation

**Notes:**
- The `(user_id, document)` pair is unique; updates use `INSERT … ON CONFLICT DO UPDATE`.
- Progress records are not linked to the `books` table — KOReader identifiers are opaque and may not correspond to a book in the library.
- Deleting a user cascades and removes their reading progress.

---

### `reading_lists`

Stores user-curated ordered lists of books. Each reading list is owned by a single user and is invisible to other users.

| Column        | Type     | Nullable | Default  | Description                                      |
|---------------|----------|----------|----------|--------------------------------------------------|
| `id`          | TEXT     | NOT NULL | auto-gen | Primary key                                      |
| `user_id`     | TEXT     | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE                |
| `name`        | TEXT     | NOT NULL | —        | Display name; normalized (trimmed, single spaces)|
| `description` | TEXT     | NULL     | NULL     | Optional free-text description                   |
| `created_at`  | DATETIME | NOT NULL | `now()`  | When the reading list was created                |
| `updated_at`  | DATETIME | NOT NULL | `now()`  | When the reading list was last updated           |

**Indexes:**
- `idx_reading_lists_user_name` (unique) — enforces one list per normalized name per user

**Notes:**
- Names are normalized before storage (`NormalizeReadingListName`); duplicate normalized names for the same user are rejected with `ErrReadingListNameExists`.
- `book_count` is computed on read via a `LEFT JOIN` to `reading_list_books` — it is not stored.
- Deleting a user cascades and removes their reading lists, which in turn cascade-deletes their `reading_list_books` entries.

---

### `reading_list_books` (join table)

Associates books with reading lists and tracks insertion time.

| Column            | Type     | Nullable | Default | Description                                   |
|-------------------|----------|----------|---------|-----------------------------------------------|
| `reading_list_id` | TEXT     | NOT NULL | —       | FK → `reading_lists.id` ON DELETE CASCADE     |
| `book_id`         | TEXT     | NOT NULL | —       | FK → `books.id` ON DELETE CASCADE             |
| `added_at`        | DATETIME | NOT NULL | `now()` | When the book was added to this list          |

**Primary key:** `(reading_list_id, book_id)`

**Indexes:**
- `idx_reading_list_books_book` — fast lookup of which lists contain a given book
- `idx_reading_list_books_list_added_at` on `(reading_list_id, added_at ASC, book_id ASC)` — supports the default `ORDER BY rlb.added_at ASC, b.id ASC` used by `ListReadingListBooks`; in SQLite this can avoid a temp B-tree sort, while in PostgreSQL it can help avoid or reduce an explicit sort step

**Notes:**
- `ADD` is idempotent: inserting a duplicate `(reading_list_id, book_id)` pair is silently ignored (`ON CONFLICT DO NOTHING`).
- `REMOVE` is also idempotent: removing a book that is not present returns `(false, nil)`.
- Deleting a book cascades and removes it from all reading lists.

---

### `reading_groups`

Collaborative reading groups. Each group has a single owner and any number of members. Groups provide a shared space for reading lists and book annotations.

| Column        | Type     | Nullable | Default  | Description                                                        |
|---------------|----------|----------|----------|--------------------------------------------------------------------|
| `id`          | TEXT     | NOT NULL | auto-gen | Primary key                                                        |
| `owner_id`    | TEXT     | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE; the user who created the group |
| `name`        | TEXT     | NOT NULL | —        | Group name; unique per owner after normalization                   |
| `description` | TEXT     | NULL     | NULL     | Optional free-text description                                     |
| `created_at`  | DATETIME | NOT NULL | `now()`  | When the group was created                                         |
| `updated_at`  | DATETIME | NOT NULL | `now()`  | When the group was last updated                                    |

**Indexes:**
- `UNIQUE(owner_id, name)` (`idx_reading_groups_owner_name`) — one group per normalized name per owner

**Notes:**
- Names are normalized before storage (`NormalizeGroupName`); duplicate normalized names for the same owner are rejected with `ErrGroupNameExists`.
- The `member_count` field is computed on read via a `LEFT JOIN` to `reading_group_members`; it is not stored.
- When a group is created, the owner is also inserted into `reading_group_members` with role `owner`; `owner_id` still records the owning user directly, and membership checks typically use `reading_group_members`.
- When a user is deleted, their groups are deleted via CASCADE; this cascades to `reading_group_members` and `reading_group_lists`.

---

### `reading_group_members` (join table)

Tracks membership of users in reading groups, including the member's role.

| Column      | Type     | Nullable | Default    | Description                                           |
|-------------|----------|----------|------------|-------------------------------------------------------|
| `group_id`  | TEXT     | NOT NULL | —          | FK → `reading_groups.id` ON DELETE CASCADE           |
| `user_id`   | TEXT     | NOT NULL | —          | FK → `users.id` ON DELETE CASCADE                    |
| `role`      | TEXT     | NOT NULL | `'member'` | Member role: `'owner'` or `'member'`                 |
| `joined_at` | DATETIME | NOT NULL | `now()`    | When the user joined the group                       |

**Primary key:** `(group_id, user_id)`

**Indexes:**
- `idx_reading_group_members_user` — fast lookup of all groups a user belongs to

**Notes:**
- Valid `role` values are `'owner'` and `'member'`; enforced by a `CHECK` constraint.
- The group owner cannot be removed from the group while they hold the `owner_id` reference on `reading_groups`.
- Deleting a group cascades and removes all its `reading_group_members` rows.
- Deleting a user cascades and removes their membership rows.

---

### `reading_group_lists` (join table)

Tracks which reading lists have been shared with which reading groups.

| Column       | Type     | Nullable | Default | Description                                              |
|--------------|----------|----------|---------|----------------------------------------------------------|
| `group_id`   | TEXT     | NOT NULL | —       | FK → `reading_groups.id` ON DELETE CASCADE              |
| `list_id`    | TEXT     | NOT NULL | —       | FK → `reading_lists.id` ON DELETE CASCADE               |
| `shared_by`  | TEXT     | NOT NULL | —       | FK → `users.id` ON DELETE CASCADE; user who shared it   |
| `shared_at`  | DATETIME | NOT NULL | `now()` | When the list was shared with the group                 |

**Primary key:** `(group_id, list_id)`

**Indexes:**
- `idx_reading_group_lists_list_id` — fast lookup of which groups a list is shared with
- `idx_reading_group_lists_shared_by` — fast lookup of lists shared by a specific user

**Notes:**
- Only the list's owner may share it with a group, and only if they are a member of that group.
- Sharing is idempotent (`ON CONFLICT DO NOTHING`).
- Deleting a group, list, or the sharing user cascades and removes the row.

---

### `book_annotations`

User annotations and highlights on books. Annotations may be private to the author or optionally shared with a reading group.

| Column       | Type     | Nullable | Default  | Description                                                                  |
|--------------|----------|----------|----------|------------------------------------------------------------------------------|
| `id`         | TEXT     | NOT NULL | auto-gen | Primary key                                                                  |
| `user_id`    | TEXT     | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE; annotation author                        |
| `book_id`    | TEXT     | NOT NULL | —        | FK → `books.id` ON DELETE CASCADE                                           |
| `text`       | TEXT     | NOT NULL | —        | Annotation body (highlight text or note)                                     |
| `cfi`        | TEXT     | NULL     | NULL     | EPUB CFI (Canonical Fragment Identifier) locating the passage in the file   |
| `group_id`   | TEXT     | NULL     | NULL     | FK → `reading_groups.id` ON DELETE SET NULL; non-NULL makes this a group annotation |
| `created_at` | DATETIME | NOT NULL | `now()`  | When the annotation was created                                              |
| `updated_at` | DATETIME | NOT NULL | `now()`  | When the annotation was last updated                                         |

**Indexes:**
- `idx_book_annotations_book_user` — composite index on `(book_id, user_id)` for listing a user's annotations for a given book
- `idx_book_annotations_group` — fast lookup of annotations shared with a given group

**Notes:**
- `cfi` is optional; annotations without a CFI are notes about the book rather than inline highlights.
- When `group_id` is set, the annotation is visible to all members of that group (the author must be a member).
- Deleting a book cascades and removes all its annotations.
- Deleting a user cascades and removes all their annotations.
- Deleting a group sets `group_id = NULL` on any annotations that referenced it (ON DELETE SET NULL), converting them to private annotations.

---

### `goodreads_metadata`

Stores Goodreads (and compatible catalog) metadata candidates fetched on behalf of a user. Each row is an imported snapshot of book metadata that can be reviewed and applied to a book record. The `status` field tracks whether the candidate has been accepted, rejected, or is awaiting review.

| Column                        | Type     | Nullable | Default    | Description                                                          |
|-------------------------------|----------|----------|------------|----------------------------------------------------------------------|
| `id`                          | TEXT     | NOT NULL | auto-gen   | Primary key                                                          |
| `user_id`                     | TEXT     | NOT NULL | —          | FK → `users.id` ON DELETE CASCADE                                    |
| `book_id`                     | TEXT     | NULL     | NULL       | FK → `books.id` ON DELETE SET NULL; the book this metadata targets   |
| `status`                      | TEXT     | NOT NULL | `'pending'`| Review status: `'pending'`, `'applied'`, or `'rejected'`             |
| `title`                       | TEXT     | NULL     | NULL       | Book title from the catalog                                          |
| `description`                 | TEXT     | NULL     | NULL       | Synopsis or blurb from the catalog                                   |
| `asin`                        | TEXT     | NULL     | NULL       | Amazon ASIN                                                          |
| `isbn10`                      | TEXT     | NULL     | NULL       | ISBN-10                                                              |
| `isbn13`                      | TEXT     | NULL     | NULL       | ISBN-13                                                              |
| `goodreads_id`                | TEXT     | NULL     | NULL       | Goodreads book ID                                                    |
| `hardcover_id`                | TEXT     | NULL     | NULL       | Hardcover book ID                                                    |
| `google_books_id`             | TEXT     | NULL     | NULL       | Google Books volume ID                                               |
| `publication_date`            | TEXT     | NULL     | NULL       | ISO 8601 publication date string                                     |
| `publisher`                   | TEXT     | NULL     | NULL       | Publisher name                                                       |
| `language`                    | TEXT     | NULL     | NULL       | BCP 47 language tag (e.g. `"en"`)                                   |
| `cover_image_url`             | TEXT     | NULL     | NULL       | URL to the cover image from the catalog                              |
| `author_name`                 | TEXT     | NULL     | NULL       | Primary author name from the catalog                                 |
| `author_goodreads_id`         | TEXT     | NULL     | NULL       | Goodreads author ID                                                  |
| `author_image_url`            | TEXT     | NULL     | NULL       | URL to the author photo from the catalog                             |
| `goodreads_work_id`           | TEXT     | NULL     | NULL       | Goodreads work ID                                                    |
| `goodreads_book_legacy_id`    | INTEGER  | NULL     | NULL       | Goodreads legacy integer book ID                                     |
| `goodreads_work_legacy_id`    | INTEGER  | NULL     | NULL       | Goodreads legacy integer work ID                                     |
| `goodreads_author_legacy_id`  | INTEGER  | NULL     | NULL       | Goodreads legacy integer author ID                                   |
| `created_at`                  | DATETIME | NOT NULL | `now()`    | Creation time                                                        |
| `updated_at`                  | DATETIME | NOT NULL | `now()`    | Last update time                                                     |

**Indexes:**
- `idx_goodreads_metadata_user_id` — fast user-scoped lookups
- `idx_goodreads_metadata_user_status_created_at_id_desc` — composite index on `(user_id, status, created_at DESC, id DESC)` for efficient paginated listing filtered by status
- `idx_goodreads_metadata_user_created_at_id_desc` — composite index on `(user_id, created_at DESC, id DESC)` for efficient paginated listing of all records for a user

**Notes:**
- **This table is internal-only and is not exposed through the REST API.** It is used exclusively by internal server-side logic and background jobs.
- Rows are scoped per user; every query filters by `user_id`.
- `book_id` is nullable: a metadata row may be imported before it has been linked to a book, or the linked book may be deleted (ON DELETE SET NULL keeps the metadata row for auditing).
- Status transitions: `pending` → `applied` (metadata accepted and written to the book) or `pending` → `rejected` (candidate discarded).

---

### `ai_enrichments`

AI-generated metadata suggestions for books, produced by a background enrichment job. Each row is a candidate set of metadata (suggested tags, reading level, generated description) for one book, awaiting user review.

| Column                  | Type     | Nullable | Default     | Description                                                                       |
|-------------------------|----------|----------|-------------|-----------------------------------------------------------------------------------|
| `id`                    | TEXT     | NOT NULL | auto-gen    | Primary key                                                                       |
| `user_id`               | TEXT     | NOT NULL | —           | FK → `users.id` ON DELETE CASCADE; the user who requested the enrichment         |
| `book_id`               | TEXT     | NULL     | NULL        | FK → `books.id` ON DELETE SET NULL; the target book                              |
| `status`                | TEXT     | NOT NULL | `'pending'` | Review status: `'pending'`, `'applied'`, or `'rejected'`; enforced by CHECK      |
| `provider`              | TEXT     | NOT NULL | —           | AI provider identifier (e.g. `"openai"`)                                         |
| `model`                 | TEXT     | NOT NULL | —           | Model name (e.g. `"gpt-4o-mini"`)                                                |
| `suggested_tags`        | TEXT     | NOT NULL | `'[]'`      | JSON array of suggested tag names                                                 |
| `reading_level`         | TEXT     | NULL     | NULL        | Optional reading level string (e.g. `"Young Adult"`)                             |
| `generated_description` | TEXT     | NULL     | NULL        | Optional AI-generated book synopsis                                               |
| `raw_response`          | TEXT     | NOT NULL | `''`        | Full raw API response from the provider (stored for audit/debugging)             |
| `created_at`            | DATETIME | NOT NULL | `now()`     | When the enrichment was created                                                   |
| `updated_at`            | DATETIME | NOT NULL | `now()`     | When the enrichment was last updated                                              |

**Indexes:**
- `idx_ai_enrichments_user_book_status` — composite index on `(user_id, book_id, status, created_at DESC)` for paginated listing filtered by status

**Notes:**
- `suggested_tags` is a JSON array of strings stored as TEXT; it is deserialized into `[]string` by `AIEnrichment.SuggestedTags` in Go.
- Status transitions: `pending` → `applied` (`ApplyAIEnrichment` writes the tags and/or description to the book and marks the row applied) or `pending` → `rejected` (candidate discarded without modifying the book).
- `ErrAIEnrichmentNotPending` is returned if an apply/reject is attempted on a non-pending row.
- `book_id` is nullable: the linked book may be deleted while the enrichment still exists (ON DELETE SET NULL).
- When a user is deleted, all their enrichment rows are deleted via CASCADE.

---

## Cascade Deletion Summary

| Deleted entity    | Also deletes                                      |
|-------------------|---------------------------------------------------|
| `users`           | `api_keys`, `passkey_credentials`, `passkey_challenges` (where user_id matches), `opds_credentials`, `kobo_tokens`, `kobo_reading_states`, `kosync_credentials`, `reading_progress`, `reading_lists` (which cascades to `reading_list_books`), `reading_groups` (owned by the user, which cascade to `reading_group_members` and `reading_group_lists`), `reading_group_members` (memberships in other groups), `book_annotations`, `goodreads_metadata`, `ai_enrichments` for that user |
| `libraries`       | `library_books` entries for that library          |
| `books`           | `book_files`, `book_authors`, `book_series`, `book_tags`, `book_annotations`, `library_books`, `kobo_reading_states`, `reading_list_books` entries for that book; sets `goodreads_metadata.book_id = NULL` and `ai_enrichments.book_id = NULL` for linked candidates |
| `authors`         | `book_authors` entries for that author            |
| `series`          | `book_series` entries for that series             |
| `tags`            | `book_tags` entries for that tag                  |
| `reading_lists`   | `reading_list_books` entries, `reading_group_lists` entries that share the list with a group |
| `reading_groups`  | `reading_group_members`, `reading_group_lists` entries for that group; sets `book_annotations.group_id = NULL` for annotations shared with the group |

---

## Code Layout

All database access lives in the `internal/db/` package. The books domain is split across several focused files; other entities each have their own file.

| File | Responsibility |
|------|----------------|
| `db.go` | `DB` struct definition, `Timestamp` custom type, dialect constants (`DialectSQLite`, `DialectPostgres`); `execAffected` internal helper that runs a write query and returns `sql.ErrNoRows` when zero rows are affected |
| `setup.go` | `SetupDatabase`: opens the correct backend (SQLite or PostgreSQL), applies PRAGMAs, and runs embedded migrations |
| `migrations.go` | Embedded migration runner used by `SetupDatabase` |
| `books.go` | `Book` struct; core CRUD: `CreateBook`, `CreateBookWithFile`, `GetBook`, `ListBooks`, `ListBooksByLibrary[Paginated]`, `UpdateBook`, `DeleteBook`, `AddBookToLibrary`, `RemoveBookFromLibrary` |
| `book_queries.go` | Additional book list/search queries: `ListBooksPaginated`, `ListRecentBooks`, `ListBooksByAuthor[Paginated]`, `ListBooksBySeries[Paginated]`, `SearchBooks` |
| `fts_sanitize.go` | `sanitizeFTS5Query` — converts a raw user search string into a safe SQLite FTS5 `MATCH` expression (prefix phrases, double-quote escaping, pure-punctuation token filtering); used exclusively by `SearchBooks` on the SQLite dialect |
| `book_relations.go` | Book–author and book–series associations: `GetBookAuthors`, `SetBookAuthors`, `GetBookSeries`, `SetBookSeries`, `GetAuthorsForBooks` |
| `book_files.go` | `BookFile` struct; file lifecycle: `CreateBookFile`, `GetBookFile`, `ListBookFiles`, `GetBookFileByPath`, `DeleteBookFile`, `GetFilesForBooks`, `IncrementBookFileDownloadCount` |
| `authors.go` | `Author` struct; `CreateAuthor`, `GetAuthor[ByName]`, `ListAuthors[Paginated]`, `UpdateAuthor`, `FindOrCreateAuthor`, `DeleteAuthor` |
| `series.go` | `Series` / `BookSeriesEntry` structs; `CreateSeries`, `GetSeries`, `GetSeriesByName`, `ListSeries[Paginated]`, `UpdateSeries`, `FindOrCreateSeries`, `DeleteSeries` |
| `libraries.go` | `Library` struct; `CreateLibrary`, `GetLibrary`, `ListLibraries`, `UpdateLibrary`, `DeleteLibrary` |
| `settings.go` | `Setting` struct; `GetSetting`, `SetSetting`, `SetSettings` (transactional multi-key save) |
| `users.go` | `User` struct; `CreateUser`, `CreateOIDCUser`, `GetUser*`, `LinkOIDCSubject`, `UpdatePassword`, `IsAdmin`, `SetAdmin`, `ListUsers` |
| `api_keys.go` | `APIKey` struct; `CreateAPIKey`, `ListAPIKeys`, `GetAPIKey`, `DeleteAPIKey`, `GetAPIKeyByHash`, `TouchAPIKeyLastUsed`, `ValidateAPIKey` |
| `passkeys.go` | `PasskeyCredential` / `PasskeyChallenge` structs; `CreatePasskeyCredential`, `GetPasskeyCredential`, `GetPasskeyCredentialByCredentialID`, `ListPasskeyCredentials`, `UpdatePasskeyCredentialData`, `DeletePasskeyCredential`, `CreatePasskeyChallenge`, `GetAndDeletePasskeyChallenge`, `DeleteExpiredPasskeyChallenges` |
| `protocol_credentials.go` | Shared base for per-protocol credential tables: `ProtocolCredential` struct, `protocolCredentialConfig` config type, and unexported helpers `getCredentialByUserID`, `getCredentialByUsername`, `upsertCredential`, `deleteCredential` — used by `opds_credentials.go` and `kosync.go` |
| `opds_credentials.go` | `OPDSCredential` (type alias for `ProtocolCredential`); `GetOPDSCredentialByUserID`, `GetOPDSCredentialByUsername`, `UpsertOPDSCredential`, `DeleteOPDSCredential` — thin wrappers around the shared helpers in `protocol_credentials.go` |
| `kobo_tokens.go` | `KoboToken` struct; `CreateKoboToken`, `GetKoboToken`, `GetKoboTokenByHash`, `ListKoboTokens`, `DeleteKoboToken` |
| `kobo_reading_states.go` | `KoboReadingState` struct; `GetKoboReadingState`, `UpsertKoboReadingState`, `ListKoboReadingStatesSince`, `GetReadingStatesForBooks` |
| `kosync.go` | `KOSyncCredential` (type alias for `ProtocolCredential`); `GetKOSyncCredentialByUserID`, `GetKOSyncCredentialByUsername`, `UpsertKOSyncCredential`, `DeleteKOSyncCredential` — thin wrappers around the shared helpers in `protocol_credentials.go`; `ReadingProgress` struct; `GetReadingProgress`, `UpsertReadingProgress` |
| `reading_progress.go` | Additional reading progress queries split from `kosync.go`: `ListReadingProgress`, `GetReadingStats`, `GetReadingStreak`, `ComputeReadingStreak` (computes a consecutive-day streak from a slice of timestamps relative to a caller-supplied reference time `now`; pass `time.Now().UTC()` in production, a fixed time in tests) |
| `reading_lists.go` | `ReadingList` struct; `CreateReadingList`, `GetReadingList`, `ListReadingLists`, `UpdateReadingList`, `DeleteReadingList`, `AddBookToReadingList`, `RemoveBookFromReadingList`, `ListReadingListBooks`, `GetReadingListsForBook` |
| `reading_groups.go` | `ReadingGroup` / `ReadingGroupMember` / `GroupMemberProgress` structs; `CreateGroup`, `GetGroup`, `ListGroups`, `UpdateGroup`, `DeleteGroup`, `ListGroupMembers`, `AddGroupMember`, `RemoveGroupMember`, `IsMember`, `ListGroupMemberProgress` |
| `reading_group_lists.go` | `ShareListWithGroup`, `UnshareListFromGroup`, `ListGroupReadingLists` |
| `book_annotations.go` | `BookAnnotation` struct; `CreateAnnotation`, `GetAnnotation`, `ListAnnotationsForBook`, `UpdateAnnotation`, `DeleteAnnotation` |
| `tags.go` | `Tag` struct; `CreateTag`, `GetTag`, `GetTagByName`, `ListTags`, `UpdateTag`, `FindOrCreateTag`, `DeleteTag`, `GetBookTags`, `SetBookTags` |
| `book_downloads.go` | `MonthlyDownloadCount` struct; `RecordBookDownload`, `GetMonthlyDownloads` |
| `book_load_relations.go` | `BookRelations` struct; `LoadBookRelations` — batch-fetches authors, files, and series for a single book by delegating to the existing batch APIs |
| `audit_logs.go` | `AuditLog` struct; `CreateAuditLog`, `ListAuditLogs` |
| `goodreads_metadata.go` | `GoodreadsMetadata` struct; `GoodreadsMetadataInput` struct (holds the 20 optional fields passed to `CreateGoodreadsMetadata` in place of positional arguments); `CreateGoodreadsMetadata`, `GetGoodreadsMetadata`, `ListGoodreadsMetadataByUser`, `ListGoodreadsMetadataByStatus`, `UpdateGoodreadsMetadataStatus`, `DeleteGoodreadsMetadata` |
| `ai_enrichments.go` | `AIEnrichment` / `ApplyAIEnrichmentInput` structs; `CreateAIEnrichment`, `GetAIEnrichment`, `GetPendingAIEnrichmentByBook`, `UpdateAIEnrichmentStatus`, `DeleteAIEnrichment`, `ApplyAIEnrichment` |
| `find_or_create.go` | Unexported generic helper `findOrCreate[T]` — implements the lookup → insert → race-fetch pattern shared by `FindOrCreateAuthor` and `FindOrCreateSeries`. Normalizes the name, validates it, attempts the insert, and falls back to a second lookup when a concurrent insert wins the unique-constraint race. |
| `named_entity_write.go` | Unexported generic helpers `namedEntityCreate[T]` and `namedEntityUpdate[T]` — normalize the name, validate it, execute the provided insert/update function, and translate unique-constraint violations into the entity-specific sentinel errors (`ErrXxxNameExists`). Used by `CreateAuthor`/`UpdateAuthor`, `CreateSeries`/`UpdateSeries`, `CreateReadingList`/`UpdateReadingList`, and `CreateTag`/`UpdateTag`. |
| `query_helpers.go` | Unexported query utilities: `dollarN(n)` — returns a PostgreSQL-style positional placeholder (`$n`); `buildInClause[T](values, startAt)` — builds positional placeholders and args for SQL `IN` clauses, shared by `books.go`, `book_relations.go`, `book_files.go`, and `kobo_reading_states.go`. |
| `scan_helpers.go` | Unexported generic scan utilities: `scanRow[T]` (wraps single-row scan to eliminate per-entity boilerplate), `collectRows[T]` (iterates `*sql.Rows` and collects results into a slice), and `collectRowsAndTotal[T]` (same as `collectRows` but also captures a `COUNT(*) OVER()` window-function total for paginated queries). |
| `tx.go` | Unexported transaction helper `deferRollback` — intended for use with `defer`; calls `tx.Rollback()`, silently ignores `sql.ErrTxDone`, and logs a warning for any other rollback error. |
| `paginate.go` | Two internal generic helpers sharing the `listQuery` interface and `allowedListTables` allowlist: `listAll[T]` — full-table SELECT with no limit; used by `ListAuthors`, `ListSeries`, `ListLibraries`; `listPaginated[T]` — issues a `COUNT(*)` then a paginated SELECT; used by `ListAuthorsPaginated` and `ListSeriesPaginated`. Both validate table names against the allowlist to prevent SQL injection |
| `sql_parser.go` | Internal helpers for parsing embedded SQL migration files |

> The `books.go` split (PR [#318](https://github.com/amalgamated-tools/biblioteka/pull/318)) separated a previously oversized `books.go` file into the four focused files above (`books.go`, `book_queries.go`, `book_relations.go`, `book_files.go`). The public API surface of the `*DB` receiver is unchanged.

---

## Running Migrations Manually

Migrations run automatically at server startup. To trigger them without starting the full HTTP server, start the server normally and stop it immediately after the migrations log output — or use the `dbmate` CLI directly against the same database:

```bash
# SQLite (default — database path mirrors the server's runtime path)
dbmate -u "sqlite:./db/biblioteka.db" up

# PostgreSQL
dbmate -u "postgres://biblioteka:secret@localhost/biblioteka" up
```

Install `dbmate` from https://github.com/amacneil/dbmate or via `mise install` if the project's `mise.toml` includes it.

The server uses its own built-in migration runner (not the `dbmate` binary), but both read the same `-- migrate:up` / `-- migrate:down` format. Migration files follow the naming convention `YYYYMMDDHHMMSS_description.sql`.
