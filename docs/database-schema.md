# Database Schema

Biblioteka uses [dbmate](https://github.com/amacneil/dbmate) migrations, which run automatically on server startup. Migrations are stored under `db/migrations/sqlite/` and `db/migrations/postgres/`. The schema is identical across both dialects except for primary key generation (SQLite uses `lower(hex(randomblob(16)))`; PostgreSQL uses `gen_random_uuid()`).

**ID format:** With SQLite, every `id` column is a 32-character lowercase hex string (e.g. `"f47ac10b58cc4372a567b409e2087bc1"`). With PostgreSQL, IDs are UUID strings (e.g. `"550e8400-e29b-41d4-a716-446655440000"`). Treat IDs as opaque strings — do not rely on their format.

---

## Entity Relationship Overview

```
users ────────────────────────────────────────────────┐
                                                       │ (audit trail)
settings                                               ▼
                                               audit_logs

libraries ──── library_books ──── books ──── book_authors ──── authors
                                    │
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

Key-value store for runtime configuration. Currently used for OIDC provider settings.

| Column       | Type    | Nullable | Default  | Description             |
|--------------|---------|----------|----------|-------------------------|
| `key`        | TEXT    | NOT NULL | —        | Primary key; setting name |
| `value`      | TEXT    | NOT NULL | —        | Setting value           |
| `updated_at` | DATETIME| NOT NULL | `now()`  | Last update time        |

**Known keys:**

| Key                    | Description                              |
|------------------------|------------------------------------------|
| `oidc_issuer_url`      | OIDC provider issuer URL                 |
| `oidc_client_id`       | OIDC application client ID               |
| `oidc_client_secret`   | OIDC application client secret           |
| `oidc_redirect_uri`    | OAuth2 redirect URI for the callback     |

**Notes:**
- Environment variables (`OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, etc.) take precedence over values stored in this table.
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

| Column           | Type    | Nullable | Default   | Description                   |
|------------------|---------|----------|-----------|-------------------------------|
| `id`             | TEXT    | NOT NULL | auto-gen  | Primary key                   |
| `name`           | TEXT    | NOT NULL | —         | Author display name (unique)  |
| `goodreads_id`   | TEXT    | NULL     | NULL      | Goodreads author ID           |
| `hardcover_id`   | TEXT    | NULL     | NULL      | Hardcover author ID           |
| `google_books_id`| TEXT    | NULL     | NULL      | Google Books author ID        |
| `image_url`      | TEXT    | NULL     | NULL      | URL to author photo           |
| `created_at`     | DATETIME| NOT NULL | `now()`   | Creation time                 |
| `updated_at`     | DATETIME| NOT NULL | `now()`   | Last update time              |

---

### `series`

Metadata about book series, shared across all libraries.

| Column           | Type    | Nullable | Default   | Description                   |
|------------------|---------|----------|-----------|-------------------------------|
| `id`             | TEXT    | NOT NULL | auto-gen  | Primary key                   |
| `name`           | TEXT    | NOT NULL | —         | Series name (unique)          |
| `goodreads_id`   | TEXT    | NULL     | NULL      | Goodreads series ID           |
| `hardcover_id`   | TEXT    | NULL     | NULL      | Hardcover series ID           |
| `google_books_id`| TEXT    | NULL     | NULL      | Google Books series ID        |
| `created_at`     | DATETIME| NOT NULL | `now()`   | Creation time                 |
| `updated_at`     | DATETIME| NOT NULL | `now()`   | Last update time              |

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

## Cascade Deletion Summary

| Deleted entity | Also deletes                                      |
|----------------|---------------------------------------------------|
| `users`        | `libraries` (previously scoped per user; no longer applies after migration `20260313010000`) |
| `libraries`    | `library_books` entries for that library          |
| `books`        | `book_files`, `book_authors`, `book_series`, `library_books` entries for that book |
| `authors`      | `book_authors` entries for that author            |
| `series`       | `book_series` entries for that series             |

---

## Running Migrations Manually

Migrations run automatically at server startup. To run them manually (e.g. when setting up a development environment without starting the server):

```bash
# SQLite (default)
DATABASE_URL=sqlite:./biblioteka.db go run ./cmd/server --migrate-only

# PostgreSQL
DATABASE_URL=postgres://biblioteka:secret@localhost/biblioteka go run ./cmd/server --migrate-only
```

The server uses [dbmate](https://github.com/amacneil/dbmate) format. Migration files follow the naming convention `YYYYMMDDHHMMSS_description.sql` and contain `-- migrate:up` / `-- migrate:down` sections.
