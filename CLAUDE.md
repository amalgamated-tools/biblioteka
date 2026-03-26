# Agent Instructions

Biblioteka is a personal digital library management system for cataloging and organizing books. It is a full-stack web application with a Go backend and a Svelte 5 frontend.

NOTE: Use American English spelling in all code, comments, and documentation (e.g., "catalog" not "catalogue").

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
cmd/cli/           # CLI tool for standalone metadata extraction
internal/
  auth/            # JWT creation/validation, rate limiting, auth middleware
  coverutil/       # Cover image decoding (base64 data: URLs; enforces 20 MB limit)
  db/              # Database layer: setup, CRUD per domain (books, authors, …)
  goodreads/       # Goodreads catalog client: search by query/ISBN, lookup by ASIN or Goodreads ID; used by CLI commands
  handlers/        # HTTP handlers, one struct per domain
  handlers/middleware/  # Logging, request ID middleware
  jobs/            # Background job definitions
  metadata/        # EPUB/MOBI/AZW3/PDF metadata extraction via ExifTool
  organize/        # File reorganization into canonical library layouts
  otel/            # OpenTelemetry logging and tracing bootstrap
  otelkeys/        # Predefined slog field-key constants (logger_keys.go)
  pathparser/      # Book path parsing from directory structure
  server/          # HTTP server init, route registration, embedded frontend dist
  sidecar/         # Sidecar file writing: OPF metadata and cover image alongside book files
  telemetry/       # Anonymous usage telemetry (opt-in)
  testutils/       # Test helpers (MakeTestEPUB, MakeTestPDF); used in _test.go files only
  worker/          # asynq worker setup and job handler registration
frontend/src/
  components/      # Svelte page components (PascalCase .svelte files)
  stores/          # Svelte reactive stores (lowercase .ts files)
  lib/api.ts       # Centralised API client
  types.ts         # Shared TypeScript types
db/migrations/
  sqlite/          # SQLite migrations (dbmate format)
  postgres/        # PostgreSQL migrations (dbmate format)
```

## OTEL Keys
- Predefine all structured log field keys as constants in `internal/otelkeys/logger_keys.go` (e.g., `UserID`, `BookID`, `RequestID`).
- Use these constants in all logging calls to ensure consistency and enable better log querying.
- If you need a new log field, add a constant in `internal/otelkeys/logger_keys.go` first before using it in your code.
- Keep the keys alphabetized

## After completing a task

- Run `make fmt` and `make hardfmt` before committing.
- Run `pnpm run lint` and `pnpm run check` in `frontend/` before committing frontend code.

## Commits and pushing

- Do not add a `Co-Authored-By:` trailer to commit messages.
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
- Prefer `errors.Is` for error handling in most cases; only check error strings as a last resort when the error type does not provide enough context.

### HTTP handlers

- Each domain has a handler struct (e.g., `BookHandler`) that holds `*db.DB` and other dependencies.
- Register routes in `internal/server/routes.go` (via `(*Server).setupRoutes`) on the standard `http.ServeMux` — do not introduce a router framework.
- Use `writeJSON(r.Context(), w, status, data)` and `writeError(r.Context(), w, status, message)` from `internal/handlers/helpers.go` for all responses.
- Extract resource IDs with `extractPathID(r.URL.Path, "/api/books/")` — there are no named URL parameters. To extract a resource ID **and** an optional sub-resource segment, use `extractPathSegments(r.URL.Path, "/api/books/")` which returns `(id, sub, ok)`.
- After fetching a resource by ID, use `handleDBErr(r.Context(), w, err, "book")` to write the error response and return early. It returns `true` when it wrote a response (caller should `return`), `false` when `err == nil`. Maps `sql.ErrNoRows` → `404 Not Found`; all other errors → `500 Internal Server Error`.
- For paginated list endpoints, use `parseLimitOffset(r, defaultPageLimit, maxPageLimit)` from `internal/handlers/pagination.go` to parse `limit` and `offset` query parameters. It silently clamps out-of-range values to safe defaults (`defaultPageLimit = 50`, `maxPageLimit = 200`).
- Before writing a named resource to the database, call `validateName(r.Context(), w, req.Name)` to guard against blank names. It returns `true` when the name is non-blank; on failure it writes a `400 Bad Request` response and returns `false`, so callers can simply return:

  ```go
  if !validateName(r.Context(), w, req.Name) {
      return
  }
  ```

- For named-resource create and update handlers, use `handleNameErr(r.Context(), w, err, db.ErrInvalidXxxName, db.ErrXxxNameExists, "an xxx")` after a failed write to translate sentinel errors into the correct HTTP responses. It returns `true` when it wrote a response (caller should `return`); returns `false` when `err` does not match either sentinel (caller handles the remaining error):

  ```go
  if err := h.DB.CreateAuthor(r.Context(), &author); err != nil {
      if handleNameErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author") {
          return
      }
      slog.ErrorContext(r.Context(), "failed to create author", slog.Any(otelkeys.Error, err))
      writeError(r.Context(), w, http.StatusInternalServerError, "failed to create author")
      return
  }
  ```

`ErrInvalidXxxName` maps to `400 Bad Request` ("name is required"); `ErrXxxNameExists` maps to `409 Conflict` ("an xxx with that name already exists").

- For update handlers, consolidate the full error block with `handleUpdateErr` instead of calling `handleNameErr` and writing a 404 by hand:

  ```go
  if handleUpdateErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author", "author", id) {
      return
  }
  ```

  `handleUpdateErr` returns `true` when it wrote a response (caller should `return`), `false` when `err == nil`. It covers: `sql.ErrNoRows` → `404 Not Found`; `ErrInvalidXxxName` → `400 Bad Request`; `ErrXxxNameExists` → `409 Conflict`; any other error → logs and returns `500 Internal Server Error`.

- For list endpoints that return a slice of DTOs, use the generic `listEntities` helper instead of hand-rolling the list-and-convert pattern:

  ```go
  listEntities(w, r, "authors", h.DB.ListAuthors, toAuthorDTO)
  ```

  `listEntities` is a generic function in `internal/handlers/helpers.go`. It calls the `list` function, converts each entity to a DTO via `toDTO`, and writes a `200 OK` JSON response. On error it logs and writes `500 Internal Server Error`. Always `return` immediately after the call.

### Deleting a resource

For DELETE handlers, use the generic `deleteResource` helper instead of hand-rolling the fetch-delete-audit pattern:

```go
deleteResource(h.DB, w, r, id, "author", otelkeys.AuthorID,
    h.DB.GetAuthor, h.DB.DeleteAuthor,
    db.AuditActionAuthorDeleted,
    func(a *db.Author) map[string]any { return map[string]any{"name": a.Name} },
)
```

`deleteResource` is a package-level generic function in `internal/handlers/helpers.go`. It fetches the entity (to capture audit metadata), deletes it, writes an audit log entry via `db.CreateAuditLog`, and responds with `204 No Content`. A failed audit write is logged as a warning and never blocks the response. Pass `nil` for `auditMeta` when no extra metadata is needed. Always `return` immediately after the call — `deleteResource` always writes the HTTP response itself.

### Deleting a user-owned resource

For DELETE handlers on **user-scoped** resources (API keys, Kobo tokens), use `deleteUserOwnedResource` instead of `deleteResource`. The difference is that the get/delete functions also accept a `userID` parameter, and there is a separate `auditEntityType` argument (a stable snake_case string written to the audit log, distinct from the human-readable display name):

```go
deleteUserOwnedResource(h.DB, w, r, id, "API key", "api_key", otelkeys.APIKeyID,
    h.DB.GetAPIKey, h.DB.DeleteAPIKey,
    db.AuditActionAPIKeyDeleted,
    func(k *db.APIKey) map[string]any { return map[string]any{"name": k.Name} },
)
```

`deleteUserOwnedResource` resolves `userID` from context and passes it to both `get` and `del`. It otherwise behaves identically to `deleteResource`. Always `return` immediately after the call.

### Audit logging (non-`deleteResource` actions)

For actions not covered by `deleteResource`, call `logAudit` after the database write succeeds:

```go
logAudit(r.Context(), h.DB, userID, db.AuditActionBookCreated, "book", b.ID, map[string]any{"title": b.Title})
```

`logAudit` is a package-level function in `internal/handlers/helpers.go`. It calls `db.CreateAuditLog` and logs a warning on failure without propagating the error, so a failed audit write never causes a request to fail. The caller must supply `userID`, typically obtained via `auth.UserIDFromContext(r.Context())`.

### Admin protection

```go
if !requireAdmin(h.DB, w, r) {
    return
}
```

`requireAdmin` is a package-level function in `internal/handlers/helpers.go`. It writes the error response itself; return immediately when it returns `false`.

### User data isolation

Every database query that reads or writes user-owned data **must** filter by `user_id`. Never return data across users.

### Dependencies

- Avoid adding new dependencies. Discuss in an issue first — the project values minimal dependencies.
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
- Run `pnpm run lint` (ESLint) and `pnpm run check` (svelte-check) before committing frontend changes.

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
