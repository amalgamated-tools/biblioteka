<!-- disable-agentic-editing: true -->

# API Reference — Auth

[← Back to API Reference](../api-reference.md)

## Auth

> **Rate limiting:** The signup, login, logout, all OIDC auth endpoints (`/api/auth/oidc/login`, `/api/auth/oidc/callback`, `/api/auth/oidc/link`), and the KOReader kosync protocol endpoints (`/api/user/create`, `/api/user/auth`, `/api/syncs/progress`) are protected by a per-IP token-bucket rate limiter (5 requests/second, burst of 10). Exceeding the limit returns `429 Too Many Requests`.

### `POST /api/auth/signup`

Create a new user account. The first user to sign up automatically becomes an admin.

**Request body:**

| Field      | Type   | Required | Description           |
|------------|--------|----------|-----------------------|
| `name`     | string | ✓        | Display name          |
| `email`    | string | ✓        | Email address         |
| `password` | string | ✓        | Password (min 8 chars) |

**Responses:**

| Status | Description |
|--------|-------------|
| `201 Created` | Account created; returns token and user object |
| `400 Bad Request` | Missing or invalid fields |
| `403 Forbidden` | Signup is disabled on this server (`DISABLE_SIGNUP=true`) |
| `409 Conflict` | Email already registered |
| `429 Too Many Requests` | Rate limit exceeded |

> **Self-registration control:** Server operators can set `DISABLE_SIGNUP=true` to prevent new accounts from being created. When disabled, the sign-up form is hidden in the UI and all `POST` requests to this endpoint return `403 Forbidden`. Check the current state with [`GET /api/auth/signup/enabled`](#get-apiauthsignupenabled).

**Response body (`201`):**

```json
{
  "token": "<jwt>",
  "user": {
    "id": "<id>",
    "name": "Alice",
    "email": "alice@example.com",
    "oidc_linked": false,
    "is_admin": true
  }
}
```

---

### `GET /api/auth/signup/enabled`

Returns whether new user self-registration is currently permitted on this server. Also accepts `HEAD` (returns headers only, no body). No authentication required.

**Response body (`200`):**

```json
{ "enabled": true }
```

| Field     | Type    | Description |
|-----------|---------|-------------|
| `enabled` | boolean | `true` when `POST /api/auth/signup` accepts new registrations; `false` when `DISABLE_SIGNUP=true` is set |

---

### `POST /api/auth/login`

Authenticate with email and password.

**Request body:**

| Field      | Type   | Required |
|------------|--------|----------|
| `email`    | string | ✓        |
| `password` | string | ✓        |

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Returns token and user object |
| `400 Bad Request` | Missing fields |
| `401 Unauthorized` | Invalid credentials or OIDC-only account |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Database or token generation error |

**Response body (`200`):** Same shape as signup response above.

---

### `GET /api/auth/me` 🔒

Return the currently authenticated user's profile.

**Response body (`200`):**

```json
{
  "id": "<id>",
  "name": "Alice",
  "email": "alice@example.com",
  "oidc_linked": false,
  "is_admin": true
}
```

| Field         | Type    | Description |
|---------------|---------|-------------|
| `id`          | string  | Opaque user ID |
| `name`        | string  | User's display name |
| `email`       | string  | User's email address |
| `oidc_linked` | boolean | `true` when the account has an OIDC/SSO identity linked; `false` for local-only accounts |
| `is_admin`    | boolean | `true` when the user has admin privileges |

---

### `PUT /api/auth/me` 🔒

Update the authenticated user's display name.

**Request body:**

| Field  | Type   | Required | Description |
|--------|--------|----------|-------------|
| `name` | string | ✓        | New display name (must be non-blank) |

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Name updated; returns updated user object |
| `400 Bad Request` | Missing or blank `name` |
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | Authenticated user no longer exists |

**Response body (`200`):** Same shape as `GET /api/auth/me` above, with the updated `name`.

---

### `PUT /api/auth/password` 🔒 **JWT only**

Change the authenticated user's password. Not supported for OIDC-only accounts.

**Request body:**

| Field             | Type   | Required |
|-------------------|--------|----------|
| `currentPassword` | string | ✓        |
| `newPassword`     | string | ✓        |

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Password updated |
| `400 Bad Request` | Missing fields, invalid password, or OIDC-only account |
| `401 Unauthorized` | Missing or invalid JWT, current password is incorrect, or user no longer exists |
| `429 Too Many Requests` | Rate limit exceeded |
| `500 Internal Server Error` | Database or password hashing error |

---

### `POST /api/auth/logout`

Sign the current user out. Clears the session cookie.

The request must originate from the same origin as the server (the `Origin` or `Referer` header is validated). Cross-origin logout requests are rejected.

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Session cookie cleared; returns confirmation |
| `403 Forbidden` | Request origin does not match server origin |
| `405 Method Not Allowed` | Non-POST request |
| `429 Too Many Requests` | Rate limit exceeded |

**Response body (`200`):**

```json
{ "message": "logged out" }
```

---

## OIDC / SSO

### `GET /api/auth/oidc/enabled`

Returns whether OIDC login is currently configured and enabled. Also accepts `HEAD` (returns headers only, no body).

**Response body (`200`):**

```json
{ "enabled": true }
```

---

### `GET /api/auth/oidc/login`

Redirects the browser to the OIDC provider's authorization endpoint. Used to initiate SSO login.

**Response:** `302 Found` redirect to OIDC provider.

---

### `GET /api/auth/oidc/callback`

Handles the OAuth 2.0 callback from the OIDC provider. Not called directly; the browser is redirected here by the OIDC provider after authentication.

For a **normal login** the server sets the `biblioteka_token` session cookie and redirects to `/?oidc_login=1`.

For an **account-link** flow (initiated via `GET /api/auth/oidc/link`) the server verifies the HMAC-signed state parameter, extracts the user ID, and links the provider's `sub` claim to that account. On success the browser is redirected to `/?oidc_linked=true`; on failure to `/?oidc_link_error=<reason>`.

---

### `GET /api/auth/oidc/link`

Link an OIDC identity to an existing local account. Expects a `nonce` query parameter generated by [`POST /api/auth/oidc/link-nonce`](#post-apiauthoidclink-nonce).

The server consumes the nonce, embeds the user ID in an HMAC-signed OIDC state parameter, and redirects the browser to the OIDC provider's authorization endpoint. The provider then redirects back to `/api/auth/oidc/callback` to complete the link. See the [account linking flow](authentication.md#oidc-account-linking) for the full sequence.

**Request:** `GET /api/auth/oidc/link?nonce=<token>`

**Response:** `302 Found` redirect to OIDC provider.

| Status | Description |
|--------|-------------|
| `302 Found` | Redirect to OIDC provider |
| `400 Bad Request` | Missing or invalid nonce, or user not found |
| `401 Unauthorized` | Nonce expired |
| `409 Conflict` | Account already linked to an SSO provider |

---

### `POST /api/auth/oidc/link-nonce` 🔒

Generate a short-lived (5 minutes), single-use nonce that authorises the OIDC account-linking flow. This is the first step of the [account-linking sequence](authentication.md#oidc-account-linking): pass the returned nonce as the `nonce` query parameter to `GET /api/auth/oidc/link`.

**Response body (`200`):**

```json
{ "nonce": "<token>" }
```

| Status | Description |
|--------|-------------|
| `200 OK` | Nonce created |
| `401 Unauthorized` | Not authenticated |
| `500 Internal Server Error` | Failed to generate nonce |

---

## API Keys

API keys allow programmatic access to Biblioteka without a JWT. Keys begin with the prefix `bib_` and are supplied via the `Authorization` header. See the [authentication guide](authentication.md#api-keys) for full details.

> **Note:** All API key management endpoints require a **JWT** (not an API key). This prevents an API key from listing, creating, or deleting other API keys.

### `GET /api/api-keys` 🔒 **JWT only**

List all API keys belonging to the authenticated user. Results are ordered by creation time (newest first). The full key value is **never** returned by this endpoint — only the prefix and metadata.

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Returns an array of API key objects |
| `401 Unauthorized` | Missing or invalid JWT |
| `500 Internal Server Error` | Unexpected error |

**Response body (`200`):**

```json
[
  {
    "id": "f47ac10b58cc4372a567b409e2087bc1",
    "name": "CI Pipeline",
    "key_prefix": "bib_a3f2c8e1d074",
    "last_used_at": "2026-03-15T10:00:00Z",
    "created_at": "2026-03-14T09:00:00Z"
  }
]
```

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Opaque key ID |
| `name` | string | Human-readable label |
| `key_prefix` | string | `bib_` prefix + first 12 hex chars — for identification only |
| `last_used_at` | string \| null | ISO 8601 timestamp of last use, or `null` if never used |
| `created_at` | string | ISO 8601 creation timestamp |

---

### `POST /api/api-keys` 🔒 **JWT only**

Create a new API key for the authenticated user.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Descriptive label (max 100 characters) |

**Responses:**

| Status | Description |
|--------|-------------|
| `201 Created` | Key created; full key value included in response |
| `400 Bad Request` | Missing or invalid `name` |
| `401 Unauthorized` | Missing or invalid JWT |
| `500 Internal Server Error` | Key generation or database error |

**Response body (`201`):**

```json
{
  "id": "f47ac10b58cc4372a567b409e2087bc1",
  "name": "CI Pipeline",
  "key_prefix": "bib_a3f2c8e1d074",
  "last_used_at": null,
  "created_at": "2026-03-15T09:00:00Z",
  "key": "bib_a3f2c8e1d074b651..."
}
```

> **Important:** The `key` field is returned **only once** at creation. Store it securely — it cannot be retrieved later. The response also sets `Cache-Control: no-store` and `Pragma: no-cache` headers to prevent caching.

---

### `DELETE /api/api-keys/{id}` 🔒 **JWT only**

Permanently revoke an API key. The caller must own the key.

**Path parameters:**

| Parameter | Description |
|-----------|-------------|
| `id` | API key ID |

**Responses:**

| Status | Description |
|--------|-------------|
| `204 No Content` | Key deleted successfully |
| `400 Bad Request` | Invalid key ID format |
| `401 Unauthorized` | Missing or invalid JWT |
| `404 Not Found` | Key not found or does not belong to the caller |
| `405 Method Not Allowed` | Invalid HTTP method |
| `500 Internal Server Error` | Unexpected error |

---

