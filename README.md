# Fichemos

A media discovery and management platform that integrates with TMDB, Radarr, Sonarr, Prowlarr, and Seerr.

## Features

- **Movies & TV Shows**: Browse your Radarr/Sonarr libraries with TMDB metadata, posters, and watch provider info
- **Service Management**: Connect and manage Radarr, Sonarr, Prowlarr, and Seerr instances from the **Services** view
- **Dark Mode**: Choose light, dark, or auto (follows OS preference) theme from **Settings → Preferences**
- **Streaming Service Preferences**: Select your subscribed streaming services in **Settings → Streaming**. Provider icons for services you don't subscribe to are greyed out in collection views
- **Deep-Linked Settings**: Settings sub-pages (Account, Preferences, TMDB, OIDC/SSO, Streaming, Users) sync with the URL hash — refreshing the page preserves your current tab
- **OIDC/SSO Authentication**: Supports single sign-on via any OIDC-compatible provider (Keycloak, Auth0, etc.)
- **Admin Panel**: Manage users with indicators showing whether each account uses OIDC/SSO or local authentication

## Self-Hosting with Docker

Fichemos ships as a single Docker image with an embedded frontend. It supports both **SQLite** (default, zero-config) and **PostgreSQL** as database backends. Redis is required for background job processing.

### Quick Start with SQLite

SQLite is the default — no external database needed.

```bash
cp .env.sample .env
# Edit .env with your settings (JWT_SECRET, TMDB_API_KEY, etc.)

docker compose up -d
```

Data is persisted in a Docker volume (`fichemos-data`). The app is available at `http://localhost:8080`.

### Running with PostgreSQL

Use the Postgres override file to add a PostgreSQL container:

```bash
cp .env.sample .env
# Edit .env — you MUST set POSTGRES_PASSWORD before starting
```

> **Important:** Set a strong `POSTGRES_PASSWORD` in your `.env` file. The PostgreSQL container will not start without it.

```bash
docker compose -f docker-compose.yml -f docker-compose.postgres.yml up -d
```

To use an external PostgreSQL instance instead of the bundled container, set `DATABASE_URL` in your `.env` file:

```
DATABASE_URL=postgres://user:password@your-host:5432/fichemos
```

Then run with just the base compose file:

```bash
docker compose up -d
```

### Configuration

Copy `.env.sample` to `.env` and configure:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | _(empty, uses SQLite)_ | PostgreSQL connection string (`postgres://...`) |
| `REDIS_URL` | `redis://localhost:6379` | Redis URL (set automatically in Docker Compose) |
| `JWT_SECRET` | _(random)_ | Secret for signing JWT tokens. Set a stable value for production |
| `TMDB_API_KEY` | _(empty)_ | TMDB API key. Can also be configured via the admin panel |
| `SECURE_COOKIES` | `true` | Set to `false` if not using HTTPS |
| `POSTGRES_PASSWORD` | _(none)_ | **Required when using PostgreSQL.** Password for the bundled Postgres container |
| `OIDC_ISSUER_URL` | _(empty)_ | OIDC provider URL for SSO |
| `OIDC_CLIENT_ID` | _(empty)_ | OIDC client ID |
| `OIDC_CLIENT_SECRET` | _(empty)_ | OIDC client secret |
| `OIDC_REDIRECT_URI` | _(empty)_ | OIDC callback URL (e.g. `http://localhost:8080/api/auth/oidc/callback`) |
| `TELEMETRY_ENABLED` | `false` | Set to `true` to opt in to anonymous telemetry collection |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, or `error` |
| `LOG_FORMAT` | `json` | Log output format: `json` or `text` |


### First-Time Setup

The first user to sign up becomes the admin automatically. After starting the app, open `http://localhost:8080` in your browser and create your account — no pre-seeding required.

As admin you can:
- Set the TMDB API key from the admin panel (Settings → Config) if you prefer not to set it via an environment variable
- Add your *arr service connections via **Services** — click **Add Service**, choose the type (Radarr, Sonarr, Prowlarr, or Seerr), enter the base URL and API key, then save. Movies and TV shows will appear once at least one Radarr or Sonarr service is connected.
- Promote or demote other users via Settings → Admin

### Stopping and Removing

```bash
# Stop containers
docker compose down

# Stop and remove volumes (deletes all data)
docker compose down -v
```

### Health Check

The server exposes a health check endpoint at `/api/health`:

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

This is useful for container readiness probes and uptime monitoring.

## Development

### Prerequisites

- Go 1.25+
- Node.js 22+ with pnpm
- Redis

### Running Locally

```bash
# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Optional: verify Redis connectivity (local redis-cli or Docker fallback)
make redis-check

# Start both frontend and backend dev servers
make dev
```

This starts:
- Go backend on port 8080 (with hot-reload via air)
- Vite dev server with proxy to the backend

### Building

```bash
# Build everything (frontend + Go binary)
make build

# Run the compiled binary
make run
```
