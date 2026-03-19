# Authentication

Biblioteka supports two authentication methods: local password-based accounts and OIDC/SSO (OpenID Connect). Both methods issue a JWT that the client uses for subsequent requests.

---

## Local Authentication

### Sign up

Send a `POST /api/auth/signup` request. The first account created is automatically an admin.

```bash
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"s3cr3t!"}'
```

The response includes a short-lived JWT and a `biblioteka_token` HttpOnly session cookie:

```json
{
  "token": "<jwt>",
  "user": {
    "id": "<id>",
    "email": "alice@example.com",
    "oidc_linked": false,
    "is_admin": true
  }
}
```

### Log in

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"s3cr3t!"}'
```

### Using the JWT

Supply the token in subsequent requests:

```bash
curl http://localhost:8080/api/books \
  -H "Authorization: Bearer <jwt>"
```

Browser clients receive the token in both the JSON body and as an HttpOnly `SameSite=Strict` cookie (`biblioteka_token`). The cookie is used automatically for browser-based access to protected server-side paths such as `/asynqmon/`.

### Change password

```bash
curl -X PUT http://localhost:8080/api/auth/password \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"currentPassword":"s3cr3t!","newPassword":"b3tt3rS3cr3t!"}'
```

---

## OIDC / SSO Authentication

Biblioteka supports any standards-compliant OpenID Connect provider. OIDC settings can be configured either via environment variables (recommended for production) or at runtime through the admin UI (`Settings → SSO`).

### How OIDC login works

The login flow follows the OAuth 2.0 Authorization Code flow with PKCE:

```
Browser                    Biblioteka              OIDC Provider
   │                           │                        │
   │  GET /api/auth/oidc/login │                        │
   │──────────────────────────▶│                        │
   │                           │  302 → provider /auth  │
   │◀──────────────────────────│                        │
   │  Redirect to provider     │                        │
   │──────────────────────────────────────────────────▶│
   │                           │    302 → /api/auth/    │
   │◀─────────────────────────────────────────────────│
   │  GET /api/auth/oidc/callback?code=…&state=…       │
   │──────────────────────────▶│                        │
   │                           │  Exchange code for     │
   │                           │  id_token             │
   │                           │──────────────────────▶│
   │                           │◀──────────────────────│
   │                           │  Verify id_token,      │
   │                           │  find or create user   │
   │  302 → /?oidc_login=1     │                        │
   │◀──────────────────────────│                        │
```

After the callback, the server sets a `biblioteka_token` session cookie and redirects to `/?oidc_login=1`. The frontend calls `GET /api/auth/me` using the cookie to retrieve the user object and populate the auth store.

#### Session persistence across page reloads

OIDC sessions are maintained entirely through the HttpOnly `biblioteka_token` cookie. On every page load, the auth store always calls `GET /api/auth/me` to restore the current user, regardless of whether a localStorage token is present. This ensures OIDC sessions survive normal browser refreshes.

If a stale localStorage token is found and the `GET /api/auth/me` request returns `401` or `404`, the store clears the stale token and retries the request. The retry succeeds when a valid OIDC session cookie is present, so users are not logged out unexpectedly due to an expired local token sitting alongside an active SSO session.

> **Note:** Transient errors (network failures, `5xx` responses) do not clear the localStorage token. Only definitive auth rejections (`401`/`404`) trigger the stale-token recovery path.

**Scopes requested:** `openid email profile`

**Required claims:** `sub` (subject), `email`, `email_verified` (must be `true`). The `name` claim is used when available; otherwise `email` is used as the display name.

> **Email verification:** Biblioteka rejects OIDC logins where `email_verified` is `false` or missing. This prevents an unverified (and potentially spoofed) email address from being automatically linked to an existing account. If your provider does not set `email_verified`, ensure it is configured to include and verify the claim before users can log in.

### Configuring OIDC

#### Via environment variables (recommended for production)

Set these variables before starting the server:

```bash
OIDC_ISSUER_URL=https://sso.example.com/realms/my-realm
OIDC_CLIENT_ID=biblioteka
OIDC_CLIENT_SECRET=<client-secret>
OIDC_REDIRECT_URI=https://books.example.com/api/auth/oidc/callback
```

Environment variable values take precedence over any settings stored in the database.

#### Via the admin UI at runtime

1. Sign in as an admin.
2. Navigate to **Settings → SSO**.
3. Fill in the four fields and click **Save**.

The server validates the issuer URL by performing OIDC discovery (`<issuer>/.well-known/openid-configuration`) before accepting the configuration.

---

## Provider Setup Examples

### Keycloak

1. Create a realm (e.g. `my-realm`).
2. Create a client:
   - **Client type:** OpenID Connect
   - **Client ID:** `biblioteka`
   - **Client authentication:** On (confidential client)
   - **Valid redirect URIs:** `https://books.example.com/api/auth/oidc/callback`
3. Note the client secret from the **Credentials** tab.

```bash
OIDC_ISSUER_URL=https://keycloak.example.com/realms/my-realm
OIDC_CLIENT_ID=biblioteka
OIDC_CLIENT_SECRET=<client-secret>
OIDC_REDIRECT_URI=https://books.example.com/api/auth/oidc/callback
```

### Authentik

1. Create a **Provider** of type *OAuth2/OpenID Connect*:
   - **Redirect URIs:** `https://books.example.com/api/auth/oidc/callback`
2. Create an **Application** linked to the provider.
3. Note the **Client ID** and **Client Secret**.

```bash
OIDC_ISSUER_URL=https://auth.example.com/application/o/<app-slug>/
OIDC_CLIENT_ID=<client-id>
OIDC_CLIENT_SECRET=<client-secret>
OIDC_REDIRECT_URI=https://books.example.com/api/auth/oidc/callback
```

### Google OAuth

1. Go to the [Google Cloud Console](https://console.cloud.google.com/) → **APIs & Services → Credentials**.
2. Create an **OAuth 2.0 Client ID** of type *Web application*.
3. Add `https://books.example.com/api/auth/oidc/callback` to **Authorized redirect URIs**.

```bash
OIDC_ISSUER_URL=https://accounts.google.com
OIDC_CLIENT_ID=<client-id>.apps.googleusercontent.com
OIDC_CLIENT_SECRET=<client-secret>
OIDC_REDIRECT_URI=https://books.example.com/api/auth/oidc/callback
```

> **Note:** Google does not issue refresh tokens by default for OIDC flows. Biblioteka uses the `sub` claim from the ID token for identity — this works correctly with Google.

### Generic OIDC provider

Any provider that exposes a discovery endpoint at `<issuer>/.well-known/openid-configuration` and issues tokens with `sub` and `email` claims will work.

---

## OIDC Account Linking

Existing password-based accounts can be linked to an OIDC provider without losing access:

1. Sign in with your password.
2. Navigate to **Settings → Account**.
3. Click **Link SSO account**.

When you next visit `/api/auth/oidc/link` with the generated nonce, the server completes the flow and links the `sub` claim from the OIDC provider to your existing account. After linking, you can log in via either method.

**Constraints:**
- Each OIDC identity (`sub`) can be linked to at most one Biblioteka account.
- An account can be linked to at most one OIDC identity at a time.

If a user signs in via OIDC and no existing account has that `sub` claim, the server looks up by `email`. If an account with that email exists **and the identity provider has set `email_verified: true`**, the OIDC subject is automatically linked to it. Otherwise, a new account is created.

> **Why `email_verified` is required for auto-linking:** Automatically tying an OIDC identity to an existing account based on email alone would allow a malicious (or misconfigured) provider to hijack an account by presenting an unverified email address. The `email_verified` check ensures the provider has confirmed ownership of the address. The manual link flow (initiated from within an authenticated session) is exempt from this requirement because the user's identity has already been established.

---

## JWT Details

| Property | Value |
|----------|-------|
| Algorithm | HS256 |
| Expiry | 24 hours |
| Secret | Configured via `JWT_SECRET` environment variable |
| Cookie name | `biblioteka_token` |
| Cookie flags | `HttpOnly`, `SameSite=Strict`, `Secure` (when `SECURE_COOKIES=true`) |

> **Security note:** Keep `JWT_SECRET` secret and rotate it periodically. Rotating the secret immediately invalidates all existing tokens, forcing all users to log in again. The default value `change-me-in-production` must not be used in production.

---

## API Keys

API keys provide a convenient way to authenticate programmatic access to Biblioteka without requiring you to store your password or manage JWT expiry.

### When to use API keys

Use API keys when:

- You are calling the Biblioteka API from a script, CI pipeline, or external service.
- You want long-lived credentials that do not expire on a fixed schedule.
- You need to avoid storing your password in automation tooling.

Use JWT tokens (from login/signup) for interactive browser sessions.

### Key format

Every API key begins with the prefix `bib_` followed by 32 lowercase hexadecimal characters (128 bits of entropy), for example:

```
bib_a3f2c8e1d074b651...
```

Only the `bib_` prefix and the first 12 hex characters (the *key prefix*) are stored in plaintext for identification in the UI. The remainder is stored only as a SHA-256 hash — the full key is never retrievable after creation.

### Using an API key

Supply the key in the `Authorization` header as a Bearer token:

```bash
curl http://localhost:8080/api/books \
  -H "Authorization: Bearer bib_a3f2c8e1d074b651..."
```

> **Security note:** API keys are accepted **only** from the `Authorization` header. They are explicitly rejected from cookies to prevent [Cross-Site Request Forgery (CSRF)](https://owasp.org/www-community/attacks/csrf) attacks.

### Managing API keys via the UI

1. Sign in to Biblioteka.
2. Navigate to **Settings → API Keys**.
3. Enter a descriptive name (e.g., `CI Pipeline`) and click **Create Key**.
4. Copy the full key immediately — it is shown **only once** at creation and cannot be retrieved later.
5. To revoke a key, click **Delete** next to it in the key list.

To manage keys programmatically, see the [API Keys endpoints](api-reference.md#api-keys) in the API reference.

### Security properties

| Property | Detail |
|----------|--------|
| Entropy | 128 bits (cryptographically random) |
| Storage | SHA-256 hash only — plaintext key is never persisted after creation |
| Scope | Tied to the creating user; inherits that user's permissions |
| Transmission | HTTPS only in production; `Authorization` header only (cookies rejected) |
| Visibility | Key prefix (`bib_XXXXXXXXXXXX`) shown in the UI for identification |
| Last used | Timestamp updated lazily (throttled to at most once per 5 minutes) |

> **Keep API keys secret.** Anyone who obtains a key can authenticate as you. Revoke and reissue keys that may have been exposed.

---

## Rate Limiting

Signup, login, logout, all OIDC auth endpoints (`/api/auth/oidc/login`, `/api/auth/oidc/callback`, `/api/auth/oidc/link`), the KOReader/KOSync protocol endpoints (`/api/user/create`, `/api/user/auth`, `/api/syncs/progress`), and the SMTP test endpoint (`POST /api/config/smtp/test`) are protected by a per-IP token-bucket rate limiter:

| Parameter | Value |
|-----------|-------|
| Rate | 5 requests per second |
| Burst | 10 requests |
| Scope | Per client IP address |
| Response on limit | `429 Too Many Requests` |
