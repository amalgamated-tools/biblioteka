# API Reference

All endpoints are prefixed with `/api`. Unless noted as **public**, every endpoint requires a valid JWT bearer token:

```
Authorization: Bearer <token>
```

Tokens are obtained from the [Auth](#auth) endpoints. All request and response bodies are JSON (`Content-Type: application/json`).

---

## Auth

### POST /api/auth/signup

Create a new user account. Returns a JWT and the new user object.

**Request**

```json
{
  "name": "Jane Austen",
  "email": "jane@example.com",
  "password": "s3cr3t!"
}
```

`name`, `email`, and `password` are all required. The password must meet the application's minimum strength requirements.

**Response `201`**

```json
{
  "token": "<jwt>",
  "user": {
    "id": "uuid",
    "email": "jane@example.com",
    "oidc_linked": false,
    "is_admin": false
  }
}
```

---

### POST /api/auth/login

Exchange credentials for a JWT.

**Request**

```json
{
  "email": "jane@example.com",
  "password": "s3cr3t!"
}
```

**Response `200`** — same shape as signup.

---

### GET /api/auth/me

Return the authenticated user's profile.

**Response `200`**

```json
{
  "id": "uuid",
  "email": "jane@example.com",
  "oidc_linked": false,
  "is_admin": false
}
```

---

### PUT /api/auth/password

Change the authenticated user's password.

**Request**

```json
{
  "currentPassword": "s3cr3t!",
  "newPassword": "b3tterS3cr3t!"
}
```

**Response `204`** — no body.

---

## OIDC / SSO

### GET /api/auth/oidc/enabled *(public)*

Returns whether OIDC is configured on the server.

**Response `200`**

```json
{ "enabled": true }
```

---

### POST /api/auth/oidc/login *(public)*

Initiate an OIDC login flow. Redirects the client to the identity provider.

---

### GET /api/auth/oidc/callback *(public)*

OAuth 2.0 callback endpoint. Exchanges the authorization code for a JWT and redirects to the frontend.

---

### POST /api/auth/oidc/link *(public)*

Link an existing local account to an OIDC identity using a one-time nonce obtained from `/api/auth/oidc/link-nonce`.

---

### GET /api/auth/oidc/link-nonce *(requires auth)*

Generate a one-time nonce used to link an OIDC identity to the currently authenticated user.

**Response `200`**

```json
{ "nonce": "random-value" }
```

---

## Libraries

Libraries group one or more file-system paths that Biblioteka watches and scans for book files.

### GET /api/libraries

List all libraries for the authenticated user.

**Response `200`**

```json
[
  {
    "id": "uuid",
    "name": "My E-books",
    "paths": ["/mnt/books/epub", "/mnt/books/pdf"],
    "organization_type": "flat",
    "monitored": true,
    "created_at": "2026-03-13T00:00:00Z",
    "updated_at": "2026-03-13T00:00:00Z"
  }
]
```

---

### POST /api/libraries

Create a new library.

**Request**

```json
{
  "name": "My E-books",
  "paths": ["/mnt/books/epub"],
  "organization_type": "flat",
  "monitored": true
}
```

**Response `201`** — returns the created library object.

---

### GET /api/libraries/{id}

Retrieve a single library.

**Response `200`** — returns the library object.

---

### PUT /api/libraries/{id}

Update a library. Accepts the same body as POST.

**Response `200`** — returns the updated library object.

---

### DELETE /api/libraries/{id}

Delete a library.

**Response `204`** — no body.

---

## Authors

### GET /api/authors

List all authors.

**Response `200`**

```json
[
  {
    "id": "uuid",
    "name": "Jane Austen",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "image_url": null,
    "created_at": "2026-03-13T00:00:00Z",
    "updated_at": "2026-03-13T00:00:00Z"
  }
]
```

---

### POST /api/authors

Create an author.

**Request**

```json
{
  "name": "Jane Austen",
  "goodreads_id": "12345",
  "hardcover_id": null,
  "google_books_id": null,
  "image_url": null
}
```

Only `name` is required. All other fields are optional.

**Response `201`** — returns the created author object.

---

### GET /api/authors/{id}

Retrieve a single author.

**Response `200`** — returns the author object.

---

### PUT /api/authors/{id}

Update an author. Accepts the same body as POST.

**Response `200`** — returns the updated author object.

---

### DELETE /api/authors/{id}

Delete an author.

**Response `204`** — no body.

---

## Series

### GET /api/series

List all series.

**Response `200`**

```json
[
  {
    "id": "uuid",
    "name": "The Lord of the Rings",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "created_at": "2026-03-13T00:00:00Z",
    "updated_at": "2026-03-13T00:00:00Z"
  }
]
```

---

### POST /api/series

Create a series.

**Request**

```json
{
  "name": "The Lord of the Rings",
  "goodreads_id": null,
  "hardcover_id": null,
  "google_books_id": null
}
```

Only `name` is required.

**Response `201`** — returns the created series object.

---

### GET /api/series/{id}

Retrieve a single series.

**Response `200`** — returns the series object.

---

### PUT /api/series/{id}

Update a series. Accepts the same body as POST.

**Response `200`** — returns the updated series object.

---

### DELETE /api/series/{id}

Delete a series.

**Response `204`** — no body.

---

## Books

### GET /api/books

List all books. Returns summary objects (no nested authors/series/files) to avoid N+1 queries.

**Response `200`**

```json
[
  {
    "id": "uuid",
    "title": "Pride and Prejudice",
    "description": null,
    "asin": null,
    "isbn10": "0141439513",
    "isbn13": "9780141439518",
    "goodreads_id": null,
    "hardcover_id": null,
    "google_books_id": null,
    "publication_date": "1813-01-28",
    "publisher": "Penguin Classics",
    "language": "en",
    "num_pages": 432,
    "cover_image_url": null,
    "created_at": "2026-03-13T00:00:00Z",
    "updated_at": "2026-03-13T00:00:00Z"
  }
]
```

---

### POST /api/books

Create a book.

**Request**

```json
{
  "title": "Pride and Prejudice",
  "description": null,
  "asin": null,
  "isbn10": "0141439513",
  "isbn13": "9780141439518",
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

Only `title` is required.

**Response `201`** — returns the full book object (with `authors`, `series`, and `files` arrays).

---

### GET /api/books/{id}

Retrieve a single book with all related data.

**Response `200`**

```json
{
  "id": "uuid",
  "title": "Pride and Prejudice",
  "authors": [{ "id": "uuid", "name": "Jane Austen", ... }],
  "series": [{ "series": { "id": "uuid", "name": "..." }, "position": null }],
  "files": [{ "id": "uuid", "file_type": "epub", ... }],
  ...
}
```

---

### PUT /api/books/{id}

Update a book. Accepts the same body as POST.

**Response `200`** — returns the updated full book object.

---

### DELETE /api/books/{id}

Delete a book.

**Response `204`** — no body.

---

### GET /api/books/{id}/authors

List authors associated with a book.

**Response `200`** — array of author objects.

---

### GET /api/books/{id}/series

List series entries for a book. Each entry contains the series object and an optional position number.

**Response `200`**

```json
[
  { "series": { "id": "uuid", "name": "..." }, "position": 1.0 }
]
```

---

### GET /api/books/{id}/files

List file records attached to a book.

**Response `200`** — array of book-file objects.

---

## Book Files

### GET /api/book-files/{id}

Retrieve metadata for a single book file.

**Response `200`**

```json
{
  "id": "uuid",
  "book_id": "uuid",
  "file_type": "epub",
  "file_name": "pride-and-prejudice.epub",
  "file_size": 524288,
  "file_hash": "sha256:abc123...",
  "file_path": "/mnt/books/epub/pride-and-prejudice.epub",
  "created_at": "2026-03-13T00:00:00Z",
  "updated_at": "2026-03-13T00:00:00Z"
}
```

---

### DELETE /api/book-files/{id}

Remove a file record from the database (does **not** delete the file from disk).

**Response `204`** — no body.

---

## Config

### GET /api/config/status

Return a summary of the server's configuration state (e.g., whether OIDC is configured).

**Response `200`**

```json
{
  "oidc_configured": true
}
```

---

### GET /api/config/oidc

Retrieve the current OIDC configuration. Admin-only.

---

### PUT /api/config/oidc

Update the OIDC configuration. Admin-only.

---

## Admin

All admin endpoints require the authenticated user to have `is_admin: true`.

### GET /api/admin/users

List all users registered in the system.

**Response `200`**

```json
[
  {
    "id": "uuid",
    "email": "jane@example.com",
    "oidc_linked": false,
    "is_admin": true
  }
]
```

---

### PUT /api/admin/users/{id}

Grant or revoke admin privileges for a user.

**Request**

```json
{ "is_admin": true }
```

**Response `200`** — returns the updated user object.

---

## Health

### GET /api/health *(public)*

Liveness probe. Returns `200 OK` when the server is running.

**Response `200`**

```json
{ "status": "ok" }
```

---

## Error responses

All errors use a consistent envelope:

```json
{ "error": "human-readable message" }
```

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| `400` | Bad request — validation error or malformed JSON |
| `401` | Unauthorized — missing or invalid JWT |
| `403` | Forbidden — authenticated but insufficient privileges |
| `404` | Not found |
| `405` | Method not allowed |
| `429` | Too many requests — auth endpoints are rate-limited |
| `500` | Internal server error |
