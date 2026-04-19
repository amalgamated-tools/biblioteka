<!-- disable-agentic-editing: true -->

# API Reference — Config

[← Back to API Reference](../api-reference.md)

## Config

### `GET /api/config/status` 🔒 **JWT only**

Returns application configuration status visible to the authenticated user.

**Response body (`200`):**

```json
{
  "oidc_configured": true,
  "smtp_configured": true,
  "is_admin": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `oidc_configured` | boolean | `true` when an OIDC provider is configured and active |
| `smtp_configured` | boolean | `true` when an SMTP host and a valid `From` address are configured |
| `is_admin` | boolean | `true` when the authenticated user has admin privileges |

---

### `GET /api/config/oidc` 🔒 **Admin** · **JWT only**

Return the current OIDC configuration. The `client_secret` value is never returned; `client_secret_set` indicates whether one is stored.

**Response body (`200`):**

```json
{
  "issuer_url": "https://accounts.example.com",
  "client_id": "my-client-id",
  "client_secret_set": true,
  "redirect_uri": "http://localhost:8080/api/auth/oidc/callback"
}
```

| Field               | Type    | Description |
|---------------------|---------|-------------|
| `issuer_url`        | string  | OIDC provider issuer URL (e.g. `https://accounts.example.com`) |
| `client_id`         | string  | OAuth 2.0 client ID registered with the provider |
| `client_secret_set` | boolean | `true` when a client secret has been stored; the secret itself is never returned |
| `redirect_uri`      | string  | Callback URL configured in the OIDC provider (must match exactly) |

---

### `PUT /api/config/oidc` 🔒 **Admin** · **JWT only**

Save OIDC provider settings. The server performs OIDC discovery on the `issuer_url` before saving. If `client_secret` is omitted, the existing stored secret is preserved; however, a `client_secret` **must** be supplied the first time OIDC is configured (when no secret is yet stored).

All four fields (`issuer_url`, `client_id`, `client_secret`, `redirect_uri`) are written in a single database transaction — either every field is saved or none are. If the database write fails mid-way, the entire update is rolled back and the previous configuration remains intact.

**Request body:**

| Field           | Type   | Required | Description |
|-----------------|--------|----------|-------------|
| `issuer_url`    | string | ✓        | OIDC provider issuer URL |
| `client_id`     | string | ✓        | OAuth 2.0 client ID |
| `client_secret` | string | ✓*       | OAuth 2.0 client secret; required on initial setup, omit to keep existing |
| `redirect_uri`  | string | ✓        | Callback URL registered with the provider |

\* Required when no `client_secret` is currently stored (initial OIDC setup); may be omitted to preserve an existing secret.

**Response body (`200`):**

```json
{ "message": "OIDC configuration saved successfully" }
```

> **Note:** If the `OIDC_ISSUER_URL` environment variable is set, it takes precedence over these stored settings at server startup — the server will use the environment variables instead of the saved configuration. When `OIDC_ISSUER_URL` is set at the time of this `PUT` request, the response message will warn about this and instruct you to remove the variable to use the stored settings.

---

### `GET /api/config/smtp` 🔒 **Admin** · **JWT only**

Return the current SMTP configuration. The `password` value is never returned; `password_set` indicates whether one is stored. When `env_override` is `true`, all SMTP settings are sourced from environment variables and any database-stored values are ignored.

**Response body (`200`):**

```json
{
  "host": "smtp.example.com",
  "port": "587",
  "username": "user@example.com",
  "password_set": true,
  "from": "biblioteka@example.com",
  "tls": "starttls",
  "env_override": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `host` | string | SMTP server hostname or IP address |
| `port` | string | SMTP server port (defaults to `"587"` when not set) |
| `username` | string | SMTP authentication username (empty for unauthenticated SMTP) |
| `password_set` | boolean | `true` when a password is stored; the password itself is never returned |
| `from` | string | Envelope `From` address used for outgoing mail |
| `tls` | string | TLS mode: `"none"`, `"starttls"`, or `"tls"` |
| `env_override` | boolean | `true` when `SMTP_HOST` is set as an environment variable and overrides database settings |

---

### `PUT /api/config/smtp` 🔒 **Admin** · **JWT only**

Save SMTP server settings. If `password` is omitted while `username` is supplied and matches the currently stored username, the existing password is preserved. Setting `username` to an empty string clears both the stored username and password.

All six fields (`host`, `port`, `username`, `password`, `from`, `tls`) are written in a single database transaction — either every field is saved or none are. If the database write fails mid-way, the entire update is rolled back and the previous configuration remains intact.

**Request body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | ✓ | SMTP server hostname or IP address (no scheme, port, or path) |
| `port` | string | | SMTP port; defaults to `"587"` when omitted |
| `username` | string | | SMTP authentication username; leave empty for unauthenticated SMTP |
| `password` | string | ✓* | SMTP authentication password; required when `username` is set and no password is currently stored |
| `from` | string | ✓ | Envelope `From` address; accepts a bare address (e.g. `no-reply@example.com`) or RFC 5322 format with a display name (e.g. `"Biblioteka" <no-reply@example.com>`) |
| `tls` | string | | `"none"`, `"starttls"` (default), or `"tls"` |

\* Required when `username` is set and no password is currently stored; may be omitted to preserve an existing password for the same username.

**Validation rules:**
- `host` must be a valid hostname or IP address — no scheme (`smtp://`), embedded port, or path component.
- Authenticated SMTP (`username` set) without TLS is only allowed for localhost/loopback addresses. Use `starttls` or `tls` for remote servers.
- `from` must be a valid email address parseable by RFC 5322 — either a bare address (`user@host`) or a display-name form (`"Display Name" <user@host>`). The display name is preserved in the message `From:` header; the bare address is used for the SMTP envelope.

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Settings saved; returns confirmation message |
| `400 Bad Request` | Validation error (invalid host, port, from address, or TLS mode) |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `500 Internal Server Error` | Database error |

**Response body (`200`):**

```json
{ "message": "SMTP configuration saved successfully" }
```

> **Note:** If the `SMTP_HOST` environment variable is set, it takes precedence over database settings at server startup. When `SMTP_HOST` is set at the time of this `PUT` request, the response message will warn about this and instruct you to remove the variable to use the stored settings.

---

### `POST /api/config/smtp/test` 🔒 **Admin** · **JWT only**

Send a test email to the authenticated admin's registered email address using the current SMTP configuration. This is useful for verifying that SMTP settings are correct before relying on them.

**Request body:** none

**Responses:**

| Status | Description |
|--------|-------------|
| `200 OK` | Test email sent successfully |
| `400 Bad Request` | SMTP not configured, incomplete environment configuration, or invalid SMTP settings |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `502 Bad Gateway` | SMTP connection or delivery failure |

**Response body (`200`):**

```json
{ "message": "Test email sent to alice@example.com" }
```

> **Note:** This endpoint is rate-limited at the same rate as the auth endpoints (5 requests/second, burst of 10). Exceeding the limit returns `429 Too Many Requests`.

---

### `GET /api/config/watch-folder` 🔒 **Admin** · **JWT only**

Return the current watch folder configuration.

**Response body (`200`):**

```json
{
  "path": "/mnt/inbox",
  "library_id": "a1b2c3d4e5f6..."
}
```

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Absolute path to the watch folder; empty string when not configured |
| `library_id` | string | ID of the library new files are imported into; empty string when not configured |

| Status | Description |
|--------|-------------|
| `200 OK` | Current watch folder configuration (may have empty `path` and `library_id` if not set) |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `500 Internal Server Error` | Failed to verify admin permissions |

---

### `PUT /api/config/watch-folder` 🔒 **Admin** · **JWT only**

Update (or clear) the watch folder configuration. The watch folder is a directory the server monitors for newly added book files; any file dropped there is automatically imported into the associated library.

**Request body:**

```json
{
  "path": "/mnt/inbox",
  "library_id": "a1b2c3d4e5f6..."
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | | Absolute path to the watch folder. Send an empty string to **clear** the configuration. |
| `library_id` | string | | ID of the library to import files into. Required when `path` is non-empty. |

**Behavior:**

- If `path` is empty, both `path` and `library_id` are cleared and the watch folder is disabled. The response body contains `{ "message": "Watch folder configuration cleared" }`.
- If `path` is non-empty it must be an absolute path to an existing directory. The server validates the path (with a 5-second timeout to guard against slow NFS/SMB mounts).
- `library_id` must refer to an existing library when `path` is set.
- A successful update is recorded in the audit log as `watch_folder.config_updated`.

**Response body (`200`):**

```json
{ "message": "Watch folder configuration saved successfully" }
```

| Status | Description |
|--------|-------------|
| `200 OK` | Configuration saved or cleared |
| `400 Bad Request` | Path is not absolute; path does not exist or is not a directory; `library_id` is missing or refers to a non-existent library |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `500 Internal Server Error` | Path validation timed out; database error |

---

### `GET /api/config/llm` 🔒 **Admin** · **JWT only**

Return the current LLM configuration used for AI metadata enrichment.

**Response body (`200`):**

```json
{
  "provider": "ollama",
  "endpoint": "http://localhost:11434",
  "model": "llama3.2",
  "enabled": true
}
```

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | LLM provider name (currently only `"ollama"` is supported); may be an empty string when not configured |
| `endpoint` | string | Base URL of the LLM server (e.g. `"http://localhost:11434"`); may be an empty string when not configured |
| `model` | string | Model name used for generation (e.g. `"llama3.2"`); may be an empty string when not configured |
| `enabled` | boolean | Stored setting for whether AI enrichment is enabled; `true` does not by itself guarantee enrichment is currently active at runtime |

| Status | Description |
|--------|-------------|
| `200 OK` | Current LLM configuration |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `500 Internal Server Error` | Database error |

---

### `PUT /api/config/llm` 🔒 **Admin** · **JWT only**

Save the LLM configuration. **A server restart is required for changes to take effect.**

**Request body:**

```json
{
  "provider": "ollama",
  "endpoint": "http://localhost:11434",
  "model": "llama3.2",
  "enabled": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | | LLM provider name. Currently only `"ollama"` is supported. Defaults to `"ollama"` when omitted and `enabled` is `true`. |
| `endpoint` | string | when `enabled` | Base URL of the LLM server. Required when `enabled` is `true`. |
| `model` | string | when `enabled` | Model name. Required when `enabled` is `true`. |
| `enabled` | boolean | | `true` to activate AI enrichment; `false` to disable it. |

**Response body (`200`):**

```json
{
  "provider": "ollama",
  "endpoint": "http://localhost:11434",
  "model": "llama3.2",
  "enabled": true,
  "restart_required": true
}
```

The response always includes `"restart_required": true` — the server must be restarted for any LLM configuration change to take effect.

> A successful update is recorded in the audit log as `llm.config_updated`.

| Status | Description |
|--------|-------------|
| `200 OK` | Configuration saved; restart required |
| `400 Bad Request` | `endpoint` or `model` missing when `enabled` is `true`; unsupported `provider` value |
| `401 Unauthorized` | Missing or invalid JWT |
| `403 Forbidden` | Caller is not an admin |
| `500 Internal Server Error` | Database error |

---

