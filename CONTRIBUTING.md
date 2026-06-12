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
- [goreman](https://github.com/mattn/goreman) — required to run `make dev` (starts the backend and frontend together); install with `go install github.com/mattn/goreman@latest`
- [ExifTool](https://exiftool.org/) (optional; used for metadata extraction across all supported formats — EPUB, MOBI, AZW3, PDF; imports succeed without it but metadata is derived from filename/path-derived metadata only)
- Docker (optional, for running the full stack locally)

### Installing Tools with mise

If you use [mise](https://mise.jdx.dev/), you can install Go, Node.js, pnpm, and golangci-lint at the versions pinned in `mise.toml` with a single command:

```bash
mise install
```

> **goreman is not managed by mise.** Install it separately with `go install github.com/mattn/goreman@latest` (requires your Go install bin directory — `$GOBIN` if set, otherwise `$(go env GOPATH)/bin` — to be on your `$PATH`).

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
  auth/              # Protocol-specific middleware (OPDS, KOSync, Kobo); re-exports JWT, rate limiting, and crypto from goauth
  authstore/         # Adapters bridging db.DB to goauth store interfaces (UserStore, APIKeyStore, PasskeyStore)
  db/                # Database abstraction (SQLite/Postgres), CRUD operations
  goodreads/         # Goodreads catalog client: search by query/ISBN, lookup by ASIN or Goodreads ID; used by CLI commands
  handlers/          # HTTP request handlers
  jobs/              # Background job definitions
  metadata/          # EPUB/MOBI/AZW3/PDF metadata extraction via ExifTool
  organize/          # File reorganization into canonical Author/Title/ directory structure
  pathparser/        # Path-based metadata extraction from directory layout (author, title, series)
  coverutil/         # Cover image decoding from base64 data: URLs; enforces 20 MB size limit
  sidecar/           # Writes OPF metadata and cover image sidecar files alongside book files
  server/            # HTTP server setup, routing, embedded frontend
  worker/            # asynq-based background job processing
  otel/              # Logging and tracing setup
  otelkeys/          # Shared log/telemetry field-name constants
  telemetry/         # Anonymous usage telemetry (opt-in)
  testutils/         # Test helpers: MakeTestEPUB, MakeTestEPUBWithOptions, MakeTestMOBI, MakeTestAZW3, MakeTestPDF (used in _test.go files only)
frontend/            # Svelte 5 SPA (TypeScript + Tailwind CSS)
e2e/                 # Playwright end-to-end tests
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

#### End-to-end tests (`e2e/`)

The `e2e/` directory contains [Playwright](https://playwright.dev/) browser tests that exercise the full application stack. They require a compiled binary — build it first, then run the tests from the `e2e/` directory:

```bash
# 1. Build the Go binary (frontend is embedded at build time)
go build -o biblioteka ./cmd/server

# 2. Install e2e dependencies and Playwright browsers (first time only)
cd e2e && pnpm install && pnpm exec playwright install chromium

# 3. Run e2e tests
pnpm test
```

Playwright automatically starts the server on port `3847` for the duration of the test run using the binary from step 1. Screenshots are saved on failure; traces are captured on the first retry.

To re-run only failed tests or a specific file:

```bash
# Run a specific test file
pnpm exec playwright test tests/auth.spec.ts
pnpm exec playwright test tests/settings.spec.ts
pnpm exec playwright test tests/libraries.spec.ts

# Run in headed mode to watch the browser
pnpm exec playwright test --headed
```

> **Note:** When `CI=true` (set automatically in GitHub Actions), Playwright always starts a fresh server. Locally, it reuses a running server on port `3847` if one is already available.

#### E2E spec files

| Spec file | `test.describe` group | Tests |
|-----------|----------------------|-------|
| `tests/auth.spec.ts` | `Authentication flow` | Full sign-up → dashboard → sign-out → sign-in round trip; validation error and wrong-credential error paths |
| `tests/auth.spec.ts` | `ARIA tabs accessibility` | Login/Sign Up tab roles, `aria-selected`, and `tabpanel` relationship; arrow-key navigation between tabs |
| `tests/settings.spec.ts` | `Account settings` | Change-password flow: client-side validation (empty, too short, mismatch), successful update, and verification that the old password is rejected and the new one accepted |
| `tests/libraries.spec.ts` | `Library management` | Create a library and verify it appears in the sidebar and as the page heading; validation errors when name or folder path is missing |

#### E2E test helpers (`e2e/tests/helpers/auth.ts`)

Shared auth utilities live in `e2e/tests/helpers/auth.ts`. Import them in any spec that needs authentication instead of duplicating login/signup logic. Key exports: `createTestUser(overrides?)`, `configureTimeouts(page)` (call in every test), `openAuthPage`, `openSignupForm`, `openLoginForm`, `signUp`, `signOut`, `getAuthErrorBanner`. Note: `signIn(page, email, password)` fills and submits the login form but **does not assert success** — callers must check the expected outcome.

**Usage example:**

```typescript
import { test, expect } from "@playwright/test";
import { configureTimeouts, createTestUser, signUp, signOut, signIn } from "./helpers/auth";

test("login with wrong password shows error", async ({ page }) => {
  configureTimeouts(page);
  const user = createTestUser();
  await signUp(page, user);
  await signOut(page);
  await signIn(page, user.email, "wrong-password");
  await expect(getAuthErrorBanner(page)).toContainText(/invalid email or password/i);
});
```

#### E2E test helpers (`e2e/tests/helpers/admin.ts`)

Admin auth utilities live in `e2e/tests/helpers/admin.ts`. Provides `signInAsAdmin(page)` to log in as the pre-seeded admin, plus `ADMIN_EMAIL` and `ADMIN_PASSWORD` re-exported from `e2e/constants.ts`.

**Usage example:**

```typescript
import { test, expect } from "@playwright/test";
import { configureTimeouts } from "./helpers/auth";
import { signInAsAdmin } from "./helpers/admin";

test.beforeEach(async ({ page }) => {
  configureTimeouts(page);
  await signInAsAdmin(page);
});
```

#### E2E global setup (`e2e/global-setup.ts`) and constants (`e2e/constants.ts`)

`playwright.config.ts` declares `global-setup.ts` as the Playwright `globalSetup` hook. It runs once before the entire test suite and provisions the pre-seeded admin account:

1. It calls `POST /api/auth/signup` with the credentials from `e2e/constants.ts` (`ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_NAME`).
2. Because Biblioteka automatically promotes the **first registered account** to admin, the setup verifies the response confirms `is_admin: true`. If it does not, global setup throws — this guards against accidentally running E2E tests against a database that already has accounts.
3. If the account already exists (HTTP 409), global setup silently succeeds. This allows reuse of a running dev server without re-seeding.

`e2e/constants.ts` centralises shared test configuration:

| Constant | Value | Description |
|----------|-------|-------------|
| `TEST_PORT` | `3847` | Port the test server listens on |
| `BASE_URL` | `http://localhost:3847` | Base URL used by Playwright |
| `ADMIN_EMAIL` | `e2e-admin@biblioteka-e2e.test` | Admin account email |
| `ADMIN_PASSWORD` | `adminpassword123` | Admin account password |
| `ADMIN_NAME` | `E2E Admin` | Admin account display name |

> **Important:** The test server started by Playwright uses a fixed `JWT_SECRET` of `e2e-test-jwt-secret` (see `playwright.config.ts`). Never use this secret value in a real deployment.

> **Clean database requirement (CI):** In CI (`CI=true`), Playwright always spins up a fresh server with an empty database. Running against a non-empty database will fail global setup because the first-account-is-admin guarantee no longer applies. Locally, if you reuse a running dev server (port `3847`) the 409 short-circuit in global setup handles this gracefully.

#### Test helpers (`internal/testutils`)

The `internal/testutils` package provides helpers for generating fixture book files in tests. Never commit binary book files to the repository — use these helpers to generate them at test time instead.

| Helper | Description |
|--------|-------------|
| `testutils.MakeTestEPUB(t, path, title, creator, identifier)` | Creates a minimal valid EPUB 3 fixture at the given path. For an EPUB 2 fixture, use `testutils.MakeTestEPUBWithOptions(...)` with `EPUBOptions{Version:"2.0"}` |
| `testutils.MakeTestEPUBWithOptions(t, path, title, creator, identifier, opts)` | Creates a minimal valid EPUB with full metadata control via `EPUBOptions` (description, publisher, publication date, language, cover image data, cover image href, cover media type, EPUB3 cover flag, EPUB version, subjects) |
| `testutils.MakeTestMOBI(t, path, title, author, opts)` | Creates a minimal valid MOBI file with optional metadata via `MOBIOptions` (ISBN, ASIN, publisher, language, cover image) |
| `testutils.MakeTestAZW3(t, path, title, author, opts)` | Creates a minimal valid AZW3 file (same PalmDB/MOBI binary format as MOBI; only the extension differs) |
| `testutils.MakeTestPDF(t, path, title, author, et)` | Creates a minimal valid PDF and writes metadata via ExifTool (skips the test if ExifTool is unavailable) |
| `testutils.TinyPNG()` | Returns the bytes of a minimal 1×1 pixel PNG image — useful as `CoverImageData` in `EPUBOptions` |
| `testutils.TinyJPEG()` | Returns the bytes of a minimal 1×1 pixel JPEG image — useful as `CoverImageData` in `MOBIOptions` |

Import path: `github.com/amalgamated-tools/biblioteka/internal/testutils`

```go
func TestMyExtractor(t *testing.T) {
    dir := t.TempDir()

    // EPUB
    epubPath := filepath.Join(dir, "hamlet.epub")
    testutils.MakeTestEPUB(t, epubPath, "Hamlet", "William Shakespeare", "urn:isbn:9780141396507")

    // MOBI with optional metadata
    mobiPath := filepath.Join(dir, "The Prince.mobi")
    testutils.MakeTestMOBI(t, mobiPath, "The Prince", "Niccolò Machiavelli", testutils.MOBIOptions{
        Publisher: "Public Domain",
        Language:  "en",
    })
    // ...
}
```

> **Sample books for CLI launch configs**: The "Run CLI" launch configs for individual file formats expect book files in a local `books/` directory (e.g. `books/theprince.azw3`). These files are **not** committed to the repository. Create the `books/` directory and add your own copies of the relevant files before using those launch configs. The Goodreads and **Run Server** launch configs require no local book files.

### Building

```bash
# Build frontend then compile Go binary
make build

# Run the compiled binary
make run

# Clean build artefacts
make clean
```

### Capturing Application Screenshots

The `screenshots/` directory contains the images embedded in `README.md`. Refresh them whenever you make visible UI changes:

```bash
make screenshots
```

This command:

1. Builds the frontend (`pnpm run build`) and installs root-level `node_modules`.
2. Starts a production-mode server pair using `Procfile.screen` (Vite serve on port 5173 + Go backend on port 8080) and waits until both are ready.
3. Runs `script/take-screenshots.mjs`, which drives a headless Chromium instance (via Playwright) to capture each view in **light/dark × desktop/mobile** variants.
4. Saves the resulting images to `screenshots/`.
5. Stops the background server processes.

> **Note:** `make screenshots` requires a running Redis instance on `localhost:6379`. Run `make redis-check` first or start Redis with `docker compose up -d redis`.

The screenshot script accepts the environment variables `BASE_URL`, `DEMO_NAME`, `DEMO_EMAIL`, `DEMO_PASSWORD`, `NONADMIN_NAME`, `NONADMIN_EMAIL`, `NONADMIN_PASSWORD`, `SCREENSHOT_TIMEOUT_MS`, and `SCREENSHOT_NAVIGATION_TIMEOUT_MS`. See `script/screenshots/shared.mjs` for defaults.

### IDE and Editor Support

#### VS Code

The repository includes a `.vscode/launch.json` with **Run and Debug** configurations (`Ctrl+Shift+D` / `⇧⌘D`) for the server (`cmd/server/main.go`) and CLI tool (`cmd/cli/main.go`). The CLI configurations cover scanning a directory, processing individual files by format (AZW3, MOBI, EPUB, EPUB3), and Goodreads search/fetch commands. All CLI configurations load environment variables from `.env`; copy `.env.sample` first:

```bash
cp .env.sample .env
# Edit .env with your local values (DATABASE_URL, REDIS_URL, JWT_SECRET, …)
```

> **Sample books**: The `books/` directory is not version-controlled. To use the file-specific "Run CLI" launch configs (AZW3, MOBI, EPUB, EPUB3), create the directory and add your own book files with the expected names:
>
> ```bash
> mkdir -p books
> # Add your own books/theprince.azw3, books/theprince.mobi,
> # books/alice.epub, and books/epub30-spec.epub
> ```
>
> The **Run CLI (Folder)** config works with any EPUB, MOBI, or AZW3 files you place in `books/`.

The repository also includes a `.vscode/settings.json` that configures golangci-lint (with `--fast` on save), lints on workspace save, and auto-formats Go files on save. Run `make lint` for the complete linter output before opening a pull request.

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

The OpenAPI spec (`docs/swagger/swagger.json`, `docs/swagger/swagger.yaml`, `docs/swagger/docs.go`) is generated from [swag](https://github.com/swaggo/swag) annotations on the handler functions. Regenerate it whenever you add, remove, or change an API endpoint:

```bash
# Regenerate docs/swagger/swagger.json, docs/swagger/swagger.yaml, and docs/swagger/docs.go
make swagger

# Reformat swag annotations in handler files (run after editing annotations and before committing)
make swagger-fmt
```

Always commit the updated spec files alongside the handler changes that prompted them. At runtime, the interactive Swagger UI at `/swagger/` is served via the `http-swagger` UI assets, while the raw spec at `/swagger/doc.json` is generated from `docs/swagger/docs.go`; the `docs/swagger/swagger.json` and `docs/swagger/swagger.yaml` files are primarily for committing to the repository and for client/tooling consumption, and are not served directly by the backend.

## Code Conventions

- **No new dependencies** without a discussion issue first. The project values minimal dependencies.
- **Standard library routing**: Routes are registered on `http.ServeMux` via `setupRoutes` in `internal/server/routes.go`. No router framework.
- **Handler structure**: Each domain has a handler struct (e.g., `BookHandler`) holding a `*db.DB` and other dependencies. Handlers live in `internal/handlers/`.
- **JSON responses**: Use `writeJSON(r.Context(), w, status, data)` and `writeError(r.Context(), w, status, message)` from `internal/handlers/response.go`.
- **JSON request decoding**: Use `decodeJSON(r, w, &req)` from `internal/handlers/response.go` to decode the request body. It caps the body at 1 MiB, writes a `400 Bad Request` error on failure, and returns `false` so callers can simply `return`:
  ```go
  var req createBookRequest
  if !decodeJSON(r, w, &req) {
      return
  }
  ```
- **Path parameters**: Two helpers in `internal/handlers/request.go` extract URL segments — there is no router with named params:
  - `extractPathID(path, prefix)` — extracts a single resource ID. Example: `id, ok := extractPathID(r.URL.Path, "/api/books/")`.
  - `extractPathSegments(path, prefix)` — extracts a resource ID **and** an optional sub-resource. Example: `id, sub, ok := extractPathSegments(r.URL.Path, "/api/books/")` where `sub` holds the trailing segment (e.g., `"authors"`, `"files"`).
- **Database error handling**: Use `handleDBErr(ctx, w, err, resource)` from `internal/handlers/dberrors.go` after a DB lookup. It returns `true` and writes the appropriate HTTP error when the error is non-nil (404 for `sql.ErrNoRows`, 500 otherwise), so callers can simply `return`:
  ```go
  book, err := h.DB.GetBook(r.Context(), id)
  if handleDBErr(r.Context(), w, err, "book") {
      return
  }
  ```
- **Operation error handling**: For post-operation errors (delete, add, remove sub-resources) where you want `sql.ErrNoRows` → 404 and all other errors → 500, use `handleOpErr` from `internal/handlers/dberrors.go`. The `op` string is used as the client-facing 500 error message, so keep it generic, non-sensitive, and user-appropriate (for example, `"failed to delete group"` rather than including internal error details). It accepts optional `slog.Attr` values for extra log context and returns `true` when it wrote a response:
  ```go
  if handleOpErr(r.Context(), w, h.DB.DeleteGroup(r.Context(), id, userID), "group", "failed to delete group") {
      return
  }
  // With extra log attributes:
  if handleOpErr(r.Context(), w, err, "reading list", "failed to add book to reading list",
      slog.String(otelkeys.BookID, bookID)) {
      return
  }
  ```
- **Pagination**: For list endpoints that support paging, use `parseLimitOffset(r, defaultPageLimit, maxPageLimit)` from `internal/handlers/pagination.go`. It reads `limit` and `offset` query parameters, silently falls back to safe defaults for invalid or missing values, and caps `limit` at `maxPageLimit`:
  ```go
  limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)
  ```
  The package-level constants `defaultPageLimit = 50` and `maxPageLimit = 200` are the standard values for most list endpoints.
- **Admin-only endpoints**: Use the package-level `requireAdmin(h.DB, w, r) bool` function from `internal/handlers/crud.go` to protect admin endpoints. Return early if it returns `false` — the function already writes the error response:
  ```go
  if !requireAdmin(h.DB, w, r) {
      return
  }
  ```
- **Name validation**: Before writing a named resource (author, series) to the database, call `validateName(r.Context(), w, req.Name)`. It returns `true` when the name is non-blank; on failure it writes a `400 Bad Request` response and returns `false`:
  ```go
  if !validateName(r.Context(), w, req.Name) {
      return
  }
  ```
- **Name error handling**: For named-resource create handlers, use `handleNameErr(r.Context(), w, err, db.ErrInvalidXxxName, db.ErrXxxNameExists, "an xxx")` after a failed DB write. Returns `true` when it wrote a response (caller should `return`); returns `false` when `err` doesn't match either sentinel:
  ```go
  if handleNameErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author") {
      return
  }
  ```
  `ErrInvalidXxxName` → `400 Bad Request`; `ErrXxxNameExists` → `409 Conflict`.
- **Update error handling**: For named-resource update handlers, use `handleUpdateErr(r.Context(), w, err, db.ErrInvalidXxxName, db.ErrXxxNameExists, "an xxx", "xxx", id)` to cover the full error block (not-found → 404, name errors, fallback → 500) in one call:
  ```go
  if handleUpdateErr(r.Context(), w, err, db.ErrInvalidAuthorName, db.ErrAuthorNameExists, "an author", "author", id) {
      return
  }
  ```
- **List handlers**: For simple unpaginated list handlers, use the generic `listEntities[T, DTO](w, r, resource, listFn, toDTO)` helper instead of hand-rolling the fetch-convert-respond pattern:
  ```go
  func (h *AuthorHandler) listAuthors(w http.ResponseWriter, r *http.Request) {
      listEntities(w, r, "authors", h.DB.ListAuthors, toAuthorDTO)
  }
  ```
- **Deleting a resource**: For DELETE handlers on global resources, use `deleteResource` from `internal/handlers/crud.go` instead of hand-rolling the fetch-delete-audit pattern. It fetches the entity, deletes it, writes an audit log entry, and responds with `204 No Content`. Always `return` immediately after the call:
  ```go
  deleteResource(h.DB, w, r, id, "author", "author", otelkeys.AuthorID,
      h.DB.GetAuthor, h.DB.DeleteAuthor,
      db.AuditActionAuthorDeleted,
      func(a *db.Author) map[string]any { return map[string]any{"name": a.Name} },
  )
  ```
- **Deleting a user-owned resource**: For DELETE handlers on user-scoped resources (API keys, Kobo tokens), use `deleteUserOwnedResource` instead of `deleteResource`. The get/delete functions also accept a `userID`, and there is a separate `auditEntityType` parameter:
  ```go
  deleteUserOwnedResource(h.DB, w, r, id, "API key", "api_key", otelkeys.APIKeyID,
      h.DB.GetAPIKey, h.DB.DeleteAPIKey,
      db.AuditActionAPIKeyDeleted,
      func(k *db.APIKey) map[string]any { return map[string]any{"name": k.Name} },
  )
  ```
- **Audit logging**: For create/update actions not covered by `deleteResource`, call `logAudit` after the DB write succeeds. A failed audit write is logged as a warning and never fails the request:
  ```go
  logAudit(r.Context(), h.DB, userID, db.AuditActionBookCreated, "book", b.ID, map[string]any{"title": b.Title})
  ```
- **Logging**: Use `log/slog` for structured logging. Always call the **context-aware variants** — `slog.InfoContext(ctx, ...)`, `slog.ErrorContext(ctx, ...)`, `slog.WarnContext(ctx, ...)`, `slog.DebugContext(ctx, ...)` — passing `r.Context()` in HTTP handlers or a propagated `context.Context` elsewhere. The non-context versions (`slog.Info`, `slog.Error`, etc.) are **forbidden** by the `sloglint` linter. `log.Print*`, `log.Fatal*`, and `log.Panic*` are also forbidden. For log field keys, use the predefined constants from `internal/otelkeys/logger_keys.go` (e.g. `otelkeys.UserID`, `otelkeys.BookID`) — never raw string literals. If you need a new field key, add a constant there first.
- **User scoping**: All data queries must include `user_id` to enforce per-user data isolation.
- **Formatting**: Run `go fmt ./...` before committing Go code.

## Testing Conventions

### Go tests

- **Use `testify/require`** for assertions — `require.NoError(t, err)`, `require.Equal(t, expected, actual)`, `require.True(t, cond)`, etc. Do **not** use `t.Fatal`, `t.Fatalf`, or `t.FailNow` directly. Using `testify/require` keeps assertion style consistent and produces better failure messages:
  ```go
  import "github.com/stretchr/testify/require"

  // ✅ correct
  require.NoError(t, err)
  require.Equal(t, "expected", result)

  // ❌ incorrect — use require instead
  if err != nil {
      t.Fatal(err)
  }
  ```
- **Real database**: Go tests should run against a real SQLite database. When creating SQLite databases in tests, configure them with WAL mode, `synchronous=NORMAL`, and `foreign_keys=ON`. See `internal/db/testhelper_test.go` for a canonical example/reference helper pattern.
- **Every new feature needs tests**: Any new handler, function, or component must include tests. Treat missing tests as a failing CI check — do not consider a task done until tests are written and passing.
- **Test file organization**: Large handler test files are split by sub-concern. For example, `books_authors_test.go` covers only the books↔authors relationship handlers, while `books_test.go` covers book CRUD. Follow this pattern when adding tests for new endpoints or sub-resources.

### Frontend tests

- Frontend unit tests use [Vitest](https://vitest.dev/) and live alongside their source file (e.g. `copyTimeout.test.ts` next to `copyTimeout.svelte.ts`).
- Use `vi.useFakeTimers()` for tests involving `setTimeout`/`setInterval` so tests run synchronously without real delays.
- End-to-end tests live in `e2e/` and use Playwright (see [End-to-end tests](#end-to-end-tests-e2e) above).

## AI Coding Assistant Instructions

The project provides per-agent instruction files so that AI coding assistants receive the full set of project conventions without needing them to be pasted into every prompt:

| File | Agent |
|---|---|
| `CLAUDE.md` | Claude (Anthropic) — canonical source |
| `AGENTS.md` | Codex / OpenAI-based agents |
| `GEMINI.md` | Gemini (Google) |
| `.github/copilot-instructions.md` | GitHub Copilot |

All four files contain identical content. **When you update any coding convention documented here or in the project, keep all four files in sync.** `CLAUDE.md` is the canonical source; copy its content to the other three files.

These files cover the same conventions as the [Code Conventions](#code-conventions) section above, plus additional detail on logging, error handling, HTTP handler patterns, database migrations, and frontend practices that is relevant to automated agents.

## Database Migrations

Migrations live in `db/migrations/sqlite/` and `db/migrations/postgres/` using [dbmate](https://github.com/amacneil/dbmate) format:

```sql
-- migrate:up
CREATE TABLE ...;

-- migrate:down
DROP TABLE ...;
```

Name files with a timestamp prefix: `YYYYMMDDHHMMSS_description.sql`. Migrations run automatically on startup — no separate command needed.

### Database helper functions

The `internal/db` package provides two unexported helpers for detecting unique-constraint violations when a raw SQL error is returned:

- **`isUniqueViolation(err error) bool`** — returns `true` when `err` is any unique-constraint violation (covers both SQLite and PostgreSQL error messages). Used by named-entity create/update paths (e.g. `CreateLibrary`).
- **`isColumnUniqueViolation(err error, tableCol, idxName string) bool`** — gates on `isUniqueViolation`, then additionally checks the error message for the specific column reference (`tableCol`) or index name (`idxName`). Used internally by the shared `upsertCredential` helper to distinguish a username-conflict (different user, same username) from other database errors. You do not need to call this directly when adding a new sync protocol — use the shared helpers in `protocol_credentials.go` instead (see below).

### Adding a new sync protocol credential type

When implementing the database layer for a new sync protocol, use the shared helpers in `internal/db/protocol_credentials.go` instead of hand-rolling SQL. Define a `protocolCredentialConfig` value, declare a type alias for `ProtocolCredential`, and delegate all CRUD to the unexported helpers. See `opds_credentials.go` and `kosync.go` for real examples of this pattern.

## Continuous Integration

The test workflow (`.github/workflows/test.yml`) behaves differently for pushes and pull requests:

- **Pushes to `main`**: only runs when the following paths are modified:

  | Path pattern | What it covers |
  |---|---|
  | `cmd/**` | Server and CLI entry points |
  | `internal/**` | All backend packages |
  | `frontend/**` | Svelte SPA and its tests |
  | `db/**` | Migrations and schema |
  | `go.mod`, `go.sum` | Go module changes |
  | `.golangci.yml` | Linter configuration |
  | `.github/workflows/test.yml` | Workflow file itself |

- **Pull requests targeting `main`**: always triggers (no path filter), but a `check-docs-only` gate job evaluates the changed files. If every changed file is a documentation file (`docs/**` excluding code files, or `*.md`), all test and lint jobs are skipped automatically and the workflow reports success. No manual trigger is needed for docs-only PRs.

### Job structure

The workflow runs seven jobs. The `check-docs-only` job runs first and gates all downstream jobs:

```
check-docs-only ──► frontend-checks ──► frontend-all (gate)
             │ └──► go-lint ──┐
             │                └──► go-all (gate)
             └──► go-test ──┘
                             └──► coverage-comment (PRs only, when tests ran)
```

| Job | Depends on | What it does |
|---|---|---|
| `check-docs-only` | — | Detects docs-only PRs by evaluating changed file paths; sets `skip-tests=true` output to short-circuit all downstream jobs on docs-only PRs |
| `frontend-checks` | `check-docs-only` | Installs pnpm deps (cached), builds frontend (`pnpm run build`), runs TypeScript check (`pnpm run check`), Prettier format check, ESLint (`pnpm run lint`), and frontend unit tests; skipped on docs-only PRs |
| `frontend-all` | `frontend-checks` | Gate job — fails the run if the frontend job failed; passes immediately for docs-only PRs |
| `go-lint` | `check-docs-only` | Runs golangci-lint and Go format check (`gofmt`); skipped on docs-only PRs |
| `go-test` | `check-docs-only` | Installs `exiftool` (apt package cached), runs `go test ./...` with coverage; skipped on docs-only PRs |
| `go-all` | `go-lint` + `go-test` | Gate job — fails the run if either Go job failed; passes immediately for docs-only PRs |
| `coverage-comment` | `go-test` + `frontend-checks` | Posts or updates a coverage summary comment on the PR; only runs on pull requests after a successful non-skipped test run |

All test and lint jobs run fully in parallel (the gate jobs wait for their dependencies). Total CI time is roughly `max(frontend-checks, go-lint, go-test)`.

The frontend job uses pnpm's built-in cache via `actions/setup-node` (`cache: 'pnpm'`, keyed on `frontend/pnpm-lock.yaml`) to avoid re-downloading the dependency tree on every run. Both Go jobs use the Go module cache via `actions/setup-go` (`cache: true`, keyed on `go.sum`). The `go-test` job additionally caches the `libimage-exiftool-perl` apt package via `actions/cache` (keyed on the test workflow file hash). On a cache hit the job skips `apt-get update` entirely and uses `--no-download` to install directly from the cache, cutting CI overhead. On a cache miss the full `apt-get update` + install runs and the downloaded package is saved for subsequent runs.

> **Note:** Documentation-only pull requests (files in `docs/**` and `*.md`) always trigger the test workflow but skip all test and lint jobs automatically. The workflow reports success immediately without running any tests.

> **Concurrency:** The workflow uses a concurrency group keyed on the workflow + ref (`github.ref`: branch ref for pushes, PR merge ref for pull requests). A new push to a PR branch automatically cancels any in-progress run for that branch. Runs on `main` are never cancelled.

### E2E workflow (`.github/workflows/e2etest.yml`)

The E2E workflow runs on pushes and pull requests targeting `main` or `develop`, but only when the following paths are modified:

| Path pattern | What it covers |
|---|---|
| `cmd/**` | Server and CLI entry points |
| `internal/**` | All backend packages |
| `frontend/**` | Svelte SPA and its tests |
| `db/**` | Migrations and schema |
| `e2e/**` | Playwright test files |
| `go.mod`, `go.sum` | Go module changes |
| `.github/workflows/e2etest.yml` | Workflow file itself |

It builds the full application and runs the Playwright test suite in a single job:

```
e2e (build frontend → compile Go binary → Playwright / Chromium)
```

| Job | Depends on | What it does |
|---|---|---|
| `e2e` | — | Installs pnpm deps (cached), builds the frontend, compiles the Go binary, installs Playwright + Chromium (cached; on cache hit only system deps are installed via `install-deps`), runs `pnpm test` with `CI=true` (Playwright starts a fresh server for the run); job timeout is **45 minutes** |

On completion, the `playwright-report/` artifact is uploaded and retained for **7 days**, giving you screenshots and traces for any failures.

> **Note:** Pull requests that only touch documentation files (e.g. `README.md`, `CONTRIBUTING.md`, `docs/`) will not trigger the E2E workflow. If you need E2E tests to run on a docs-only PR, trigger the workflow manually via **Actions → E2E Tests → Run workflow**.

> **Concurrency:** The workflow uses a concurrency group keyed on the workflow name and branch ref. A new push automatically cancels any in-progress run for that branch. Runs on `main` and `develop` are never cancelled.

### Other CI workflows

#### PR Title Check (`.github/workflows/pr-title.yml`)

Every pull request title is validated against the [Conventional Commits](#commit-messages) format by the `action-semantic-pull-request` action. The check runs when a PR is opened, edited, or receives a new push. If the title does not match, the check fails and the PR cannot be merged until the title is corrected.

#### Docker Build (`.github/workflows/docker-build.yml`)

On every push to `main` and on every published release, multi-arch container images (`linux/amd64`, `linux/arm64`) are built and pushed to the GitHub Container Registry (GHCR) at `ghcr.io/amalgamated-tools/biblioteka`. Images are tagged `edge` and `sha-<short-sha>` for `main`-branch builds; release builds also receive `latest` and semver tags. See [Container Images](docs/deployment.md#container-images) in the deployment guide for usage details.

#### Release Please (`.github/workflows/release-please.yml`)

Releases are automated by [Release Please](https://github.com/googleapis/release-please-action). On every push to `main`, Release Please analyses the commit history since the last release, computes the next version number according to [Semantic Versioning](https://semver.org/), and either creates or updates an open "Release PR" that updates `CHANGELOG.md` and `version` references. Merging that PR triggers a GitHub Release, which in turn triggers the Docker Build workflow to publish a release-tagged container image.

You do not need to manually tag releases or edit `CHANGELOG.md`. Commit messages that follow the Conventional Commits format are required for Release Please to correctly classify changes.

#### Dependabot (`.github/dependabot.yml`)

Dependabot monitors three package ecosystems on a weekly schedule:

| Ecosystem | Directory | Grouping |
|---|---|---|
| Go modules (`gomod`) | `/` | Minor and patch updates are grouped into a single PR |
| npm | `/frontend` | Minor and patch updates are grouped into a single PR |
| GitHub Actions | `/` | Minor and patch updates are grouped into a single PR |

Major version bumps are opened as individual PRs and require manual review before merging.

#### Dependabot Auto-Merge (`.github/workflows/dependabot-auto-merge.yml`)

Dependabot pull requests for **patch and minor** version updates are automatically approved and enabled for auto-merge. Major version updates require a manual review before merging. The auto-merge workflow runs only when the pull request author is `dependabot[bot]`.

#### Doc Updater Auto-Merge (`.github/workflows/doc-updater-auto-merge.yml`)

Documentation pull requests created by the `daily-doc-updater` agentic workflow are automatically approved and enabled for auto-merge once all required CI checks pass. The workflow targets PRs whose title starts with `docs(daily):` and that carry both the `documentation` and `automation` labels. It runs only when the pull request author is `github-actions[bot]`.

### Automated agentic workflows

Biblioteka uses a set of AI-powered workflows (GitHub Agentic Workflows) that run on a schedule or in response to events. They analyze the codebase, find issues, and open pull requests or GitHub issues automatically. You do not need to trigger them manually.

| Workflow | Trigger | Output |
|---|---|---|
| **Daily Accessibility Review** | Every 3 hours | GitHub issues labeled `a11y`, `automated-analysis` |
| **Code Simplifier** | Daily | Pull requests simplifying recently changed code |
| **Issue Triage** | On every new issue | Applies type/priority labels, flags duplicates, asks clarifying questions |
| **Daily File Diet** | Daily on weekdays; on demand | GitHub issues for source files that exceed healthy size thresholds |
| **CI Coach** | Daily | Workflow optimization suggestions |
| **Daily Repo Chronicle** | Weekdays at 4 PM UTC | Narrative summary of daily repository activity |
| **Weekly Repo Map** | Mondays | ASCII file-tree visualization of the repository |
| **Greptile Labeler** | On PR open, update, or new PR comment | Adds or removes the `greptile-changes` label based on Greptile bot activity |
| **Goal** | Every hour; on `/goal`; on workflow dispatch | Works open issues labeled `goal` until their evidence-based completion contract is satisfied, maintaining a canonical branch, one draft PR, and durable status comments |
| **Update Docs** | On push to `main` | Draft pull requests with documentation updates for code changes |
| **Portfolio Analyst** | Mondays at ~09:00 UTC | GitHub Discussion in "audits" category with workflow cost and reliability analysis |
| **Static Analysis Report** | Daily | GitHub Discussion in "security" category with security scan results |
| **Artifacts Summary** | Sundays at ~06:00 UTC | GitHub Discussion in "artifacts" category with Actions artifact usage summary |
| **Daily Malicious Code Scan** | Daily | Code scanning alerts for suspicious patterns in recent code changes |
| **CI Failure Doctor** | On CI workflow failure | GitHub issues labeled `cookie` with root-cause analysis and recommended fixes |
| **Duplicate Code Detector** | Daily | GitHub issues listing duplicate code patterns with refactoring suggestions |
| **Metrics Collector** | Daily | Agent performance metrics written to repo-memory for meta-orchestrator analysis |
| **Schema Consistency Checker** | Daily | GitHub Discussion in "audits" category with schema/code/docs inconsistency findings |
| **Claude Code User Docs Review** | Daily at 08:00 UTC | GitHub Discussion in "audits" category evaluating docs from a non-Copilot developer's perspective |
| **Daily Assign Issue to User** | Daily | Assigns one unassigned open issue to an active contributor |
| **Daily Code Metrics** | Daily | GitHub Discussion in "audits" category with code health metrics and 30-day trend charts |
| **Daily Copilot Token Report** | Weekdays at 11:00 UTC | GitHub Discussion in "audits" category with Copilot token consumption and cost trends |
| **Daily Doc Updater** | Daily at 06:00 UTC | Pull requests (auto-merged) correcting and expanding documentation |
| **Daily Issues Report** | Daily | GitHub Discussion in "audits" category with issue clustering, metrics, and trend charts |
| **Daily Multi-Device Docs Tester** | Daily | GitHub issues for responsive-design failures; asset uploads with test results |
| **Daily Observability Report** | Daily | GitHub Discussion in "audits" category with logging and telemetry coverage analysis |
| **Daily Performance Summary** | Daily | GitHub Discussion in "audits" category with 90-day performance metrics and trend charts |
| **Daily Safe Output Optimizer** | Daily | GitHub issues labeled `[safeoutputs]` when safe-output tool calls fail |
| **Daily Semgrep Scan** | Daily | Code scanning alerts for SQL injection and other security vulnerabilities |


#### Daily Accessibility Review

The `daily-accessibility-review` workflow (`daily-accessibility-review.md`) runs every three hours. It:

1. Builds the full application and starts the HTTP server with `-mode server`.
2. Uses Playwright to navigate the live app and check for [WCAG 2.2](https://www.w3.org/TR/WCAG22/) violations.
3. Reviews source code for additional accessibility issues.
4. Opens GitHub issues for any problems found, labeled `a11y` and `automated-analysis`.

When you are assigned an `a11y`-labeled issue, refer to the WCAG criterion cited in the issue body and the [Accessibility patterns](docs/frontend.md#accessibility-patterns) section in the frontend docs for implementation guidance. Close the issue with a commit that includes `Fixes #<issue-number>`.

#### Code Simplifier

The `code-simplifier` workflow runs daily and creates pull requests that simplify recently changed Go and TypeScript code. Review these PRs the same way you would a human-authored PR; merge if the simplification is correct and reject if it changes behavior.

#### Issue Triage

The `issue-triage` workflow fires on every newly opened issue. It applies conventional-commit type labels (e.g. `bug`, `feat`), detects duplicates, and posts clarifying questions when the issue description is unclear. Labels applied by triage are informational — override them if they are wrong.

#### Daily File Diet

The `daily-file-diet` workflow runs on weekdays and can also be triggered on demand. It:

1. Scans all non-test production source files (Go, TypeScript, JavaScript, and other languages) and identifies the single largest file by line count.
2. Applies a **500-line threshold**: if the largest file is under 500 lines the workflow calls `noop` and exits without creating an issue.
3. When a file exceeds 500 lines, it analyses the file's structure and opens a GitHub issue labeled `refactoring`, `code-health`, and `automated-analysis` with the title prefix `chore(file-diet):`. The issue identifies the oversized file and includes specific guidance for splitting it into smaller, more focused units.
4. The workflow is configured to skip if there is already an open issue whose title contains `file-diet`. Automatically created issues use the `chore(file-diet):` prefix, which matches this guard, so at most one file-diet issue will be open at a time.

Review these issues the same way you would a human-authored refactoring suggestion. Close an issue once the file has been split or you have determined the size is intentional and acceptable.

#### CI Coach

The `ci-coach` workflow runs daily and can also be triggered on demand. It analyses GitHub Actions workflow performance across the repository to find efficiency improvements and cost reduction opportunities. When it identifies actionable optimizations, it opens a pull request with the title prefix `ci(ci-coach):`. If a pull request cannot be created (e.g. due to branch protection), it falls back to opening a GitHub issue instead.

#### Update Docs

The `update-docs` workflow runs on every push to `main`. It examines the diff, identifies new or changed APIs, functions, configuration, and other user-visible behaviour, and opens a draft pull request with the corresponding documentation updates. If documentation is already up to date, it does nothing. Merge or close these PRs as you would any human-authored documentation PR.

#### Daily Malicious Code Scan

The `daily-malicious-code-scan` workflow runs daily. It reviews code changes from the previous three days for suspicious patterns that could indicate a malicious agentic threat — for example, credential exfiltration, supply-chain tampering, or obfuscated command execution. Confirmed findings are reported as code scanning alerts visible under **Security → Code scanning**. If you receive an alert from this workflow, treat it as a high-priority security event and investigate promptly.

#### CI Failure Doctor

The `ci-doctor` workflow fires automatically whenever the **CI** workflow completes with a `failure` or `cancelled` conclusion. It:

1. Downloads logs and artifacts from the failed run and pre-locates error lines using grep heuristics.
2. Analyses the root cause by inspecting job logs, test output, and historical failure patterns stored in the Actions cache.
3. Opens a GitHub issue labeled `cookie` with a title prefix of `[CI Failure Doctor]`. The issue includes an executive summary, root-cause analysis, reproduction steps, recommended fixes, prevention strategies, and AI self-improvement instructions.
4. Closes older duplicate `[CI Failure Doctor]` issues from the same failure pattern to keep the issue tracker tidy.

If the CI run **succeeds**, the workflow calls `noop` and exits without creating an issue.

When you see a `[CI Failure Doctor]` issue, follow the recommended actions listed in the issue body before marking it as resolved.

#### Duplicate Code Detector

The `duplicate-code-detector` workflow runs daily and scans the codebase for duplicate code patterns. It:

1. Analyses Go and CommonJS JavaScript (`.cjs`) source files for structural and logical duplication.
2. Evaluates severity: high-impact duplicates that should be refactored before they spread versus low-risk patterns that may be acceptable.
3. Opens GitHub issues for the highest-priority findings (including affected file paths, a description of the pattern, and a suggested refactoring approach), subject to a safety cap of at most 3 new issues per run.

Review these issues the same way you would a human-authored refactoring suggestion. Close an issue once you have refactored the duplicate or determined it is intentional. Keep in mind that the detector may find more duplicates than it files issues for in a single run due to this per-run cap and internal thresholding.

#### Daily Doc Updater

The `daily-doc-updater` workflow runs every day at 06:00 UTC. It reviews recent code changes and the existing documentation for gaps, inaccuracies, and outdated content. When it identifies documentation that needs updating, it opens a pull request with the title prefix `docs(daily):` and the labels `documentation` and `automation`. These PRs are automatically approved and auto-merged once CI passes (via the `doc-updater-auto-merge` workflow). PRs expire after one day if not merged.

This is a complement to the event-driven `update-docs` workflow (which fires on every push to `main`): `update-docs` documents specific code changes immediately, while `daily-doc-updater` sweeps for broader documentation drift on a schedule.

#### Daily Multi-Device Docs Tester

The `daily-multi-device-docs-tester` workflow runs every day. It opens the documentation site using Playwright at three device widths — mobile (375 px), tablet (768 px), and desktop (1280 px) — and checks for layout breakage, unreadable text, inaccessible interactive elements, and missing responsive behaviour. Any failures are reported as GitHub issues. Full test results are uploaded as Actions artifacts and retained for 2 days.

When assigned a docs-tester issue, inspect the attached artifact for screenshots and check the affected page at the reported viewport width.

#### Daily Safe Output Optimizer

The `daily-safe-output-optimizer` workflow runs every day. It inspects gateway logs for failed `safe-outputs` tool calls (e.g. `create_pull_request`, `create-issue`) and creates a GitHub issue labeled `[safeoutputs]`, `bug`, and `tool-improvement` when it identifies tool descriptions or configurations that cause repeated failures. Only one such issue is open at a time; the workflow skips if a `[safeoutputs]` issue is already open.

If you see a `[safeoutputs]`-labeled issue, update the relevant workflow's safe-output configuration or tool description as directed by the issue body.

#### Daily Semgrep Scan

The `daily-semgrep-scan` workflow runs every day. It runs [Semgrep](https://semgrep.dev/) against the repository to detect SQL injection vulnerabilities and other common security issues. Confirmed findings are reported as code scanning alerts visible under **Security → Code scanning**. The workflow uses the shared `shared/mcp/semgrep.md` tool configuration.

Treat any code-scanning alert from this workflow as a security concern and address it before the affected code reaches production.


### Slash-command workflows

These workflows are triggered manually by posting a slash command in a PR or issue comment. They do not run automatically.

| Slash command | Where | What it does |
|---|---|---|
| `/grumpy` | PR comment or review comment | Performs a critical code review focused on edge cases, potential bugs, and code quality. Posts up to 5 inline review comments. |
| `/nit` | PR comment or review comment | Performs a detailed nitpick review focused on style, naming, and best practices that linters miss. Posts up to 10 inline review comments and publishes a summary report as a GitHub Discussion. |
| `/q` | Issue or PR comment | Answers questions about the codebase, analyzes agentic workflow performance, and can open pull requests with workflow optimizations. Also triggered by a 🚀 reaction on a comment. |

#### Using the on-demand code reviewers

Two workflows are available for requesting AI-assisted code review at any point during a pull request:

- **`grumpy-reviewer`** (`/grumpy`) — takes a harsh, critical stance to surface edge cases and subtle bugs that are easy to overlook. Use it when you want a second opinion on correctness and robustness.
- **`pr-nitpick-reviewer`** (`/nit`) — focuses on style, readability, and minor best-practice improvements. Use it when you want feedback on code polish after the logic is solid.

Both are compiled and ready to use. To invoke either, post a comment on the PR containing only the slash command (e.g. `/grumpy` or `/nit`). The workflow will respond in-thread and post inline review comments on the changed files.

### On-demand workflows

These workflows are triggered manually from the **Actions** tab in GitHub. They do not run automatically.

#### Commit Changes Analyzer

The `commit-changes-analyzer` workflow generates a detailed developer-focused report of every change made to the repository since a specified commit. To run it:

1. Go to **Actions → Commit Changes Analyzer → Run workflow**.
2. Paste a full GitHub commit URL into the `commit_url` field (for example, `https://github.com/amalgamated-tools/biblioteka/commit/abc1234`).
3. Click **Run workflow**.

The workflow validates that the commit exists and is an ancestor of `HEAD`, then collects:

- Files added, modified, deleted, and renamed since that commit.
- Per-author contribution counts and the overall commit timeline.
- Functional areas touched (inferred from directory structure).
- Associated pull requests and issues.
- Line-level stats (insertions/deletions per file).

Results are published as a new Discussion in the **dev** category. The workflow does not close previous discussions, so each run produces an independent report.

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
