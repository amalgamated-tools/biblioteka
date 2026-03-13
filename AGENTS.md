# Agents Guide for Fichemos

Fichemos is a self-hosted media management dashboard that aggregates *arr services (Radarr, Sonarr, Prowlarr, Seerr) into a single interface.

## Architecture

- **Backend**: Go HTTP server (`cmd/server/main.go`) using only the standard library `net/http` — no router framework.
- **Frontend**: Svelte 5 SPA (`frontend/`) with TypeScript, Tailwind CSS, and Vite. Built output is embedded into the Go binary via `//go:embed`.
- **Database**: SQLite (default, via `modernc.org/sqlite`, pure Go) or PostgreSQL (via `pgx/v5`). Dialect is selected by the `DATABASE_URL` env var.
- **API clients**: Generated from OpenAPI specs using `oapi-codegen`. Generated files live in `pkg/{radarr,sonarr,prowlarr,seerr,tmdb,plex}/client.gen.go` — never edit these by hand.
- **Auth**: JWT-based (Bearer tokens). `internal/auth/middleware.go` extracts user ID into request context via `auth.UserIDFromContext(ctx)`.
- **Background jobs**: Redis-based via `asynq` (`internal/worker/`).

## Build & Run

```bash
make redis-check      # Verify Redis connectivity (local redis-cli or Docker fallback)
make dev              # Frontend + backend dev servers (goreman + air hot-reload)
make build            # Build frontend then compile Go binary
make run              # Build and run production binary
make clean            # Clean build artifacts
```

## Testing

```bash
go test ./...                  # Run all Go tests
go test ./internal/db/         # Run tests for a single package
go test -run TestName ./...    # Run a specific test
```

## Frontend (from `frontend/`)

```bash
pnpm install          # Install dependencies
pnpm run dev          # Vite dev server (proxies /api to localhost:8080)
pnpm run build        # Production build → internal/server/dist/
pnpm run check        # svelte-check type checking
pnpm run lint         # ESLint
pnpm run format       # Prettier
```

## Code Generation

```bash
go generate ./...              # Regenerate all API clients from OpenAPI specs
scripts/generate.sh            # Download latest OpenAPI specs + regenerate clients
```

OpenAPI config files are in `openapi/*.yaml`; specs are in `openapi/*.json`.

> **Note:** The Plex spec is not available as a static download URL. `scripts/generate.sh` runs `scripts/download_plex_spec.py` to scrape and patch the spec from `developer.plex.tv/pms/` before regenerating the client.

## Database

### Migrations

Migrations use dbmate format (`-- migrate:up` / `-- migrate:down`) in `db/migrations/`. Separate directories exist for each dialect: `db/migrations/sqlite/` and `db/migrations/postgres/`. Files are named with timestamps: `YYYYMMDDHHMMSS_description.sql`. Migrations run automatically on startup.

### Schema

Full schema is in `db/schema.sql`. Key tables: `users`, `media_services`, `movies`, `tv_series`, `user_movies`, `user_tv_series`, `watch_providers`, `settings`.

## Key Conventions

- **Routing**: Routes are registered manually on `http.ServeMux` in `internal/server/server.go`. API routes are prefixed with `/api/`. Handlers dispatch on HTTP method internally.
- **Handler structure**: Each domain has a handler struct (e.g., `AuthHandler`, `ArrServiceHandler`) that holds a `*db.DB` and dependencies. Handlers live in `internal/handlers/`.
- **JSON responses**: Use `writeJSON(w, status, data)` and `writeError(w, status, message)` from `internal/handlers/helpers.go`.
- **Path parameters**: Extracted manually via `extractPathID(path, prefix)` from `internal/handlers/helpers.go` — there is no router with named params.
- **Admin-only endpoints**: Guard handlers with the handler's `requireAdmin(w, r) bool` method. It writes the error response and returns `false` when the user is not an admin, so callers just check the return value and `return`.
- **User scoping**: All data queries include `user_id` to enforce per-user data isolation.
- **Logging**: Use `log/slog` for structured logging. The `log.Print*` family is forbidden by the linter (`.golangci.yml`).
- **Generated code**: Never edit `client.gen.go` files. Regenerate with `go generate ./...`.

## Project Layout

```
cmd/server/          # Main entry point
internal/
  auth/              # JWT, rate limiting, middleware
  db/                # Database abstraction (SQLite/Postgres), CRUD operations
  handlers/          # HTTP request handlers
  jobs/              # Background job definitions
  server/            # HTTP server setup, routing, embedded frontend
  worker/            # asynq-based job processing
  otel/              # Logging and tracing setup
pkg/
  tmdb/              # TMDB API client (generated + wrapper)
  radarr/            # Generated Radarr API client
  sonarr/            # Generated Sonarr API client
  prowlarr/          # Generated Prowlarr API client
  seerr/             # Generated Seerr API client
  plex/              # Generated Plex API client (scraped from developer.plex.tv)
frontend/            # Svelte 5 SPA
db/
  schema.sql         # Full database schema
  migrations/        # sqlite/ and postgres/ migration directories
openapi/             # OpenAPI specs and config for code generation
```

## Environment Variables

- `PORT` — Server port (default 8080)
- `DATABASE_URL` — PostgreSQL connection string; omit for SQLite
- `REDIS_URL` — Redis URL for background jobs
- `TMDB_API_KEY` — The Movie Database API key
- `JWT_SECRET` — Secret for JWT signing (random if unset)
- `SECURE_COOKIES` — Set to `false` to disable secure cookies (required for local HTTP dev)
- `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI` — Optional OIDC auth
- `LOG_LEVEL` — Log verbosity: `debug`, `info` (default), `warn`, `error`
- `LOG_FORMAT` — Log output format: `json` (default) or `text`
- `TELEMETRY_ENABLED` — Set to `true` to opt in to anonymous telemetry (disabled by default)

## Agentic Workflows

Fichemos uses several GitHub Copilot-powered agentic workflows (defined in `.github/workflows/*.md`, compiled to `*.lock.yml` via `gh aw compile`). Do **not** edit the `.lock.yml` files by hand — edit the corresponding `.md` file and recompile.

| Workflow | Trigger | Description |
|---|---|---|
| `repo-assist` | Daily + `workflow_dispatch` | Repository assistant: triages issues, creates fix PRs, manages labels, nudges stale PRs, and welcomes new contributors. Requires the `COPILOT_GITHUB_TOKEN` secret. |
| `daily-repo-status` | Daily + `workflow_dispatch` | Posts an upbeat daily GitHub issue summarising recent activity, progress, and actionable next steps for maintainers. |
| `daily-activity-report` | Daily on weekdays | Posts a structured GitHub issue covering new issues, merged PRs, and open blockers for the last 24 hours (weekday Mondays cover since Friday). |
| `daily-doc-updater` | Daily + `workflow_dispatch` | Scans merged PRs and commits from the last 24 hours, identifies undocumented features, and opens a PR with documentation updates. |
| `update-docs` | Push to `main` + `workflow_dispatch` | Mirrors every push to `main` with a draft documentation PR, keeping docs in sync with code changes following documentation-as-code principles. |
| `code-simplifier` | Daily | Analyses code changed in the last 24 hours and opens a PR with targeted simplifications (clarity, consistency, maintainability) — skipped if a `[code-simplifier]` PR is already open. |

### Shared Workflow Fragments

Reusable prompt fragments used by some workflows live in `.github/workflows/shared/`:

- `formatting.md` — Output formatting conventions
- `reporting.md` — Reporting style and structure

## GitHub Actions Secrets

The following repository secrets are required for automated workflows:

| Secret | Purpose | Required |
|---|---|---|
| `COPILOT_GITHUB_TOKEN` | Authenticates the GitHub Copilot CLI used by the Repo Assist agentic workflow | Required for Repo Assist |
| `GH_AW_GITHUB_MCP_SERVER_TOKEN` | GitHub token for the MCP server used by agentic workflows (falls back to `GH_AW_GITHUB_TOKEN` or `GITHUB_TOKEN`) | Optional |
| `GH_AW_GITHUB_TOKEN` | Fallback GitHub token for agentic workflow MCP server operations | Optional |

> **Note:** If the `COPILOT_GITHUB_TOKEN` secret is not set or is invalid, the Repo Assist workflow will fail with "No authentication information found." Set this secret in **Settings → Secrets and variables → Actions** with a GitHub personal access token or fine-grained PAT that has the `copilot` scope.

## Agentic Workflows

Automated AI-driven workflows run on a schedule and are defined in `.github/workflows/*.md`. Lock files (`*.lock.yml`) are compiled from the markdown definitions via `gh aw compile` — do not edit lock files manually.

| Workflow | Trigger | Description |
|---|---|---|
| `repo-assist` | Issue/PR/discussion comments | AI assistant that responds to `@copilot` mentions |
| `daily-activity-report` | Daily (weekdays) | Creates a GitHub issue summarizing new issues, merged PRs, and open blockers |
| `daily-repo-status` | Daily | Generates daily repository status reports with productivity insights and project recommendations |
| `daily-doc-updater` | Daily | Reviews merged PRs and commits, then opens a PR with documentation updates |
| `update-docs` | Push to `main` | Analyzes code diffs on every push to main and creates draft PRs keeping docs in sync |
| `code-simplifier` | Daily | Analyzes recently modified code and opens PRs with clarity/maintainability improvements |
