# Contributing to biblioteka

Thanks for your interest in contributing! This guide covers everything you need to get started.

## Before You Start: Contributor License Agreement

All contributors must sign the [Contributor License Agreement (CLA)](CLA.md) before their pull request can be merged. The CLA bot will automatically prompt you when you open a pull request. Sign by leaving a comment on the PR with the following exact text:

```
I have read the CLA Document and I hereby sign the CLA
```

Your signature is recorded once and applies to all future contributions.

## Prerequisites

- Go 1.26+
- Node.js 22+ with [pnpm](https://pnpm.io/)
- Redis (for background jobs)
- [ExifTool](https://exiftool.org/) (optional; only needed for PDF and MOBI/AZW3 metadata extraction)
- Docker (optional, for running the full stack locally)

### Installing Tools with mise

If you use [mise](https://mise.jdx.dev/), you can install Go, Node.js, pnpm, and golangci-lint at the versions pinned in `mise.toml` with a single command:

```bash
mise install
```

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
  otelkeys/          # Shared log/telemetry field-name constants
  telemetry/         # Anonymous usage telemetry (opt-in)
  testutils/         # Test helpers: MakeTestEPUB, MakeTestPDF (used in _test.go files only)
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

#### Test helpers (`internal/testutils`)

The `internal/testutils` package provides helpers for generating fixture book files in tests:

| Helper | Description |
|--------|-------------|
| `testutils.MakeTestEPUB(t, path, title, creator, identifier)` | Creates a minimal valid EPUB at the given path |
| `testutils.MakeTestEPUBWithOptions(t, path, title, creator, identifier, opts)` | Creates a minimal valid EPUB with additional OPF fields (description, publisher, publication date, language) |
| `testutils.MakeTestPDF(t, path, title, author, et)` | Creates a minimal valid PDF and writes metadata via ExifTool (skips the test if ExifTool is unavailable) |

Import path: `github.com/amalgamated-tools/biblioteka/internal/testutils`

```go
func TestMyExtractor(t *testing.T) {
    path := filepath.Join(t.TempDir(), "test.epub")
    testutils.MakeTestEPUB(t, path, "Hamlet", "William Shakespeare", "urn:isbn:9780141396507")
    // ...
}
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

### Updating the OpenAPI Specification

The OpenAPI spec (`docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go`) is generated from [swag](https://github.com/swaggo/swag) annotations on the handler functions. Regenerate it whenever you add, remove, or change an API endpoint:

```bash
# Regenerate docs/swagger.json, docs/swagger.yaml, and docs/docs.go
make swagger

# Reformat swag annotations in handler files (run after editing annotations and before committing)
make swagger-fmt
```

Always commit the updated spec files alongside the handler changes that prompted them. At runtime, the interactive Swagger UI at `/swagger/` is served via the `http-swagger` UI assets, while the raw spec at `/swagger/doc.json` is generated from `docs/docs.go`; the `docs/swagger.json` and `docs/swagger.yaml` files are primarily for committing to the repository and for client/tooling consumption, and are not served directly by the backend.

## Code Conventions

- **No new dependencies** without a discussion issue first. The project values minimal dependencies.
- **Standard library routing**: Routes are registered on `http.ServeMux` via `setupRoutes` in `internal/server/routes.go`. No router framework.
- **Handler structure**: Each domain has a handler struct (e.g., `BookHandler`) holding a `*db.DB` and other dependencies. Handlers live in `internal/handlers/`.
- **JSON responses**: Use `writeJSON(r.Context(), w, status, data)` and `writeError(r.Context(), w, status, message)` from `internal/handlers/helpers.go`.
- **JSON request decoding**: Use `decodeJSON(r, w, &req)` from `internal/handlers/helpers.go` to decode the request body. It caps the body at 1 MiB, writes a `400 Bad Request` error on failure, and returns `false` so callers can simply `return`:
  ```go
  var req createBookRequest
  if !decodeJSON(r, w, &req) {
      return
  }
  ```
- **Path parameters**: Two helpers in `internal/handlers/helpers.go` extract URL segments — there is no router with named params:
  - `extractPathID(path, prefix)` — extracts a single resource ID. Example: `id, ok := extractPathID(r.URL.Path, "/api/books/")`.
  - `extractPathSegments(path, prefix)` — extracts a resource ID **and** an optional sub-resource. Example: `id, sub, ok := extractPathSegments(r.URL.Path, "/api/books/")` where `sub` holds the trailing segment (e.g., `"authors"`, `"files"`).
- **Admin-only endpoints**: Use the handler's `requireAdmin(w, r) bool` method to protect admin endpoints. Return early if it returns `false` — the method already writes the error response:
  ```go
  if !h.requireAdmin(w, r) {
      return
  }
  ```
- **Logging**: Use `log/slog` for structured logging. Always call the **context-aware variants** — `slog.InfoContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`, `slog.DebugContext(ctx, ...)` — passing `r.Context()` in HTTP handlers or a propagated `context.Context` elsewhere. The non-context versions (`slog.Info`, `slog.Error`, etc.) are **forbidden** by the `sloglint` linter. `log.Print*`, `log.Fatal*`, and `log.Panic*` are also forbidden. For log field keys, use the predefined constants from `internal/otelkeys/logger_keys.go` (e.g. `otelkeys.UserID`, `otelkeys.BookID`) — never raw string literals. If you need a new field key, add a constant there first.
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

## Continuous Integration

The test workflow (`.github/workflows/test.yml`) runs on pushes and pull requests targeting `main`, but only when the following paths are modified:

| Path pattern | What it covers |
|---|---|
| `cmd/**` | Server and CLI entry points |
| `internal/**` | All backend packages |
| `frontend/**` | Svelte SPA and its tests |
| `db/**` | Migrations and schema |
| `go.mod`, `go.sum` | Go module changes |
| `.golangci.yml` | Linter configuration |
| `.github/workflows/test.yml` | Workflow file itself |

### Job structure

The workflow runs four jobs. `frontend-build` and `frontend-checks` start in parallel at the beginning of every run:

```
frontend-build ──┬──► go-tests
                 │
frontend-checks ─┤
                 │
                 └──► frontend-all (gate)
```

| Job | Depends on | What it does |
|---|---|---|
| `frontend-build` | — | Installs pnpm deps (cached), runs `pnpm run build`, uploads the `dist` artifact |
| `frontend-checks` | — | Installs pnpm deps (cached), runs TypeScript check, Prettier format check, and frontend unit tests |
| `go-tests` | `frontend-build` | Downloads `dist`, runs golangci-lint, `go test -v ./...`, and Go format check |
| `frontend-all` | `frontend-build` + `frontend-checks` | Gate job — fails the run if either frontend job failed |

Because `frontend-checks` and `go-tests` run in parallel, total CI time is roughly `max(frontend-checks, go-tests)` rather than their sum.

Both frontend jobs use pnpm's built-in cache via `actions/setup-node` (`cache: 'pnpm'`, keyed on `frontend/pnpm-lock.yaml`) to avoid re-downloading the dependency tree on every run.

> **Note:** Pull requests that only touch documentation files (e.g. `README.md`, `CONTRIBUTING.md`, `docs/`) will not trigger the test workflow. If you need CI to run on a docs-only PR, trigger it manually via **Actions → Test → Run workflow**.

## Commit Messages

This project follows the [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) specification for all commit messages and pull request titles.

### Format

```
<type>[optional scope][optional !]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation-only changes |
| `style` | Code style / formatting (no logic change) |
| `refactor` | Refactoring (no feature or bug fix) |
| `perf` | Performance improvement |
| `test` | Adding or fixing tests |
| `build` | Build system or dependency changes |
| `ci` | CI/CD pipeline changes |
| `chore` | Maintenance tasks, tooling, or repo upkeep |

### Examples

```
feat(books): add bulk import endpoint
fix: prevent duplicate authors on concurrent inserts
docs(api): update OPDS feed examples
refactor(handlers)!: consolidate error response helpers

BREAKING CHANGE: writeError now requires a context parameter
```

### Rules

- The type **must** be one of the types listed above.
- A scope **may** be provided in parentheses after the type (e.g., `fix(parser):`).
- A `!` before the colon indicates a breaking change.
- Breaking changes **must** also be described in a `BREAKING CHANGE:` footer or in the commit description when using `!`.
- The description **must** immediately follow the colon and space.
- Pull request titles **must** also follow this format, since PRs are squash-merged.

## Submitting a Pull Request

1. Fork the repository and create a feature branch from `main`.
2. Make your changes following the conventions above.
3. Run `go fmt ./...` and `go test ./...` to verify all Go code is formatted and tests pass.
4. Run `cd frontend && pnpm run lint && pnpm run check` to verify frontend code.
5. **Update documentation** in `docs/` if your change adds, removes, or modifies an API endpoint, database table, configuration option, background job, or any user-visible behaviour. If you add or change API handler annotations, regenerate the OpenAPI spec with `make swagger`.
6. Open a pull request against `main` with a title that follows the [Conventional Commits](#commit-messages) format and a clear description of what and why.
7. Sign the [CLA](CLA.md) if prompted by the CLA bot (first-time contributors only).
8. A maintainer will review your PR and provide feedback.

## Questions?

Open an issue if you have questions or want to discuss a feature before implementing it.
