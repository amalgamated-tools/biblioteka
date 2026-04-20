# Authentication

Biblioteka supports two authentication methods: local password-based accounts and OIDC/SSO (OpenID Connect). Both methods issue a JWT that the client uses for subsequent requests.

---

## Local Authentication

### Sign up

Send a `POST /api/auth/signup` request. The first account created is automatically an admin.

```bash
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"s3cr3t!1"}'
```

The response includes a short-lived JWT and a `biblioteka_token` HttpOnly session cookie:

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

### Log in

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"s3cr3t!1"}'
```

### Using the JWT

Supply the token in subsequent requests:

```bash
curl http://localhost:8080/api/books \
  -H "Authorization: Bearer <jwt>"
```

Browser clients receive the token in both the JSON body and as an HttpOnly `SameSite=Strict` cookie (`biblioteka_token`). The cookie is used automatically for browser-based access to protected server-side paths such as `/asynqmon/`.

### Update profile

Change your display name:

```bash
curl -X PUT http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Wonderland"}'
```

Returns the updated user object.

### Change password

```bash
curl -X PUT http://localhost:8080/api/auth/password \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"currentPassword":"s3cr3t!1","newPassword":"b3tt3rS3cr3t!"}'
```

### Password security properties

| Property | Detail |
|----------|--------|
| Minimum length | [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html) recommends a minimum of 8 characters; this server currently enforces a minimum of 8 bytes because password length validation is byte-counted |
| Maximum length | 72 bytes (bcrypt silently truncates inputs longer than 72 bytes; any two passwords sharing the same first 72 bytes would be treated as identical — this cap prevents that collision) |
| Storage | bcrypt hash with work factor 12; plaintext is never persisted |
| Work factor rationale | Cost 12 is used instead of bcrypt's default (cost 10) because modern hardware can brute-force cost-10 hashes significantly faster than when the default was standardized |
| Account enumeration | The main login endpoint (`POST /api/auth/login`) returns the same generic `"invalid email or password"` error for both non-existent accounts and OIDC-only accounts (accounts with no password set). This prevents an attacker from using the login endpoint to discover which email addresses are registered as OIDC-only versus not registered at all. |
| Timing safety | Protocol-layer credential endpoints (OPDS, KOSync) perform a constant-time dummy bcrypt comparison when a username is not found to mitigate timing-based enumeration. The main login endpoint (`POST /api/auth/login`) applies the same protection: a dummy bcrypt comparison is performed when the account does not exist or is OIDC-only, so response times are indistinguishable across all three cases. |

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

Existing password-based accounts can be linked to an OIDC provider without losing access.

### How the link flow works

The link flow is a two-phase process that piggybacks on the same OAuth 2.0 Authorization Code + PKCE flow used for normal login, with the addition of a signed state parameter to carry the user identity.

```
Browser (authenticated)    Biblioteka              OIDC Provider
         │                     │                        │
         │  POST /api/auth/    │                        │
         │  oidc/link-nonce    │                        │
         │────────────────────▶│                        │
         │  { "nonce": "…" }   │                        │
         │◀────────────────────│                        │
         │                     │                        │
         │  GET /api/auth/oidc/link?nonce=…             │
         │────────────────────▶│                        │
         │                     │ nonce consumed,        │
         │                     │ signed state created   │
         │                     │  302 → provider /auth  │
         │◀────────────────────│                        │
         │  Redirect to provider                        │
         │──────────────────────────────────────────────▶│
         │                     │    302 → /api/auth/    │
         │◀────────────────────────────────────────────│
         │  GET /api/auth/oidc/callback?code=…&state=…  │
         │────────────────────▶│                        │
         │                     │ state verified,        │
         │                     │ user ID extracted,     │
         │                     │ OIDC sub linked        │
         │  302 → /?oidc_linked=true                    │
         │◀────────────────────│                        │
```

**Step 1 — Create a link nonce:** The frontend calls `POST /api/auth/oidc/link-nonce` (authenticated). The server stores a short-lived (5-minute), single-use token mapping the nonce to the current user ID.

**Step 2 — Initiate the OIDC flow:** The frontend navigates the browser to `GET /api/auth/oidc/link?nonce=…`. The server consumes the nonce (removing it from the store), then embeds the user ID into the OIDC `state` parameter as an **HMAC-SHA256–signed payload** and redirects to the provider's authorization endpoint. This signed state makes user-ID propagation through the OIDC round-trip tamper-proof and stateless — no per-instance server-side state is needed between this step and the callback.

> **Signed state format:** `<random>.<base64url(userID)>.<base64url(HMAC-SHA256(random + "|" + userID))>`. The HMAC key is derived from `JWT_SECRET` via HKDF with the purpose label `oidc-link-state`.

**Step 3 — Callback completes the link:** After the provider redirects back to `/api/auth/oidc/callback`, the server verifies the HMAC signature in the state parameter, extracts the user ID, and links the provider's `sub` claim to that account. On success the browser is redirected to `/?oidc_linked=true`. On failure it is redirected to `/?oidc_link_error=<reason>`.

### Initiating account linking from the UI

1. Sign in with your password.
2. Navigate to **Settings → Account**.
3. Click **Link SSO account**.

After the flow completes you can log in via either your password or your SSO provider.

**Constraints:**
- Each OIDC identity (`sub`) can be linked to at most one Biblioteka account.
- An account can be linked to at most one OIDC identity at a time.
- The link nonce expires after 5 minutes and can only be used once.

### Automatic linking on first OIDC login

If a user signs in via OIDC and no existing account has that `sub` claim, the server looks up by `email`. If an account with that email exists **and the identity provider has set `email_verified: true`**, the OIDC subject is automatically linked to it. Otherwise, a new account is created.

> **Why `email_verified` is required for auto-linking:** Automatically tying an OIDC identity to an existing account based on email alone would allow a malicious (or misconfigured) provider to hijack an account by presenting an unverified email address. The `email_verified` check ensures the provider has confirmed ownership of the address. The manual link flow (initiated from within an authenticated session) is exempt from this requirement because the user's identity has already been established.

---

## JWT Details

| Property | Value |
|----------|-------|
| Algorithm | HS256 |
| Expiry | 24 hours |
| `iss` claim | `"biblioteka"` — issuer; validated on every request |
| `aud` claim | `"biblioteka"` — audience; validated on every request; tokens missing this claim are rejected |
| Secret | Configured via `JWT_SECRET` environment variable; minimum **required** length is **32 bytes** (enforced at startup) |
| Cookie name | `biblioteka_token` |
| Cookie flags | `HttpOnly`, `SameSite=Strict`, `Secure` (when `SECURE_COOKIES=true`) |

> **Security note:** Keep `JWT_SECRET` secret and rotate it periodically. Rotating the secret immediately invalidates all existing tokens, forcing all users to log in again. The default value `change-me-in-production` must not be used in production.
>
> `JWT_SECRET` is also used to derive the encryption key for sensitive settings stored in the database — specifically the SMTP password and OIDC client secret. These values are encrypted at rest using AES-256-GCM with an HKDF-derived key (purpose label: `settings-secret-v1`). If `JWT_SECRET` changes, existing encrypted DB-stored values become unreadable, so operators must reconfigure the SMTP password and OIDC client secret after a key rotation. This can be done through the admin UI or the admin config APIs, and values supplied via environment variables take precedence over database settings. Settings stored in plaintext before the upgrade are accepted transparently and re-encrypted on the next write.
>
> If `JWT_SECRET` is set but shorter than 32 bytes, the **server refuses to start** with an error. Either unset `JWT_SECRET` (the server generates a random secret on startup — suitable for development only) or provide a value of at least 32 bytes. Use `openssl rand -hex 32` to generate a strong 64-character secret.

---

## Passkeys (WebAuthn)

Passkeys provide a phishing-resistant, passwordless way to log in to Biblioteka. A passkey is a cryptographic credential stored on your device — a hardware security key, a fingerprint reader, Face ID, or a platform authenticator (Windows Hello, macOS Touch ID, or a mobile device). Once registered, a passkey gives the user an additional sign-in option using a local biometric or PIN gesture instead of entering a password; password and OIDC/SSO login remain available if configured.

> **Local development only by default:** Passkeys are pre-configured for `localhost` out of the box and will appear in the UI on any server. They will only work when `WEBAUTHN_RP_ID` matches the domain users actually access. In any environment other than `http://localhost:8080`, an operator must set the `WEBAUTHN_*` variables below or all passkey ceremonies will fail.

### Configuring passkeys (server configuration)

Set these environment variables before starting the server:

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBAUTHN_RP_ID` | `localhost` | Relying party ID — must be a valid domain name that matches the hostname your users access Biblioteka from (e.g. `books.example.com`). |
| `WEBAUTHN_RP_ORIGINS` | `http://localhost:8080` | Comma-separated list of fully-qualified origins allowed to use passkeys (e.g. `https://books.example.com`). Must include the scheme and port when non-standard. **Only `localhost` may use `http://` for local development; all other origins must use `https://`**. Browsers require WebAuthn to run in a secure context, so configuring `http://` for a non-`localhost` domain is invalid and will cause passkey ceremonies to fail. |
| `WEBAUTHN_RP_NAME` | `Biblioteka` | Human-readable name shown in the browser's passkey dialog. |

```bash
WEBAUTHN_RP_ID=books.example.com
WEBAUTHN_RP_ORIGINS=https://books.example.com
WEBAUTHN_RP_NAME=My Biblioteka
```

> **Production requirement:** `WEBAUTHN_RP_ID` must exactly match the effective domain of your Biblioteka instance. For example, if your instance is at `https://books.example.com`, set `WEBAUTHN_RP_ID=books.example.com`. The default `localhost` value makes passkeys non-functional outside of local development — ceremonies will silently fail while the UI still shows passkeys as available. On startup, `WebAuthn passkeys enabled` is logged at `INFO` level confirming the RP ID and name.

The `GET /api/auth/passkey/enabled` endpoint returns `{"enabled": true}` when WebAuthn initializes successfully, including when the server falls back to localhost defaults, and `{"enabled": false}` only when WebAuthn initialization fails. The frontend uses this to conditionally show the passkey login button.

For non-localhost or production deployments, you must set `WEBAUTHN_RP_ID` and `WEBAUTHN_RP_ORIGINS` to the real domain and allowed origins for your Biblioteka instance. If these values are left at localhost defaults or otherwise do not match the deployed site, the endpoint may still report `{"enabled": true}` and the UI may show passkey actions, but passkey registration and login ceremonies will fail in the browser.

### Registering a passkey

A passkey can only be registered by an authenticated user (the registration flow requires a valid JWT session). After logging in with your password or OIDC, go to **Settings → Account → Passkeys** and click **Add passkey**. Give the passkey a descriptive name (e.g. `"MacBook Touch ID"` or `"YubiKey 5"`), then follow the browser prompt to authenticate with your device.

**Via the API:**

1. **Begin registration** — send a `POST /api/auth/passkey/register/begin` request with a name for the new passkey:

   ```bash
   curl -X POST http://localhost:8080/api/auth/passkey/register/begin \
     -H "Authorization: Bearer <jwt>" \
     -H "Content-Type: application/json" \
     -d '{"name": "YubiKey 5"}'
   ```

   Response:

   ```json
   {
     "session_id": "<session-id>",
     "options": { ... }   // PublicKeyCredentialCreationOptions — pass to navigator.credentials.create()
   }
   ```

   The `session_id` is a short-lived server-side challenge (valid for 5 minutes) that must accompany the finish request.

2. **Finish registration** — after the authenticator responds, send the `PublicKeyCredential` to `POST /api/auth/passkey/register/finish?session_id=<session-id>`:

   ```bash
   curl -X POST "http://localhost:8080/api/auth/passkey/register/finish?session_id=<session-id>" \
     -H "Authorization: Bearer <jwt>" \
     -H "Content-Type: application/json" \
     -d '<raw PublicKeyCredential JSON from navigator.credentials.create()>'
   ```

   A `201 Created` response returns the stored credential:

   ```json
   {
     "id": "<credential-id>",
     "name": "YubiKey 5",
     "aaguid": "<aaguid>",
     "created_at": "2026-04-15T12:00:00Z"
   }
   ```

A passkey is excluded from future registration options once registered, preventing duplicate entries for the same authenticator.

### Logging in with a passkey

Passkey login uses discoverable credentials — no username is required.

1. **Begin login** — `POST /api/auth/passkey/login/begin` (no body required):

   ```bash
   curl -X POST http://localhost:8080/api/auth/passkey/login/begin
   ```

   Response:

   ```json
   {
     "session_id": "<session-id>",
     "options": { ... }   // PublicKeyCredentialRequestOptions — pass to navigator.credentials.get()
   }
   ```

2. **Finish login** — `POST /api/auth/passkey/login/finish?session_id=<session-id>` with the credential from `navigator.credentials.get()`:

   A `200 OK` response returns a JWT and user object (same shape as the password login response):

   ```json
   {
     "token": "<jwt>",
     "user": { ... }
   }
   ```

Both login endpoints are rate-limited (5 requests/second, burst 10) and do not require authentication.

### Managing passkey credentials

Each user can hold multiple passkeys — one per device is recommended.

#### List passkeys

```bash
curl http://localhost:8080/api/auth/passkey/credentials \
  -H "Authorization: Bearer <jwt>"
```

Returns an array of registered passkeys (IDs, names, AAGUIDs, and creation timestamps). Raw credential data is never returned.

#### Delete a passkey

```bash
curl -X DELETE http://localhost:8080/api/auth/passkey/credentials/<credential-id> \
  -H "Authorization: Bearer <jwt>"
```

Returns `204 No Content`. Deleting your last passkey does not affect password or OIDC login.

### Security model

| Property | Detail |
|----------|--------|
| Ceremony | WebAuthn Level 2 (FIDO2) using the [`go-webauthn/webauthn`](https://github.com/go-webauthn/webauthn) library |
| Challenge storage | Server-side, short-lived (5-minute TTL); expired challenges are pruned on each ceremony begin |
| Credential storage | Credential ID and serialized `webauthn.Credential` (public key + sign counter) stored in `passkey_credentials` table; private key never leaves the authenticator |
| Sign counter | Updated on every successful authentication to detect cloned authenticators |
| Discoverable login | Registration uses resident/discoverable credentials; no username hint is required at login time |
| Rate limiting | Login begin/finish endpoints share the global auth rate limiter (5 req/s, burst 10 per IP) |
| Audit trail | Passkey registration (`passkey.created`) and deletion (`passkey.deleted`) are recorded in the audit log |

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

Every API key begins with the prefix `bib_` followed by 40 lowercase hexadecimal characters (160 bits of entropy), for example:

```
bib_a3f2c8e1d074b651...
```

Only the `bib_` prefix and the first 12 hex characters (the *key prefix*) are stored in plaintext for identification in the UI. The remainder is stored only as a SHA-256 hash — the full key is never retrievable after creation.

> **Why SHA-256 and not bcrypt or Argon2?** API keys are 160-bit cryptographically random values, not user-chosen passwords. An attacker cannot brute-force 160 bits of randomness regardless of how fast or slow the hash function is, so the added computational cost of a password-hashing algorithm would provide no security benefit. SHA-256 is the appropriate choice for high-entropy tokens.

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

To manage keys programmatically, see the [API Keys endpoints](api/auth.md#api-keys) in the API reference.

### Security properties

| Property | Detail |
|----------|--------|
| Entropy | 160 bits (cryptographically random; meets [NIST SP 800-63B §5.1.2.1](https://pages.nist.gov/800-63-3/sp800-63b.html) ≥ 20-byte minimum for look-up secrets) |
| Storage | SHA-256 hash only — plaintext key is never persisted after creation (SHA-256 is appropriate because keys are 160-bit random values, not passwords) |
| Scope | Tied to the creating user; inherits that user's permissions |
| Transmission | HTTPS only in production; `Authorization` header only (cookies rejected) |
| Visibility | Key prefix (`bib_XXXXXXXXXXXX`) shown in the UI for identification |
| Last used | Timestamp updated lazily (throttled to at most once per 5 minutes) |

> **Keep API keys secret.** Anyone who obtains a key can authenticate as you. Revoke and reissue keys that may have been exposed.

---

## Rate Limiting

Signup, login, logout, password change (`PUT /api/auth/password`), all OIDC auth endpoints (`/api/auth/oidc/login`, `/api/auth/oidc/callback`, `/api/auth/oidc/link`), the KOReader/KOSync protocol endpoints (`/api/user/create`, `/api/user/auth`, `/api/syncs/progress`), and the SMTP test endpoint (`POST /api/config/smtp/test`) are protected by a per-IP token-bucket rate limiter:

| Parameter | Value |
|-----------|-------|
| Rate | 5 requests per second |
| Burst | 10 requests |
| Scope | Per client IP address |
| Response on limit | `429 Too Many Requests` |

### Client IP detection and `TRUSTED_PROXIES`

By default the rate limiter identifies clients by `RemoteAddr` (the direct TCP peer) and **ignores** the `X-Forwarded-For` header. This is the safe default when Biblioteka is exposed directly to the internet, because an attacker could inject arbitrary IP values through `X-Forwarded-For` to bypass per-IP limits.

When Biblioteka runs behind a reverse proxy (nginx, Caddy, Traefik, etc.), the proxy's IP would be the direct peer, causing all clients to share a single rate-limit bucket. Set the `TRUSTED_PROXIES` environment variable to unlock `X-Forwarded-For` processing:

```
TRUSTED_PROXIES=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

When `TRUSTED_PROXIES` is set, the rate limiter:

1. Checks whether `RemoteAddr` falls within a trusted CIDR range.
2. If it does, walks the `X-Forwarded-For` list from right to left, stopping at the rightmost IP that is **not** in a trusted range — that IP is used as the client identifier.
3. If `RemoteAddr` is not trusted, `X-Forwarded-For` is still ignored and `RemoteAddr` is used directly.

This "rightmost untrusted" strategy prevents a client from spoofing its IP by prepending values to the `X-Forwarded-For` header: only the IP added by the trusted proxy at the network boundary is honored.

> **Security note:** Only include CIDRs that correspond to your own reverse proxies in `TRUSTED_PROXIES`. Setting this to `0.0.0.0/0` effectively trusts every peer and allows any client to spoof its IP address, bypassing rate limiting entirely. See also the `TRUSTED_PROXIES` entry in [Environment variables](deployment.md#environment-variables).
