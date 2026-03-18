# Database Schema

Biblioteka uses [dbmate](https://github.com/amacneil/dbmate) migrations, which run automatically on server startup. Migrations are stored under `db/migrations/sqlite/` and `db/migrations/postgres/`. The schema is identical across both dialects except for primary key generation (SQLite uses `lower(hex(randomblob(16)))`; PostgreSQL uses `gen_random_uuid()`).

**ID format:** With SQLite, every `id` column is a 32-character lowercase hex string (e.g. `"f47ac10b58cc4372a567b409e2087bc1"`). With PostgreSQL, IDs are UUID strings (e.g. `"550e8400-e29b-41d4-a716-446655440000"`). Treat IDs as opaque strings — do not rely on their format.

---

## Access model

Biblioteka uses a **shared catalog** model. Books, libraries, authors, and series are global resources visible to every authenticated user. There is no per-user book ownership or private reading list in the current implementation.

| Entity | Scope | Notes |
|--------|-------|-------|
| `books`, `authors`, `series`, `book_files` | Global — all users | Any authenticated user can read, create, update, and delete |
| `libraries` | Global — all users | Paths scanned and books indexed are shared |
| `api_keys` | Per-user | Each key is owned by the user who created it; scoped to that user's permissions |
| `opds_credentials` | Per-user | One credential set per user; used only for OPDS Basic Auth |
| `kobo_tokens` | Per-user | One or more sync tokens per user; each token authenticates a single Kobo device |
| `kobo_reading_states` | Per-user | Reading progress reported by Kobo devices; one record per user–book pair |
| `audit_logs` | Global record — admin read | Logs record which user performed each action |

> **Note:** The `MyLibrary` view in the frontend is a planned feature (currently a placeholder) that will eventually let each user maintain a personal reading list independent of the shared catalog.

---

## Entity Relationship Overview

```
users ────────────────────────────────────────────────┐
  │                                                    │ (audit trail)
  ├──── api_keys                  settings             ▼
  ├──── opds_credentials                       audit_logs
  ├──── kobo_tokens
  └──── kobo_reading_states ─────────────── books ──── book_authors ──── authors
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

Key-value store for runtime configuration. Used for OIDC provider settings, SMTP mail configuration, and application-level feature flags (e.g. `organize_files`).

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

*SMTP settings* (see [API reference — `GET /api/config/smtp`](api-reference.md#get-apiconfigsmtp--admin)):

| Key               | Description                                                      |
|-------------------|------------------------------------------------------------------|
| `smtp_host`       | SMTP server hostname or IP address                               |
| `smtp_port`       | SMTP server port (defaults to `587` when absent)                 |
| `smtp_username`   | SMTP authentication username; empty for unauthenticated relay    |
| `smtp_password`   | SMTP authentication password (stored as plaintext; never returned by the API) |
| `smtp_from`       | Envelope `From` address for outgoing mail                        |
| `smtp_tls`        | TLS mode: `none`, `starttls` (default), or `tls`                 |

*Application settings:*

| Key               | Description                                                                                          |
|-------------------|------------------------------------------------------------------------------------------------------|
| `organize_files`  | When `"true"`, the `process:file` job moves imported files into `<library_root>/<Author>/<Title>/`. No HTTP API endpoint exists yet — set directly in the database. See [File organization](administration.md#file-organization). |

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
| `book_per_folder` | Each immediate subdirectory is treated as one book (default) |

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
| `num_pages`       | INTEGER | NULL     | NULL     | Page count                             |
| `cover_image_url` | TEXT    | NULL     | NULL     | URL to cover image                     |
| `created_at`      | DATETIME| NOT NULL | `now()`  | Creation time                          |
| `updated_at`      | DATETIME| NOT NULL | `now()`  | Last update time                       |

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
| `file_path` | TEXT    | NOT NULL | —        | Absolute path to the file on the server filesystem  |
| `created_at`| DATETIME| NOT NULL | `now()`  | Creation time                                       |
| `updated_at`| DATETIME| NOT NULL | `now()`  | Last update time                                    |

**Notes:**
- Supported `file_type` values (matched by the scanner): `epub`, `mobi`, `pdf`, `azw3`.
- Deleting a `book_file` record does **not** delete the file from disk.
- A book can have multiple files of the same type (e.g. two different EPUB editions).

**Indexes:**
- `UNIQUE(file_path)` (`idx_book_files_file_path`) — each physical file path is indexed at most once. The `process:file` handler relies on this constraint to prevent duplicate `book_file` rows when a file is encountered again after Redis state is lost or when the same path is scanned from multiple library configurations.

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
- `idx_api_keys_user_id` — list all keys for a user

**Notes:**
- The full API key (`bib_` + 32 hex chars) is shown **once** at creation. Only the `key_hash` persists.
- When a user is deleted, all their API keys are deleted via CASCADE.
- See the [Authentication guide — API Keys](authentication.md#api-keys) for usage details.

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
- See the [Audit Logs API reference](api-reference.md#get-apiaudit-logs--admin) for the full list of `action` values.

---

### `kobo_tokens`

Named sync tokens that authenticate a Kobo e-reader device. Each token grants access to one user's library via the `/kobo/<token>/` device API.

| Column       | Type    | Nullable | Default  | Description                                              |
|--------------|---------|----------|----------|----------------------------------------------------------|
| `id`         | TEXT    | NOT NULL | auto-gen | Primary key                                              |
| `user_id`    | TEXT    | NOT NULL | —        | FK → `users.id` ON DELETE CASCADE                        |
| `name`       | TEXT    | NOT NULL | —        | Human-readable label (max 100 chars)                    |
| `token_hash` | TEXT    | NOT NULL | —        | SHA-256 hex digest of the raw token (raw token never stored) |
| `created_at` | DATETIME| NOT NULL | `now()`  | When the token was created                               |

**Indexes:**
- `idx_kobo_tokens_user_id` — list all tokens for a user
- `idx_kobo_tokens_hash` (unique) — fast lookup during device authentication

**Notes:**
- The raw token is a 32-byte cryptographically random value encoded as 64 hex characters.
- Only the SHA-256 hash is stored. If a token is lost, the user must delete it and create a new one.
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
- `idx_reading_progress_user_id` — fast user-scoped lookups

**Notes:**
- The `(user_id, document)` pair is unique; updates use `INSERT … ON CONFLICT DO UPDATE`.
- Progress records are not linked to the `books` table — KOReader identifiers are opaque and may not correspond to a book in the library.
- Deleting a user cascades and removes their reading progress.

---

## Cascade Deletion Summary

| Deleted entity | Also deletes                                      |
|----------------|---------------------------------------------------|
| `users`        | `api_keys`, `opds_credentials`, `kobo_tokens`, `kobo_reading_states`, `kosync_credentials`, `reading_progress` for that user |
| `libraries`    | `library_books` entries for that library          |
| `books`        | `book_files`, `book_authors`, `book_series`, `library_books`, `kobo_reading_states` entries for that book |
| `authors`      | `book_authors` entries for that author            |
| `series`       | `book_series` entries for that series             |

---

## Code Layout

All database access lives in the `internal/db/` package. The books domain is split across several focused files; other entities each have their own file.

| File | Responsibility |
|------|----------------|
| `db.go` | `DB` struct definition, `Timestamp` custom type, dialect constants (`DialectSQLite`, `DialectPostgres`) |
| `setup.go` | `SetupDatabase`: opens the correct backend (SQLite or PostgreSQL), applies PRAGMAs, and runs embedded migrations |
| `migrations.go` | Embedded migration runner used by `SetupDatabase` |
| `books.go` | `Book` struct; core CRUD: `CreateBook`, `CreateBookWithFile`, `GetBook`, `ListBooks`, `ListBooksByLibrary[Paginated]`, `UpdateBook`, `DeleteBook`, `AddBookToLibrary`, `RemoveBookFromLibrary` |
| `book_queries.go` | Additional book list/search queries: `ListBooksPaginated`, `ListRecentBooks`, `ListBooksByAuthor[Paginated]`, `ListBooksBySeries[Paginated]`, `SearchBooks` |
| `book_relations.go` | Book–author and book–series associations: `GetBookAuthors`, `SetBookAuthors`, `GetBookSeries`, `SetBookSeries`, `GetAuthorsForBooks` |
| `book_files.go` | `BookFile` struct; file lifecycle: `CreateBookFile`, `GetBookFile`, `ListBookFiles`, `GetBookFileByPath`, `DeleteBookFile`, `GetFilesForBooks` |
| `authors.go` | `Author` struct; `CreateAuthor`, `GetAuthor[ByName]`, `ListAuthors[Paginated]`, `UpdateAuthor`, `FindOrCreateAuthor`, `DeleteAuthor` |
| `series.go` | `Series` / `BookSeriesEntry` structs; `CreateSeries`, `GetSeries`, `ListSeries[Paginated]`, `UpdateSeries`, `FindOrCreateSeries`, `DeleteSeries` |
| `libraries.go` | `Library` struct; `CreateLibrary`, `GetLibrary`, `ListLibraries`, `UpdateLibrary`, `DeleteLibrary` |
| `settings.go` | `Setting` struct; `GetSetting`, `SetSetting`, `SetSettings` (transactional multi-key save) |
| `users.go` | `User` struct; `CreateUser`, `CreateOIDCUser`, `GetUser*`, `LinkOIDCSubject`, `UpdatePassword`, `IsAdmin`, `SetAdmin`, `ListUsers` |
| `api_keys.go` | `APIKey` struct; `CreateAPIKey`, `ListAPIKeys`, `GetAPIKey`, `DeleteAPIKey`, `GetAPIKeyByHash`, `TouchAPIKeyLastUsed`, `ValidateAPIKey` |
| `opds_credentials.go` | `OPDSCredential` struct; `GetOPDSCredential*`, `UpsertOPDSCredential`, `DeleteOPDSCredential` |
| `kobo_tokens.go` | `KoboToken` struct; `CreateKoboToken`, `GetKoboToken`, `GetKoboTokenByHash`, `ListKoboTokens`, `DeleteKoboToken` |
| `kobo_reading_states.go` | `KoboReadingState` struct; `GetKoboReadingState`, `UpsertKoboReadingState`, `ListKoboReadingStatesSince`, `GetReadingStatesForBooks` |
| `kosync.go` | `KOSyncCredential` struct; `GetKOSyncCredentialByUserID`, `GetKOSyncCredentialByUsername`, `UpsertKOSyncCredential`, `DeleteKOSyncCredential`; `ReadingProgress` struct; `GetReadingProgress`, `UpsertReadingProgress` |
| `audit_logs.go` | `AuditLog` struct; `CreateAuditLog`, `ListAuditLogs` |
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
