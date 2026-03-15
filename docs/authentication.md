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

After the callback, the server sets a `biblioteka_token` session cookie and redirects to `/?oidc_login=1`. The frontend then calls `GET /api/auth/me` using the cookie to retrieve the user object.

**Scopes requested:** `openid email profile`

**Required claims:** `sub` (subject), `email`. The `name` claim is used when available; otherwise `email` is used as the display name.

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

If a user signs in via OIDC and no existing account has that `sub` claim, the server looks up by `email`. If an account with that email exists, the OIDC subject is automatically linked to it. Otherwise, a new account is created.

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

## Rate Limiting

Signup, login, and all OIDC endpoints are protected by a per-IP token-bucket rate limiter:

| Parameter | Value |
|-----------|-------|
| Rate | 5 requests per second |
| Burst | 10 requests |
| Scope | Per client IP address |
| Response on limit | `429 Too Many Requests` |
