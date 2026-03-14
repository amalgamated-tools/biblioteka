# Copilot Instructions for Biblioteka

Biblioteka is a personal digital library management system for cataloging and organizing books. It is a full-stack web application with a Go backend and a Svelte 5 frontend.

## Tech Stack

- **Backend**: Go 1.25, standard `net/http` (no router framework), `database/sql`
- **Databases**: SQLite (default) and PostgreSQL — both are supported; use dialect-aware helpers
- **Background jobs**: `asynq` (Redis-backed) via `internal/worker`
- **Auth**: JWT (`golang-jwt/jwt/v5`) + OIDC (`coreos/go-oidc/v3`)
- **Middleware**: `justinas/alice` for middleware chaining
- **Frontend**: Svelte 5, TypeScript, Tailwind CSS 3, Vite
- **Migrations**: `dbmate` format, run automatically on startup

## Project Structure

```
cmd/server/        # Binary entry point
internal/
  auth/            # JWT creation/validation, rate limiting, auth middleware
  db/              # Database layer: setup, CRUD per domain (books, authors, …)
  handlers/        # HTTP handlers, one struct per domain
  handlers/middleware/  # Logging, request ID middleware
  jobs/            # Background job definitions
  server/          # HTTP server init, route registration, embedded frontend dist
  worker/          # asynq worker setup and job handler registration
  otel/            # OpenTelemetry logging and tracing bootstrap
frontend/src/
  components/      # Svelte page components (PascalCase .svelte files)
  stores/          # Svelte reactive stores (lowercase .ts files)
  lib/api.ts       # Centralised API client
  types.ts         # Shared TypeScript types
db/migrations/
  sqlite/          # SQLite migrations (dbmate format)
  postgres/        # PostgreSQL migrations (dbmate format)
```

## Go Conventions

### Logging
- Always use `log/slog` for structured logging.
- **Always use context-aware variants**: `slog.InfoContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, 
  `slog.WarnContext(ctx, ...)`, `slog.DebugContext(ctx, ...)`. The non-context versions 
  (`slog.Info`, `slog.Error`, etc.) are **forbidden by the `sloglint` linter**.
- `log.Print*`, `log.Fatal*`, and `log.Panic*` are **forbidden** by the linter (`golangci-lint` / `forbidigo`).
- Pass `r.Context()` in HTTP handlers or propagate `context.Context` through function signatures.

### Error handling
- Check every error explicitly with `if err != nil`.
- Wrap errors with context: `fmt.Errorf("context: %w", err)`.

### HTTP handlers
- Each domain has a handler struct (e.g., `BookHandler`) that holds `*db.DB` and other dependencies.
- Register routes in `internal/server/server.go` on the standard `http.ServeMux` — do not introduce a router framework.
- Use `writeJSON(w, status, data)` and `writeError(w, status, message)` from `internal/handlers/helpers.go` for all responses.
- Extract resource IDs from paths with `extractPathID(r.URL.Path, "/api/books/")` — there are no named URL parameters.

### Admin protection
```go
if !h.requireAdmin(w, r) {
    return
}
```
`requireAdmin` writes the error response itself; return immediately when it returns `false`.

### User data isolation
Every database query that reads or writes user-owned data **must** filter by `user_id`. Never return data across users.

### Dependencies
- Avoid adding new dependencies. Discuss in an issue first — the project values minimal dependencies.
- Never edit `*.gen.go` files by hand; regenerate them with `go generate ./...`.

### Formatting
Run `go fmt ./...` before committing Go code.

## Database Conventions

- Migrations live in `db/migrations/sqlite/` and `db/migrations/postgres/`.
- Name files with a timestamp prefix: `YYYYMMDDHHMMSS_description.sql`.
- Use dbmate format:
  ```sql
  -- migrate:up
  CREATE TABLE ...;

  -- migrate:down
  DROP TABLE ...;
  ```
- Migrations run automatically on server startup — no separate command is needed.
- SQLite connections use `PRAGMA journal_mode=WAL`, `synchronous=NORMAL`, and `foreign_keys=ON`.
- Use `db.Timestamp` for time columns and `db.now()` for dialect-aware current-time expressions.

## Frontend Conventions

- Use TypeScript strict mode; put shared types in `src/types.ts`.
- All API calls go through `src/lib/api.ts`.
- Manage reactive state in Svelte stores under `src/stores/`.
- Style with Tailwind CSS utility classes — no component library.
- Component files are PascalCase `.svelte`; store files are lowercase `.ts`.
- Run `pnpm run lint` (ESLint) and `pnpm run check` (svelte-check) before committing frontend code.

## Testing

```bash
# Go tests
go test ./...

# Frontend tests
cd frontend && pnpm run test
```

- Go tests use a real SQLite database configured with WAL, `synchronous=NORMAL`, and `foreign_keys=ON` (see `internal/db/testhelper_test.go`).

## Common Commands

```bash
make dev        # Start backend (air hot-reload) + Vite frontend dev server
make build      # Build frontend then compile Go binary
go test ./...   # Run all Go tests
go fmt ./...    # Format Go code
cd frontend && pnpm run lint && pnpm run check   # Lint & type-check frontend
```
