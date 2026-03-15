# Deployment Guide

This guide covers deploying Biblioteka in production. For local development, see the [Quick Start](../README.md#quick-start) section in the README.

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
    depends_on:
      - redis
```

```bash
docker compose -f docker-compose.yml -f docker-compose.override.yml up -d
```

Database migrations still run on startup of the `server` container; the `worker` container skips the HTTP listener and begins processing jobs immediately.

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
- [ ] **PostgreSQL backups** — if using PostgreSQL, schedule regular `pg_dump` backups of the `biblioteka` database.
- [ ] **SQLite backups** — if using SQLite, back up the Docker volume (`biblioteka-data`) or the `*.db` file.
- [ ] **`TELEMETRY_ENABLED`** — leave unset (or set to `false`) to keep anonymous telemetry disabled (default). Set to `true` to enable it.

## Environment Variables

See the [Configuration](../README.md#configuration) table in the README for the full list of supported environment variables. The most security-critical settings for production are:

| Variable | Required | Notes |
|----------|----------|-------|
| `JWT_SECRET` | **Yes** | Random secret for signing JWTs; leaked tokens are valid until they expire |
| `SECURE_COOKIES` | **Yes** (set to `true`) | Prevents cookies being sent over HTTP |
| `DATABASE_URL` | No | Omit for SQLite; set to a PostgreSQL DSN for Postgres |
| `REDIS_URL` | No | Defaults to `redis://localhost:6379` |

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

The SQLite database is stored in a Docker named volume (`biblioteka-data`). To back it up:

```bash
# Copy the database file from the running container
docker compose cp biblioteka:/data/biblioteka.db ./biblioteka.db.bak
```

Or stop the container before copying for a guaranteed consistent snapshot:

```bash
docker compose stop biblioteka
docker compose cp biblioteka:/data/biblioteka.db ./biblioteka-$(date +%Y%m%d).db
docker compose start biblioteka
```

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

## Health Check

The `GET /api/health` endpoint returns `200 OK` with `{"status":"ok"}` and requires no authentication. Use it for liveness/readiness probes:

```bash
curl -sf http://localhost:8080/api/health
```
