# Biblioteka

A self-hosted personal book library manager. Scan local files, extract metadata, and browse your e-book and physical book collection through a clean web interface.

## Features

- **Multi-format support** – EPUB, MOBI, AZW3, and PDF
- **Automatic metadata extraction** – title, author, ISBN, and more extracted on import
- **Library organisation** – group books into multiple named libraries with configurable file-system paths
- **Author & series tracking** – browse by author or series, with position numbers within each series
- **User authentication** – JWT-based login, optional OpenID Connect (OIDC/SSO)
- **Background processing** – Redis-backed job queue scans paths and processes files asynchronously
- **Two database backends** – SQLite (zero-config, default) or PostgreSQL
- **Single binary** – Go backend embeds the Svelte frontend; one executable to deploy

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, `net/http` |
| Frontend | Svelte 5, TypeScript, Tailwind CSS 3.4, Vite 7 |
| Database | SQLite (default) · PostgreSQL |
| Job queue | asynq (Redis) |
| Auth | JWT · OIDC |
| Observability | OpenTelemetry (tracing + structured logging) |

## Quick Start

### Docker Compose (recommended)

```bash
# SQLite + Redis (simplest setup)
docker compose up -d
```

The application is available at <http://localhost:8080>.

For PostgreSQL, use the alternate compose file:

```bash
docker compose -f docker-compose.postgres.yml up -d
```

### Build from Source

**Prerequisites:** Go 1.25+, Node.js 22+, pnpm, Redis

```bash
git clone https://github.com/amalgamated-tools/biblioteka.git
cd biblioteka

# Build frontend then compile Go binary
make build

# Run
make run
```

### Development Mode (hot-reload)

```bash
# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Check Redis is available
make redis-check

# Start backend (air) + frontend (Vite) together
make dev
```

The Go API runs on `http://localhost:8080`; Vite proxies `/api` requests to it.

## Configuration

Copy `.env.sample` to `.env` and adjust as needed:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | *(empty – SQLite)* | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `JWT_SECRET` | — | **Required in production** – random secret for signing tokens |
| `SECURE_COOKIES` | `false` | Set `true` behind HTTPS |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `OIDC_ISSUER_URL` | *(empty)* | OIDC provider issuer URL |
| `OIDC_CLIENT_ID` | *(empty)* | OIDC client ID |
| `OIDC_CLIENT_SECRET` | *(empty)* | OIDC client secret |
| `OIDC_REDIRECT_URI` | `http://localhost:8080/api/auth/oidc/callback` | OIDC callback URL |
| `TELEMETRY_ENABLED` | `false` | Enable OpenTelemetry export |

## Database Migrations

Migrations in `db/migrations/{sqlite,postgres}/` run automatically on startup. No separate migration command is needed.

To add a migration, create a file named `YYYYMMDDHHMMSS_description.sql` in the appropriate directory using [dbmate](https://github.com/amacneil/dbmate) format:

```sql
-- migrate:up
CREATE TABLE ...;

-- migrate:down
DROP TABLE ...;
```

## Testing

```bash
# Go tests
go test ./...

# Frontend unit tests
cd frontend && pnpm run test

# Frontend type checking
cd frontend && pnpm run check

# Frontend lint
cd frontend && pnpm run lint
```

## Project Layout

```
cmd/
  server/          Main server entry point
  cli/             CLI tool for standalone metadata extraction
internal/
  auth/            JWT, OIDC, rate-limiting, middleware
  db/              Database layer (SQLite/PostgreSQL), migrations, CRUD
  handlers/        HTTP request handlers (books, authors, series, libraries, auth)
  jobs/            Background job handlers (scan path, process file)
  metadata/        EPUB/MOBI/PDF metadata extraction
  server/          Route registration, middleware setup, embedded frontend
  worker/          asynq worker setup
  otel/            Logging and tracing
frontend/          Svelte 5 SPA
db/
  migrations/      sqlite/ and postgres/ SQL migration files
  schema.sql       Reference schema
script/            Build and release helper scripts
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, code conventions, and how to submit a pull request.
