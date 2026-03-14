# Contributing to biblioteka

Thanks for your interest in contributing! This guide covers everything you need to get started.

## Prerequisites

- Go 1.25+
- Node.js 22+ with [pnpm](https://pnpm.io/)
- Redis (for background jobs)
- Docker (optional, for running the full stack locally)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/amalgamated-tools/biblioteka.git
cd biblioteka

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# (Optional) Check Redis is available
make redis-check

# Start the development servers (backend + frontend with hot-reload)
make dev
```

The backend runs on `http://localhost:8080` and the Vite dev server proxies `/api` to it.

## Project Layout

```
cmd/
  server/            # Main entry point (server binary)
  cli/               # CLI tool for standalone metadata extraction
internal/
  auth/              # JWT, rate limiting, middleware
  db/                # Database abstraction (SQLite/Postgres), CRUD operations
  handlers/          # HTTP request handlers
  jobs/              # Background job definitions
  metadata/          # EPUB/MOBI/PDF metadata extraction
  server/            # HTTP server setup, routing, embedded frontend
  worker/            # asynq-based background job processing
  otel/              # Logging and tracing setup
  telemetry/         # Anonymous usage telemetry (opt-in)
frontend/            # Svelte 5 SPA (TypeScript + Tailwind CSS)
db/
  schema.sql         # Reference schema
  migrations/        # sqlite/ and postgres/ migration directories
script/              # Build and release helper scripts
```

## Development Workflow

### Running Tests

```bash
# All Go tests
go test ./...

# Single package
go test ./internal/handlers/

# Frontend tests
cd frontend && pnpm run test
```

### Building

```bash
# Build frontend then compile Go binary
make build

# Run the compiled binary
make run

# Clean build artefacts
make clean
```

### Frontend (from `frontend/`)

```bash
pnpm run dev      # Vite dev server
pnpm run build    # Production build → internal/server/dist/
pnpm run check    # svelte-check type checking
pnpm run lint     # ESLint
pnpm run format   # Prettier
```

See [docs/frontend.md](docs/frontend.md) for the frontend architecture overview, including the Svelte 5 `$state` class-based store pattern, hash-based routing, and guidance on adding new stores and views.

## Code Conventions

- **No new dependencies** without a discussion issue first. The project values minimal dependencies.
- **Standard library routing**: Routes are registered on `http.ServeMux` in `internal/server/server.go`. No router framework.
- **Handler structure**: Each domain has a handler struct (e.g., `BookHandler`) holding a `*db.DB` and other dependencies. Handlers live in `internal/handlers/`.
- **JSON responses**: Use `writeJSON(w, status, data)` and `writeError(w, status, message)` from `internal/handlers/helpers.go`.
- **Path parameters**: Extract resource IDs with `extractPathID(path, prefix)` from `internal/handlers/helpers.go` — there is no router with named params. Example: `id, ok := extractPathID(r.URL.Path, "/api/books/")`.
- **Admin-only endpoints**: Use the handler's `requireAdmin(w, r) bool` method to protect admin endpoints. Return early if it returns `false` — the method already writes the error response:
  ```go
  if !h.requireAdmin(w, r) {
      return
  }
  ```
- **Logging**: Use `log/slog` for structured logging. `log.Print*` is forbidden by the linter.
- **User scoping**: All data queries must include `user_id` to enforce per-user data isolation.
- **Formatting**: Run `go fmt ./...` before committing Go code.

## Database Migrations

Migrations live in `db/migrations/sqlite/` and `db/migrations/postgres/` using [dbmate](https://github.com/amacneil/dbmate) format:

```sql
-- migrate:up
CREATE TABLE ...;

-- migrate:down
DROP TABLE ...;
```

Name files with a timestamp prefix: `YYYYMMDDHHMMSS_description.sql`. Migrations run automatically on startup — no separate command needed.

## Submitting a Pull Request

1. Fork the repository and create a feature branch from `main`.
2. Make your changes following the conventions above.
3. Run `go test ./...` and `cd frontend && pnpm run lint && pnpm run check` to verify everything passes.
4. Open a pull request against `main` with a clear description of what and why.
5. A maintainer will review your PR and provide feedback.

## Questions?

Open an issue if you have questions or want to discuss a feature before implementing it.
