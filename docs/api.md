# API Overview

Biblioteka exposes a JSON REST API under the `/api` base path. This page summarises the key conventions and links to detailed endpoint documentation.

> For the full endpoint-by-endpoint reference, see **[API Reference](api-reference.md)**.

---

## Base URL

All API endpoints live under `/api`. Timestamps are ISO 8601 strings (e.g. `"2026-03-14T02:00:00Z"`). All request and response bodies use JSON.

---

## Resource IDs

Resource IDs are opaque strings. With the default **SQLite** backend they are 32-character lowercase hex strings; with **PostgreSQL** they are UUID strings. Do not rely on their format — always treat them as opaque.

---

## Authentication

Most endpoints require a bearer credential. Two token types are accepted:

| Token type | Prefix | Obtain via | Accepted on |
|------------|--------|-----------|-------------|
| **JWT** | _(none)_ | `POST /api/auth/login` or `POST /api/auth/signup` | All protected endpoints |
| **API key** | `bib_` | `POST /api/api-keys` | Most protected endpoints (**not** JWT-only routes) |

Supply the token in the `Authorization` header:

```
Authorization: Bearer <token-or-api-key>
```

The browser also receives an `HttpOnly` cookie (`biblioteka_token`) on login, which is sent automatically with all subsequent browser requests — no manual header is required in a browser context.

Certain sensitive endpoints accept **JWT tokens only** (API keys are rejected). See [JWT-only endpoints](api-reference.md#jwt-only-endpoints) for the full list.

---

## Admin-only endpoints

Endpoints marked 🔒 **Admin** require the caller to be a site administrator. Non-admin callers receive `403 Forbidden`.

---

## Pagination

List endpoints accept `limit` (default `50`, max `200`) and `offset` (default `0`) query parameters. Paginated responses include `total`, `limit`, and `offset` envelope fields alongside the data array. Most endpoints silently clamp out-of-range values; `GET /api/audit-logs` returns `400 Bad Request` for invalid values instead.

---

## Rate limiting

The auth, signup, logout, OIDC, and KOSync protocol endpoints are protected by a per-IP token-bucket rate limiter (5 req/s, burst of 10). Exceeding the limit returns `429 Too Many Requests`.

---

## Error responses

All errors are returned as JSON:

```json
{ "error": "human-readable message" }
```

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| `400` | Bad request (invalid input) |
| `401` | Missing or invalid credential |
| `403` | Insufficient permissions |
| `404` | Resource not found |
| `405` | Method not allowed |
| `409` | Conflict (e.g. duplicate name) |
| `429` | Rate limit exceeded |
| `500` | Internal server error |

---

## Resource groups

| Group | Endpoints | Notes |
|-------|-----------|-------|
| **Auth** | `POST /api/auth/signup`, `/login`, `/logout`, `GET /api/auth/me`, `PUT /api/auth/password` | Account creation, session management, password change |
| **OIDC / SSO** | `/api/auth/oidc/*` | OIDC login and account-link flow |
| **API Keys** | `/api/api-keys`, `/api/api-keys/{id}` | Long-lived `bib_`-prefixed access tokens |
| **Config** | `/api/config/status`, `/api/config/oidc`, `/api/config/smtp` | Server configuration (admin only) |
| **Admin** | `/api/admin/users`, `/api/admin/users/{id}` | User management (admin only) |
| **Libraries** | `/api/libraries`, `/api/libraries/{id}` | Named book collections with file-system paths |
| **Authors** | `/api/authors`, `/api/authors/{id}` | Author management |
| **Series** | `/api/series`, `/api/series/{id}` | Series management |
| **Books** | `/api/books`, `/api/books/{id}` and sub-resources | Full book CRUD including author, series, and file associations |
| **Book Files** | `/api/book-files/{id}` | Individual book file records |
| **Audit Logs** | `GET /api/audit-logs` | Append-only action history (admin only) |
| **OPDS Credentials** | `/api/opds/credentials` | Per-user OPDS Basic Auth credential management |
| **OPDS Catalog** | `/opds/*` | OPDS 1.2 catalog for e-reader apps |
| **Kobo Tokens** | `/api/kobo/tokens`, `/api/kobo/tokens/{id}` | Kobo sync token management |
| **KOReader / KOSync** | `/api/kosync/credentials`, `/api/user/*`, `/api/syncs/*` | KOReader kosync-compatible sync |
| **Swagger UI** | `/swagger/` | Interactive API documentation |
| **Monitoring** | `/asynqmon/` | Background job dashboard (admin only) |

---

## Related guides

- [Authentication guide](authentication.md) — configure JWT, OIDC, and account linking
- [OPDS Catalog guide](opds.md) — set up OPDS credentials and connect an e-reader
- [Kobo Sync guide](kobo.md) — pair a Kobo device and manage sync tokens
- [KOReader Sync guide](koreader.md) — configure KOSync credentials
- [Administration guide](administration.md) — audit logs, user management, and library setup
