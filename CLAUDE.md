# Agent Instructions

Biblioteka is a personal digital library management system for cataloging and organizing books. It is a full-stack web application with a Go backend and a Svelte 5 frontend.

## Tech Stack

- **Backend**: Go 1.26.1, standard `net/http` (no router framework), `database/sql`
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

## After completing a task

- Run `make fmt` and `make hardfmt` before committing.
- Run `pnpm run lint` and `pnpm run check` in `frontend/` before committing frontend code.

## Commits and pushing

- Do not include a co-author line in commit messages.
- Require user approval before committing.
- Require user approval before pushing.
- Follow [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/): `<type>[optional scope][optional !]: <description>`
- Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.
- A scope may be added in parentheses (e.g., `fix(parser):`). Use `!` before the colon for breaking changes.
- PR titles must also follow this format since PRs are squash-merged.

## Go Conventions

### Logging

- Always use `log/slog` for structured logging.
- **Always use context-aware variants**: `slog.InfoContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`, `slog.DebugContext(ctx, ...)`. The non-context versions (`slog.Info`, `slog.Error`, etc.) are **forbidden by `sloglint`**.
- `log.Print*`, `log.Fatal*`, and `log.Panic*` are **forbidden** by `forbidigo`.
- Pass `r.Context()` in HTTP handlers or propagate `context.Context` through function signatures.
- Do not use raw string keys in log fields; use the predefined constants in `internal/otelkeys/logger_keys.go` (e.g., `otelkeys.UserID`).
- If you need a new log field, add a constant in `internal/otelkeys/logger_keys.go` first.

### Error handling

- Check every error explicitly with `if err != nil`.
- Do not ignore errors in tests either.
- Wrap errors with context: `fmt.Errorf("context: %w", err)`.

### HTTP handlers

- Each domain has a handler struct (e.g., `BookHandler`) that holds `*db.DB` and other dependencies.
- Register routes in `internal/server/server.go` on the standard `http.ServeMux` — do not introduce a router framework.
- Use `writeJSON(r.Context(), w, status, data)` and `writeError(r.Context(), w, status, message)` from `internal/handlers/helpers.go` for all responses.
- Extract resource IDs with `extractPathID(r.URL.Path, "/api/books/")` — there are no named URL parameters.

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

- Avoid adding new dependencies. The project values minimal dependencies.
- Never edit `*.gen.go` files by hand; regenerate with `go generate ./...`.

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
- Migrations run automatically on server startup.
- SQLite connections use `PRAGMA journal_mode=WAL`, `synchronous=NORMAL`, and `foreign_keys=ON`.
- Use `db.Timestamp` for time columns and `db.now()` for dialect-aware current-time expressions.

## Frontend Conventions

- Use TypeScript strict mode; put shared types in `src/types.ts`.
- All API calls go through `src/lib/api.ts`.
- Manage reactive state in Svelte stores under `src/stores/`.
- Style with Tailwind CSS utility classes — no component library.
- Component files are PascalCase `.svelte`; store files are lowercase `.ts`.

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
make fmt        # Format code
make hardfmt    # Strict formatting
go test ./...   # Run all Go tests
cd frontend && pnpm run lint && pnpm run check   # Lint & type-check frontend
```
