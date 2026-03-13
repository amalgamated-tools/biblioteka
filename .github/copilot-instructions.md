# Copilot Instructions for Fichemos

Fichemos is a self-hosted media management dashboard that aggregates *arr services (Radarr, Sonarr, Prowlarr, Seerr) into a single interface.

## Architecture

- **Backend**: Go HTTP server (`cmd/server/main.go`) using only the standard library `net/http` — no router framework.
- **Frontend**: Svelte 5 SPA (`frontend/`) with TypeScript, Tailwind CSS, and Vite. Built output is embedded into the Go binary via `//go:embed`.
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGO). Database file lives at `/data/fichemos.db` (Docker) or `db/fichemos.db` (local dev).
- **API clients**: Generated from OpenAPI specs using `oapi-codegen`. Generated files live in `pkg/{radarr,sonarr,prowlarr,seerr}/client.gen.go` — do not edit these by hand.
- **Auth**: JWT-based (Bearer tokens). `internal/auth/middleware.go` extracts user ID into request context via `auth.UserIDFromContext(ctx)`.

## Build & Run

```bash
make dev          # Run frontend + backend dev servers (goreman + air hot-reload)
make build        # Build frontend then compile Go binary
make run          # Build and run production binary
go test ./...     # Run all Go tests
go test ./internal/db/  # Run tests for a single package
```

### Frontend (from `frontend/`)

```bash
pnpm install      # Install dependencies
pnpm run dev      # Vite dev server (proxies /api to localhost:8080)
pnpm run build    # Production build → internal/server/dist/
pnpm run check    # svelte-check type checking
pnpm run lint     # ESLint
pnpm run format   # Prettier
```

### Code Generation

```bash
go generate ./...              # Regenerate all *arr API clients from OpenAPI specs
scripts/generate.sh            # Download latest OpenAPI specs + regenerate clients
```

OpenAPI config files are in `openapi/*.yaml`; specs are in `openapi/*.json`.

## Database Migrations

Migrations use dbmate format (`-- migrate:up` / `-- migrate:down`) in `db/migrations/`. Files are named with timestamps: `YYYYMMDDHHMMSS_description.sql`. Migrations run automatically on startup — no separate migration command.

## Key Conventions

- **Routing**: Routes are registered manually on `http.ServeMux` in `main.go`. API routes are prefixed with `/api/`. Handlers dispatch on HTTP method internally (e.g., `switch r.Method`).
- **Handler structure**: Each domain has a handler struct (e.g., `AuthHandler`, `ArrServiceHandler`) that holds a `*db.DB` and any other dependencies. Handlers live in `internal/handlers/`.
- **JSON responses**: Use `writeJSON(w, status, data)` and `writeError(w, status, message)` from `internal/handlers/helpers.go`.
- **Path parameters**: Extracted manually via `extractPathID(path, prefix)` — there is no router with named params.
- **User scoping**: All data queries include `user_id` to enforce per-user data isolation.
- **Logging**: Use `log/slog` for structured logging in library/db code; `log.Printf` in handlers.
- **Environment variables**: `PORT` (server port, default 8080), `JWT_SECRET` (optional, random if unset).
