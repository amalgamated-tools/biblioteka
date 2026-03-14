# Biblioteka API Reference

All API responses use JSON. Error responses have the shape `{"error": "<message>"}`.

## Authentication

Most endpoints require a JWT bearer token obtained from the login or signup endpoints:

```
Authorization: Bearer <token>
```

Endpoints that require an **admin** role are noted with 🔒 Admin.

---

## Health

### `GET /api/health`

Returns the server health status. No authentication required.

**Response `200 OK`**
```json
{"status": "ok"}
```

---

## Auth

### `POST /api/auth/signup`

Create a new account. Rate-limited. The first account created is automatically granted admin privileges.

**Request body**
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "secret123"
}
```

**Response `201 Created`**
```json
{
  "token": "<jwt>",
  "user": {
    "id": "<uuid>",
    "email": "jane@example.com",
    "oidc_linked": false,
    "is_admin": true
  }
}
```

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing or invalid fields; password too short (minimum 6 characters) |
| `409` | Email already registered |

---

### `POST /api/auth/login`

Authenticate with email and password. Rate-limited.

**Request body**
```json
{"email": "jane@example.com", "password": "secret123"}
```

**Response `200 OK`** — same shape as signup.

**Errors**
| Status | Meaning |
|--------|---------|
| `401` | Invalid credentials; or account uses OIDC-only login |

---

### `GET /api/auth/me` 🔑

Return the currently authenticated user.

**Response `200 OK`**
```json
{
  "id": "<uuid>",
  "email": "jane@example.com",
  "oidc_linked": false,
  "is_admin": true
}
```

---

### `PUT /api/auth/password` 🔑

Change the authenticated user's password. Not available for OIDC-only accounts.

**Request body**
```json
{"currentPassword": "old", "newPassword": "new-secret"}
```

**Response `200 OK`**
```json
{"message": "password updated"}
```

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing fields; new password too short; OIDC-only account |
| `401` | Current password is incorrect |

---

## OIDC Single Sign-On

OIDC must be configured before these endpoints are functional (see [OIDC Configuration](#oidc-configuration) below).

### `GET /api/auth/oidc/enabled`

Check whether OIDC is configured. No authentication required.

**Response `200 OK`**
```json
{"enabled": true}
```

---

### `GET /api/auth/oidc/login`

Redirect the browser to the OIDC provider's authorization endpoint. Rate-limited.

Sets `oidc_state` and `oidc_verifier` cookies (PKCE) and issues an HTTP `302` redirect.

**Response `302 Found`** — redirect to OIDC provider.

---

### `GET /api/auth/oidc/callback`

Handle the OIDC provider's redirect after successful authentication. Rate-limited.

On success, redirects to `/?token=<jwt>` for the frontend to consume.

On error, redirects to `/?oidc_link_error=<message>` (link flow only).

---

### `POST /api/auth/oidc/link-nonce` 🔑

Generate a short-lived, single-use nonce to initiate OIDC account linking for the authenticated user.

The nonce expires after 5 minutes.

**Response `200 OK`**
```json
{"nonce": "<nonce>"}
```

---

### `GET /api/auth/oidc/link`

Start the OIDC account-linking flow for an existing local account. Rate-limited.

Pass the nonce from `link-nonce` as a query parameter:

```
GET /api/auth/oidc/link?nonce=<nonce>
```

On success, redirects to `/?oidc_linked=true`.

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing nonce |
| `401` | Invalid or expired nonce |
| `409` | Account is already linked to an SSO provider |

---

## Configuration

### `GET /api/config/status` 🔑

Return whether OIDC is configured and whether the caller is an admin.

**Response `200 OK`**
```json
{"oidc_configured": false, "is_admin": true}
```

---

### `GET /api/config/oidc` 🔑 🔒 Admin

Return the current OIDC configuration. The `client_secret` value is never returned; `client_secret_set` indicates whether one is stored.

**Response `200 OK`**
```json
{
  "issuer_url": "https://accounts.example.com",
  "client_id": "my-client",
  "client_secret_set": true,
  "redirect_uri": "https://biblioteka.example.com/api/auth/oidc/callback"
}
```

---

### `PUT /api/config/oidc` 🔑 🔒 Admin

Save OIDC configuration and apply it immediately (without server restart).

If `OIDC_ISSUER_URL` is set as an environment variable, the environment variable takes precedence over this stored configuration and a warning is included in the response message.

**Request body**
```json
{
  "issuer_url": "https://accounts.example.com",
  "client_id": "my-client",
  "client_secret": "secret",
  "redirect_uri": "https://biblioteka.example.com/api/auth/oidc/callback"
}
```

`client_secret` may be omitted to preserve the existing stored secret.

**Response `200 OK`**
```json
{"message": "OIDC configuration saved successfully"}
```

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing required fields; OIDC provider discovery failed (invalid `issuer_url`) |
| `403` | Admin access required |

---

## Admin

### `GET /api/admin/users` 🔑 🔒 Admin

List all registered users.

**Response `200 OK`**
```json
[
  {
    "id": "<uuid>",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "is_admin": true,
    "oidc_linked": false,
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `PUT /api/admin/users/{id}` 🔑 🔒 Admin

Grant or revoke admin privileges for a user. An admin cannot change their own status.

**Request body**
```json
{"is_admin": true}
```

**Response `200 OK`**
```json
{"message": "admin status updated"}
```

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Attempting to change own admin status |
| `404` | User not found |

---

## Libraries

### `GET /api/libraries` 🔑

List all libraries for the authenticated user.

**Response `200 OK`**
```json
[
  {
    "id": "<uuid>",
    "name": "Fiction",
    "paths": ["/books/fiction"],
    "organization_type": "book_per_folder",
    "monitored": true,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `POST /api/libraries` 🔑

Create a new library.

**Request body**
```json
{
  "name": "Fiction",
  "paths": ["/books/fiction", "/books/sci-fi"],
  "organization_type": "book_per_folder",
  "monitored": true
}
```

`name` and at least one entry in `paths` are required. Each path must point to an existing directory on the server filesystem. `organization_type` defaults to `"book_per_folder"` when omitted.

> **Note:** Creating a library automatically enqueues a background scan job for each path. Files found at those paths are indexed asynchronously.

**Response `201 Created`** — Library object.

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing name or paths; a path is empty or does not exist as a directory on the server |
| `409` | A library with that name already exists |

---

### `GET /api/libraries/{id}` 🔑

Get a single library by ID.

**Response `200 OK`** — Library object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Library not found |

---

### `PUT /api/libraries/{id}` 🔑

Update a library.

**Request body** — same shape as POST.

**Response `200 OK`** — Updated library object.

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing name or paths; a path is empty or does not exist as a directory on the server |
| `404` | Library not found |
| `409` | A library with that name already exists |

---

### `DELETE /api/libraries/{id}` 🔑

Delete a library.

**Response `204 No Content`**

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Library not found |

---

## Authors

### `GET /api/authors` 🔑

List all authors.

**Response `200 OK`**
```json
[
  {
    "id": "<uuid>",
    "name": "Jane Austen",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "image_url": null,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `POST /api/authors` 🔑

Create an author.

**Request body**
```json
{
  "name": "Jane Austen",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null,
  "image_url": null
}
```

**Response `201 Created`** — Author object.

---

### `GET /api/authors/{id}` 🔑

Get a single author by ID.

**Response `200 OK`** — Author object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Author not found |

---

### `PUT /api/authors/{id}` 🔑

Update an author.

**Request body** — same shape as POST.

**Response `200 OK`** — Updated author object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Author not found |

---

### `DELETE /api/authors/{id}` 🔑

Delete an author.

**Response `204 No Content`**

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Author not found |

---

## Series

### `GET /api/series` 🔑

List all series.

**Response `200 OK`**
```json
[
  {
    "id": "<uuid>",
    "name": "Wheel of Time",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `POST /api/series` 🔑

Create a series.

**Request body**
```json
{
  "name": "Wheel of Time",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null
}
```

**Response `201 Created`** — Series object.

---

### `GET /api/series/{id}` 🔑

Get a single series by ID.

**Response `200 OK`** — Series object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Series not found |

---

### `PUT /api/series/{id}` 🔑

Update a series.

**Request body** — same shape as POST.

**Response `200 OK`** — Updated series object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Series not found |

---

### `DELETE /api/series/{id}` 🔑

Delete a series.

**Response `204 No Content`**

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Series not found |

---

## Books

### `GET /api/books` 🔑

List all books. Returns a summary representation (without embedded authors, series, or files).

**Response `200 OK`**
```json
[
  {
    "id": "<uuid>",
    "title": "Pride and Prejudice",
    "description": null,
    "asin": null,
    "isbn10": null,
    "isbn13": "978-0-14-143951-8",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "publication_date": "1813-01-28",
    "publisher": "Penguin Classics",
    "language": "en",
    "num_pages": 432,
    "cover_image_url": null,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `POST /api/books` 🔑

Create a book.

**Request body**
```json
{
  "title": "Pride and Prejudice",
  "description": null,
  "asin": null,
  "isbn10": null,
  "isbn13": "978-0-14-143951-8",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null,
  "publication_date": "1813-01-28",
  "publisher": "Penguin Classics",
  "language": "en",
  "num_pages": 432,
  "cover_image_url": null
}
```

`title` is required. All other fields are optional.

**Response `201 Created`** — Full book object (includes `authors`, `series`, and `files` arrays).

---

### `GET /api/books/{id}` 🔑

Get a single book with its associated authors, series entries, and files.

**Response `200 OK`** — Full book object.

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Book not found |

---

### `PUT /api/books/{id}` 🔑

Update a book.

**Request body** — same shape as POST. `title` is required.

**Response `200 OK`** — Updated full book object.

**Errors**
| Status | Meaning |
|--------|---------|
| `400` | Missing `title` |
| `404` | Book not found |

---

### `DELETE /api/books/{id}` 🔑

Delete a book and its associated book-file records.

**Response `204 No Content`**

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Book not found |

---

### `GET /api/books/{id}/authors` 🔑

List the authors associated with a book.

**Response `200 OK`** — Array of author objects.

---

### `PUT /api/books/{id}/authors` 🔑

Replace the full set of authors for a book.

**Request body**
```json
{"author_ids": ["<uuid>", "<uuid>"]}
```

**Response `200 OK`** — Updated array of author objects.

---

### `GET /api/books/{id}/series` 🔑

List the series entries for a book.

**Response `200 OK`**
```json
[
  {
    "series": { "id": "<uuid>", "name": "Wheel of Time", ... },
    "position": 1.0
  }
]
```

---

### `PUT /api/books/{id}/series` 🔑

Replace the full set of series entries for a book.

**Request body**
```json
{
  "entries": [
    {"series_id": "<uuid>", "position": 1.0}
  ]
}
```

**Response `200 OK`** — Updated array of book series entries.

---

### `GET /api/books/{id}/files` 🔑

List all file records associated with a book.

**Response `200 OK`** — Array of book-file objects.

---

### `POST /api/books/{id}/files` 🔑

Register a new file record for a book.

**Request body**
```json
{
  "file_type": "epub",
  "file_name": "pride-and-prejudice.epub",
  "file_size": 524288,
  "file_hash": "sha256:abc123...",
  "file_path": "/books/fiction/pride-and-prejudice.epub"
}
```

`file_type`, `file_name`, and `file_path` are required. `file_hash` is optional.

**Response `201 Created`** — Book-file object.

---

## Book Files

### `GET /api/book-files/{id}` 🔑

Get a single book-file record by its own ID.

**Response `200 OK`**
```json
{
  "id": "<uuid>",
  "book_id": "<uuid>",
  "file_type": "epub",
  "file_name": "pride-and-prejudice.epub",
  "file_size": 524288,
  "file_hash": "sha256:abc123...",
  "file_path": "/books/fiction/pride-and-prejudice.epub",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Book file not found |

---

### `DELETE /api/book-files/{id}` 🔑

Delete a book-file record by its own ID.

**Response `204 No Content`**

**Errors**
| Status | Meaning |
|--------|---------|
| `404` | Book file not found |

---

## Conventions

### Authentication legend

| Symbol | Meaning |
|--------|---------|
| 🔑 | Requires `Authorization: Bearer <jwt>` header |
| 🔒 Admin | Caller must have admin privileges |

### Error response shape

All error responses use HTTP `4xx`/`5xx` status codes and the following body:

```json
{"error": "<human-readable message>"}
```

### IDs

All resource IDs are random lowercase hex strings (16 bytes, 32 hex characters), generated by the database. They are URL-safe and can be used directly in path segments.

### Timestamps

All `created_at` / `updated_at` fields are ISO 8601 UTC strings (e.g. `"2026-01-01T00:00:00Z"`).
