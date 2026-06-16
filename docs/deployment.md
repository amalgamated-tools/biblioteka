# Deployment Guide

This guide covers deploying Biblioteka in production. For local development, see the [Quick Start](../README.md#quick-start) section in the README.

## Cloud Platform Guides

Want a one-click cloud install without managing servers? See the platform-specific guides:

- [Deploy to Fly.io](deploy-flyio.md) — persistent volumes, Upstash Redis, automatic TLS
- [Deploy to Render](deploy-render.md) — managed disks, built-in Redis, Blueprint IaC
- [Deploy to Railway](deploy-railway.md) — UI-driven setup, managed PostgreSQL option

## Prerequisites

| Component | Minimum version | Notes |
|-----------|----------------|-------|
| Docker + Docker Compose | Docker 24+ | Recommended deployment method |
| Redis | 7+ | Required for background job processing |
| PostgreSQL | 17+ | Optional — SQLite is used by default |
| Reverse proxy | — | Required for TLS and production traffic |

## Split-Process Deployment (Server + Worker)

By default, the binary starts both the HTTP server and the background worker in a single process. The `-mode` flag lets you run them as **separate containers or processes**, which is useful for:

- **Horizontal scaling** — run multiple `server` replicas behind a load balancer while keeping a single `worker` instance.
- **Resource isolation** — give job processing its own CPU/memory budget independent of request-serving capacity.
- **Container-per-role architectures** — common in Kubernetes and Docker Swarm deployments.

| Flag value | What starts |
|------------|-------------|
| `all` *(default)* | HTTP server + background worker |
| `server` | HTTP server only |
| `worker` | Background worker only |

> **Note:** Both the `server` and `worker` roles require Redis. The server uses Redis to enqueue jobs; the worker uses it to dequeue and process them.

### Example: Docker Compose split-process override

Add a `docker-compose.override.yml` to your deployment alongside the default `docker-compose.yml`:

```yaml
# docker-compose.override.yml — runs server and worker as separate services
services:
  biblioteka:
    command: ["-mode", "server"]

  biblioteka-worker:
    image: biblioteka          # same image as the server
    restart: unless-stopped
    command: ["-mode", "worker"]
    environment:
      DATABASE_URL: ${DATABASE_URL:-}
      REDIS_URL: ${REDIS_URL:-redis://redis:6379}
      JWT_SECRET: ${JWT_SECRET}
      LOG_LEVEL: ${LOG_LEVEL:-info}
      LOG_FORMAT: ${LOG_FORMAT:-json}
    depends_on:
      - redis
```

```bash
docker compose -f docker-compose.yml -f docker-compose.override.yml up -d
```

Database migrations still run on startup of the `server` container; the `worker` container skips the HTTP listener and begins processing jobs immediately.

> **Shared book storage:** In split-process deployments, both roles must see the same library path, but they use it differently. The `server` serves and downloads book files and needs read access to the shared book storage. The `worker` scans libraries, reads book files to extract metadata, processes imports and organize jobs, and writes sidecars when enabled — it needs read access at minimum, and write access if you use workflows that reorganize files or create sidecar files. If your books are on a bind-mounted volume or named Docker volume, add the same volume mount to the `biblioteka-worker` service definition; otherwise scan and process jobs will fail because the worker cannot read the underlying files. A read-only mount can be appropriate for the `server` in serving-only setups, but not for a `worker` that must reorganize files or write sidecars.

## Container Images

Pre-built multi-arch container images (`linux/amd64`, `linux/arm64`) are published to the GitHub Container Registry (GHCR) automatically by the **Build Container** CI workflow:

- On every push to `main` → tagged `edge` and `sha-<short-sha>`
- On every published release → tagged `latest`, `v<major>.<minor>`, and `v<major>.<minor>.<patch>`

| Tag | Example | Use for |
|-----|---------|---------|
| `latest` | `ghcr.io/amalgamated-tools/biblioteka:latest` | Production (most recent release) |
| `edge` | `ghcr.io/amalgamated-tools/biblioteka:edge` | Bleeding-edge / staging (built from `main`) |
| `v<major>.<minor>` | `ghcr.io/amalgamated-tools/biblioteka:v0.1` | Minor-version pin |
| `v<major>.<minor>.<patch>` | `ghcr.io/amalgamated-tools/biblioteka:v0.1.0` | Exact-version pin |
| `sha-<short-sha>` | `ghcr.io/amalgamated-tools/biblioteka:sha-6fcf024` | Reproducing a specific build |

### Using a pre-built image with Docker Compose

The default `docker-compose.yml` builds the image locally (`build: .`). To use a pre-built image instead, create a `docker-compose.override.yml` alongside it:

```yaml
# docker-compose.override.yml — use GHCR pre-built image instead of local build
services:
  biblioteka:
    image: ghcr.io/amalgamated-tools/biblioteka:latest
    build: null   # disable local build
```

Then run as normal:

```bash
export JWT_SECRET=$(openssl rand -hex 32)
export SECURE_COOKIES=true

docker compose up -d
```

> **Note:** If you receive a `403 Forbidden` error when pushing or pulling the image, the GHCR package may need its permissions configured. Go to **https://github.com/orgs/amalgamated-tools/packages**, find the `biblioteka` package, and check that the repository has read access. For CI pushes, ensure the workflow has `packages: write` permission or a `GHCR_TOKEN` secret is set.

## Docker Compose Deployments

### SQLite + Redis (simplest)

Use the default `docker-compose.yml`, which bundles the application with Redis and stores SQLite data in a named Docker volume.

```bash
# Set a strong JWT secret before starting
export JWT_SECRET=$(openssl rand -hex 32)
export SECURE_COOKIES=true

docker compose up -d
```

The application is available at `http://localhost:8080`. Place a reverse proxy in front to handle TLS (see [Reverse Proxy Setup](#reverse-proxy-setup) below).

#### SQLite connection settings

Biblioteka automatically applies the following PRAGMAs to the SQLite connection at startup:

| PRAGMA | Value | Effect |
|--------|-------|--------|
| `journal_mode` | `WAL` | Write-ahead logging; readers do not block writers |
| `synchronous` | `NORMAL` | Fsync only at WAL checkpoint; faster than `FULL` but recent transactions may be lost on OS crash or power loss (acceptable for personal-library use) |
| `foreign_keys` | `ON` | Enforces referential integrity on every write |
| `temp_store` | `MEMORY` | Sort buffers (temp B-trees for `ORDER BY`) are held in RAM instead of OS temp files |
| `cache_size` | `-16384` (16 MB) | Per-connection page cache; reduces disk reads for frequently accessed pages |

The connection pool is capped to a single connection so that the per-connection PRAGMAs (`foreign_keys`, `synchronous`, `temp_store`, `cache_size`) are applied consistently to every query — `journal_mode=WAL` is a persistent, database-level setting and does not need this protection. The trade-off is serialized reads: WAL normally allows multiple concurrent readers, but a single-connection pool eliminates that benefit. For a personal-library workload this is sufficient — requests queue briefly at the database layer rather than competing for connections. SQLite serializes writers regardless of pool size, so this adds no meaningful write-latency penalty.

### PostgreSQL + Redis

Merge `docker-compose.yml` with the PostgreSQL overlay. This adds a `postgres` service and configures the application to use it.

```bash
export JWT_SECRET=$(openssl rand -hex 32)
export POSTGRES_PASSWORD=$(openssl rand -hex 24)
export SECURE_COOKIES=true

docker compose -f docker-compose.yml -f docker-compose.postgres.yml up -d
```

### Persistent book storage

To give the container access to books stored on the host, add a bind mount to the `biblioteka` service in your `docker-compose.override.yml`:

```yaml
# docker-compose.override.yml
services:
  biblioteka:
    volumes:
      - /srv/books:/books:ro   # host path : container path
```

Then configure library paths in the application's UI to point to `/books/...`.

## Production Checklist

Before going live, verify each item:

- [ ] **`JWT_SECRET`** — set to a long random string (`openssl rand -hex 32`). The default `change-me-in-production` value must not be used.
- [ ] **`SECURE_COOKIES=true`** — ensures session cookies are only sent over HTTPS. Only set this to `false` for local HTTP development.
- [ ] **TLS** — terminate TLS at a reverse proxy (nginx, Caddy, Traefik). Do not expose port 8080 directly to the internet.
- [ ] **Redis persistence** — configure Redis with at least `appendonly yes` if background job durability matters to you.
- [ ] **Redis eviction policy** — set `maxmemory-policy noeviction`. Other policies may silently evict queued jobs under memory pressure; `noeviction` surfaces the problem as an error instead.
- [ ] **Redis authentication and TLS** — For deployments where Redis is reachable over a network, configure both authentication and TLS. Use the full URL form: `REDIS_URL=rediss://:yourpassword@host:6379` for TLS with password auth (recommended for networked deployments), `REDIS_URL=redis://:yourpassword@host:6379` for password auth without TLS, or `REDIS_URL=rediss://host:6379` for TLS without a password. The default `redis://localhost:6379` is appropriate only for single-host, trusted-network deployments.
- [ ] **PostgreSQL backups** — if using PostgreSQL, schedule regular `pg_dump` backups of the `biblioteka` database.
- [ ] **SQLite backups** — if using SQLite, back up the Docker volume (`biblioteka-data`) or the `*.db` file.
- [ ] **`TELEMETRY_ENABLED`** — leave unset (or set to `false`) to keep anonymous telemetry disabled (default). Set to `true` to enable it.
- [ ] **`TRUSTED_PROXIES`** — set to your reverse-proxy CIDR(s) if behind nginx/Caddy/Traefik. Leave unset if deploying without a proxy (direct exposure).
- [ ] **Passkeys** — if using passkeys in production, set `WEBAUTHN_RP_ID` to your production domain (e.g. `books.example.com`) and set `WEBAUTHN_RP_ORIGINS` to the matching HTTPS origin (e.g. `https://books.example.com`). Leaving both at their `localhost` defaults makes passkeys non-functional in production — ceremonies will silently fail while the UI still shows passkeys as available. See [Authentication → Passkeys](authentication.md#passkeys-webauthn).
- [ ] **Remove `INITIAL_ADMIN_*`** — unset `INITIAL_ADMIN_EMAIL`, `INITIAL_ADMIN_PASSWORD`, and `INITIAL_ADMIN_NAME` once the first admin account has been confirmed. These variables have no effect once any user exists, but leaving them set adds unnecessary secret surface.
- [ ] **SMTP** — if you want email delivery, configure the variables below (or use the admin UI under *Settings → Email*). Environment variables take precedence over UI settings when `SMTP_HOST` is set; unset `SMTP_HOST` to switch back to UI-managed config.

  | Variable | Default | Notes |
  |----------|---------|-------|
  | `SMTP_HOST` | *(empty)* | Hostname or IP of the SMTP server. When set, all `SMTP_*` env vars override database-stored settings (UI becomes read-only) |
  | `SMTP_PORT` | `587` | Server port |
  | `SMTP_TLS` | `starttls` | TLS mode: `none`, `starttls`, or `tls`. Authenticated SMTP (`SMTP_USERNAME` set) without TLS is only permitted for loopback addresses |
  | `SMTP_USERNAME` | *(empty)* | Auth username; leave empty for unauthenticated relay |
  | `SMTP_PASSWORD` | *(empty)* | Auth password; required when `SMTP_USERNAME` is set |
  | `SMTP_FROM` | *(empty)* | Sender address for outgoing mail; accepts a bare address (e.g. `biblioteka@example.com`) or RFC 5322 display-name format (e.g. `"Biblioteka" <biblioteka@example.com>`); required when `SMTP_HOST` is set |

  These can be passed via a `.env` file or exported in your shell before running `docker compose up`.

## Environment Variables

See the [Configuration](../README.md#configuration) table in the README for the full list of supported environment variables. The most security-critical settings for production are:

| Variable | Required | Notes |
|----------|----------|-------|
| `JWT_SECRET` | **Yes** | Signs JWTs (24 h validity) and derives the AES-256-GCM key for sensitive DB-stored settings (SMTP password, OIDC client secret). Minimum **32 bytes required** — the server refuses to start if a shorter value is provided. **Rotating invalidates all active sessions and makes previously-encrypted DB settings unreadable.** See [JWT Secret Rotation](#jwt-secret-rotation) |
| `SECURE_COOKIES` | **Yes** (set to `true`) | Prevents cookies being sent over HTTP |
| `DATABASE_URL` | No | Omit for SQLite; set to a PostgreSQL DSN for Postgres |
| `REDIS_URL` | No | Defaults to `redis://localhost:6379` |
| `TRUSTED_PROXIES` | No | Comma-separated CIDR ranges of trusted reverse proxies (e.g. `10.0.0.0/8,172.16.0.0/12`). When set, the rate limiter uses the rightmost non-trusted IP from `X-Forwarded-For`. When unset, `X-Forwarded-For` is ignored and `RemoteAddr` is used directly. |
| `INITIAL_ADMIN_EMAIL` | No | Email address of the first admin user to create on startup. Only takes effect when the users table is empty **and** `INITIAL_ADMIN_PASSWORD` is also set. Idempotent — if a user already exists the variable is ignored. |
| `INITIAL_ADMIN_PASSWORD` | No | Password for the bootstrap admin user. Stored as a bcrypt hash. Requires `INITIAL_ADMIN_EMAIL`. Has no effect once any user exists. |
| `INITIAL_ADMIN_NAME` | No | Display name for the bootstrap admin. Defaults to `"Admin"` when omitted. Only used alongside `INITIAL_ADMIN_EMAIL` + `INITIAL_ADMIN_PASSWORD`. |

### Bootstrap Admin User

For automated, headless, or infrastructure-as-code deployments where signing up interactively is not possible, set `INITIAL_ADMIN_EMAIL` and `INITIAL_ADMIN_PASSWORD` to pre-create the first admin account. Consider also setting `DISABLE_SIGNUP=true` to prevent public self-registration once the first account exists.

```bash
INITIAL_ADMIN_EMAIL=admin@example.com
INITIAL_ADMIN_PASSWORD=<your-bootstrap-password>
INITIAL_ADMIN_NAME=Library Admin   # optional; defaults to "Admin"
```

**Behavior:**

- Only runs when the `users` table is **empty**. If any user already exists, the variables are ignored and startup logs may include an informational message that initial admin seeding was skipped.
- The operation is idempotent — if two instances start concurrently and both attempt to seed, the second duplicate insert is treated as a no-op.
- The password is stored as a bcrypt hash; the plaintext value is never persisted.
- The created account has admin privileges (same as the first account created via the sign-up form).

> **Security note:** These variables are start-up secrets. Remove them from your environment or secrets manager once the initial admin account has been verified. The variables are no longer needed after at least one user exists, and leaving them set has no effect (but adds unnecessary secret surface).

### JWT Secret Rotation

JWT tokens are **stateless** — the server does not maintain a token revocation list. Rotating `JWT_SECRET` and redeploying is the only way to invalidate all active sessions. In deployments with multiple replicas or rolling updates, sessions are not fully invalidated until all running instances have been restarted with the new secret.

**When to rotate:** when the secret is accidentally exposed (version control, logs), when a server is decommissioned, or on a regular security schedule.

**How to rotate:**

```bash
# 1. Generate a new secret
NEW_SECRET=$(openssl rand -hex 32)

# 2. Update your environment (docker-compose.override.yml, .env, secrets manager, etc.)
#    Replace the JWT_SECRET value with $NEW_SECRET

# 3. Redeploy
docker compose up -d --force-recreate biblioteka

# In split-process deployments, also recreate the worker to keep the environment consistent:
docker compose up -d --force-recreate biblioteka-worker
```

**Consequences of rotation:**
- All active browser sessions are immediately invalidated — every logged-in user will appear logged out and be shown the login screen on their next request.
- All API clients using JWT Bearer tokens must re-authenticate to obtain a new token.
- [API keys](../README.md#api-keys) (`bib_…`) and **Kobo sync tokens** are **not** affected — they authenticate via independent mechanisms and remain valid after rotation.
- **Encrypted settings become unreadable.** Re-enter the SMTP password and OIDC client secret in the admin UI (*Settings → Email / SMTP* and *Settings → OIDC / SSO*) after rotation.

> **OIDC sessions:** Rotating `JWT_SECRET` does not invalidate users' sessions with their identity provider. Users are redirected through the OIDC login flow, but most providers will silently reuse an existing IdP session without prompting for credentials again. Any OIDC login or account-link flow already in progress at rotation time will fail and must be restarted (OIDC `state` validation is derived from `JWT_SECRET`).

## Reverse Proxy Setup

Terminate TLS at a reverse proxy and forward traffic to port `8080`.

### Caddy (recommended)

```caddyfile
books.example.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically provisions TLS certificates via Let's Encrypt.

### nginx

```nginx
server {
    listen 443 ssl;
    server_name books.example.com;

    ssl_certificate     /etc/letsencrypt/live/books.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/books.example.com/privkey.pem;

    location / {
        proxy_pass         http://localhost:8080;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
    }
}
```

> **Asynqmon access:** The background-job dashboard at `/asynqmon/` accepts authentication via the `Authorization: Bearer <JWT>` header **or** the `biblioteka_token` session cookie. Admin users who are already signed in through the web UI can navigate directly to `/asynqmon/` in their browser — the login cookie is sent automatically. API clients should supply the `Authorization` header.

## Backups

### SQLite

The SQLite database is stored in a Docker named volume (`biblioteka-data`).

**Recommended — stop all SQLite writers for a guaranteed consistent snapshot:**

```bash
# Single-container deployment:
docker compose stop biblioteka
docker compose cp biblioteka:/data/biblioteka.db ./biblioteka-$(date +%Y%m%d).db
docker compose start biblioteka
```

For split-process deployments, stop and start both services instead: `docker compose stop biblioteka biblioteka-worker` and `docker compose start biblioteka biblioteka-worker`.

**Hot backup (container running, best effort):** Biblioteka uses SQLite in [WAL mode](https://www.sqlite.org/wal.html). Copying only the `.db` file while the server is running may produce an **incomplete backup** — committed writes that have not yet been checkpointed live in `.db-wal`, not in `.db`. For the best chance of a usable hot backup, copy `.db-wal` first, then `.db`:

```bash
# WAL-mode hot backup — copy WAL first, then main database
BACKUP_DATE=$(date +%Y%m%d)
docker compose cp biblioteka:/data/biblioteka.db-wal ./biblioteka-${BACKUP_DATE}.db-wal \
  && echo "WAL captured" \
  || { rm -f ./biblioteka-${BACKUP_DATE}.db-wal; echo "WAL not present — database is fully checkpointed; .db alone is sufficient"; }
docker compose cp biblioteka:/data/biblioteka.db ./biblioteka-${BACKUP_DATE}.db
```

Keep both files in the same directory. To restore, place them together (e.g., `biblioteka-20260101.db` and `biblioteka-20260101.db-wal`). If `.db-wal` does not exist (the database has been fully checkpointed — normal after a clean shutdown), the `.db` file alone is sufficient. SQLite recreates `.db-shm` automatically.

> **After restoring from backup:** If you ran `VACUUM` on the restored database file outside of Biblioteka, the full-text search index may be corrupt. Start the server once normally — it runs a startup integrity check and rebuilds the FTS index automatically if needed. See [Search Index Maintenance](administration.md#search-index-maintenance-sqlite).

### PostgreSQL

```bash
docker compose -f docker-compose.yml -f docker-compose.postgres.yml exec -T postgres \
  pg_dump -U biblioteka biblioteka | gzip > biblioteka-$(date +%Y%m%d).sql.gz
```

Restore:

```bash
gunzip < biblioteka-20260314.sql.gz | \
  docker compose -f docker-compose.yml -f docker-compose.postgres.yml exec -T postgres \
  psql -U biblioteka biblioteka
```

## Upgrading

1. Pull the new image (or rebuild from source):
   ```bash
   docker compose pull          # if using a registry image
   # — or —
   docker compose build         # if building locally
   ```
2. Restart the service:
   ```bash
   docker compose up -d
   ```

Database migrations run automatically on startup — no separate migration step is needed.

> **In-flight background jobs:** `docker compose up -d` issues a graceful stop to the running container before starting the new one. Any Asynq job actively processing at that moment will not complete cleanly. Asynq re-enqueues the interrupted job once the worker reconnects to Redis, using the configured retry budget (`DefaultMaxRetry = 5`). Long-running jobs such as large library scans may appear in the Asynqmon *Retrying* tab immediately after an upgrade — this is expected behaviour and not an error.

> **Split-process deployments:** When running separate `server` and `worker` containers, restart the `server` container first. The `server` runs database migrations on startup; restart the `worker` only after the `server` has started and those migrations have completed successfully. Restarting both containers simultaneously is usually safe for minor upgrades, but sequential restart (`server` → `worker`) minimizes the risk of a job handler running against a partially-migrated schema.

## HTTP Security Headers

Biblioteka sets the following HTTP security headers on every response via the `NewSecurityHeadersMiddleware` middleware:

| Header | Value | Purpose |
|--------|-------|---------|
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' 'sha256-fH8pmaGT8bEGA0OitMqoXdy+W8xbN89w8ghrDCdlrwA='; style-src 'self' https://fonts.googleapis.com; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data: https://fonts.gstatic.com;` | Restricts external resource origins; neither `script-src` nor `style-src` uses `'unsafe-inline'` — `script-src` uses a SHA-256 hash of the theme bootstrap script; `style-src` omits `'unsafe-inline'` because Svelte applies dynamic styles via CSSOM property calls, which are not governed by `style-src` |
| `X-Content-Type-Options` | `nosniff` | Prevents MIME-type sniffing |
| `X-Frame-Options` | `DENY` | Blocks framing (clickjacking protection) |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Limits referrer information sent in cross-origin requests |
| `Permissions-Policy` | `camera=(), microphone=(), geolocation=()` | Disables browser feature access not needed by the application |
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains` | Enforces HTTPS for two years; set only when `SECURE_COOKIES=true` (HTTPS deployments) |

The CSP permits the frontend's inline theme bootstrap script (via SHA-256 hash) and Google Fonts. Individual route handlers may override the CSP for their specific use case.

> **Note:** If the inline `<script>` block in `frontend/index.html` is ever modified (even whitespace), the SHA-256 hash in `internal/handlers/middleware/security_headers.go` must be recomputed. See the comment in that file for the one-line regeneration command.

No additional reverse proxy configuration is required — the application server sets these headers directly.

---

## HTTP Response Compression

Biblioteka automatically compresses HTTP responses using gzip via `GzipMiddleware`. Compression is applied transparently on the server side with no configuration required.

**Conditions for compression:**

1. The client signals support for gzip via `Accept-Encoding` (i.e., includes `gzip` with a non-zero `q` value).
2. The response `Content-Type` is a known text-based format.

**Compressed content types:**

| Content-Type | Examples |
|---|---|
| `application/json` | API responses |
| `application/xml`, `application/atom+xml` | OPDS feeds |
| `application/javascript`, `text/javascript` | Frontend assets |
| `text/css` | Stylesheets |
| `text/html` | HTML pages |
| `text/plain` | Plain-text responses |
| `text/xml` | XML documents |
| `image/svg+xml` | SVG images |

Binary responses (EPUB, MOBI, PDF, JPEG, PNG) and streaming responses (Server-Sent Events) are passed through uncompressed.

**Cache correctness:** Every response (compressed or not) includes `Vary: Accept-Encoding` so that caches store separate entries for compressed and uncompressed clients.

**Range requests:** Compression is skipped for requests that include a `Range` header to preserve `Content-Range` semantics (e.g. partial file downloads).

> **Reverse proxy note:** If your reverse proxy (Caddy, nginx) also compresses responses, you can either disable its compression for upstream responses or leave both active — Biblioteka will only compress when the client explicitly requests gzip, and the proxy will see `Content-Encoding: gzip` on already-compressed responses and forward them unchanged.

---

## Static Asset Caching

Biblioteka uses Vite's content-hashing for frontend assets (JavaScript, CSS, fonts). Every compiled asset under `/assets/` has a content-derived hash in its filename (e.g., `index-DfzxFbzN.js`), meaning a changed file always gets a new URL.

The embedded file server sets `Cache-Control` headers automatically based on path:

| Path pattern | `Cache-Control` value | Effect |
|---|---|---|
| `/assets/*` (content-hashed files) | `public, max-age=31536000, immutable` | Cached by browser and CDN for one year; not revalidated during `max-age` |
| `/` and `/index.html` | `no-cache` | Always revalidated; ensures the browser fetches new asset hashes after a deploy |
| Other static files | *(ETag / Last-Modified only)* | Standard conditional-request revalidation |

The `immutable` directive tells browsers not to issue conditional requests for `/assets/` files even within the `max-age` window. Because the filename changes on every build, stale assets are never served — the browser will request the new filename as soon as it reloads the revalidated `index.html`.

> **Reverse proxy note:** If your reverse proxy also caches responses (e.g., Nginx proxy_cache, Varnish, a CDN), the `Cache-Control: public, max-age=31536000, immutable` header on `/assets/*` responses is safe to honour — those filenames are permanent. For `index.html` (`no-cache`), ensure the proxy forwards the header to clients unchanged and does not apply its own long-lived cache to it.

---

## Health Check

The `GET /api/health` endpoint returns `200 OK` with `{"status":"ok"}` and requires no authentication. Use it for liveness/readiness probes:

```bash
curl -sf http://localhost:8080/api/health
```
