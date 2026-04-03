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
- [ExifTool](https://exiftool.org/) (optional; used for metadata extraction across all supported formats — EPUB, MOBI, AZW3, PDF; imports succeed without it but metadata is derived from filename/path-derived metadata only)
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
  testutils/         # Test helpers: MakeTestEPUB, MakeTestMOBI, MakeTestAZW3, MakeTestPDF (used in _test.go files only)
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

Shared auth utilities live in `e2e/tests/helpers/auth.ts`. Import them in any spec that needs authentication instead of duplicating login/signup logic.

| Export | Description |
|--------|-------------|
| `createTestUser(overrides?)` | Returns a `TestUser` with a unique email, default display name, and password. Pass `overrides` to customise individual fields. |
| `configureTimeouts(page)` | Sets `page.setDefaultTimeout` and `page.setDefaultNavigationTimeout` from `E2E_TIMEOUT_MS` / `E2E_NAVIGATION_TIMEOUT_MS` env vars (defaults: 5 000 ms each). Call this at the top of every test or `test.beforeEach`. |
| `openAuthPage(page)` | Navigates to `/` and waits for the login button to appear. |
| `openSignupForm(page)` | Calls `openAuthPage`, then clicks "Sign Up" and waits for the registration form. |
| `openLoginForm(page)` | Calls `openAuthPage`, then clicks the login button and waits for email/password inputs. |
| `signUp(page, user)` | Completes the sign-up flow for a `TestUser` and waits for the authenticated home screen. |
| `signIn(page, email, password)` | Fills and submits the login form. **Does not assert success** — callers must check the expected outcome (some tests deliberately sign in with bad credentials). |
| `signOut(page)` | Clicks the logout button and waits for the login screen and `localStorage` token removal. |
| `getAuthErrorBanner(page)` | Returns the Playwright locator for the auth error banner (`data-testid="auth-error"`). |

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

Admin auth utilities live in `e2e/tests/helpers/admin.ts`. Use these in specs that need to act as the pre-seeded admin user instead of creating a fresh user each time.

| Export | Description |
|--------|-------------|
| `signInAsAdmin(page)` | Opens the login form, signs in as the global admin (seeded by `global-setup.ts`), and waits for the authenticated dashboard. |
| `ADMIN_EMAIL` | The email address of the seeded admin user (re-exported from `e2e/constants.ts`). |
| `ADMIN_PASSWORD` | The password of the seeded admin user (re-exported from `e2e/constants.ts`). |

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
| `testutils.MakeTestEPUB(t, path, title, creator, identifier)` | Creates a minimal valid EPUB 3 at the given path |
| `testutils.MakeTestEPUBWithOptions(t, path, title, creator, identifier, opts)` | Creates a minimal valid EPUB with metadata and format control via `EPUBOptions`, including description, publisher, publication date, language, cover image settings, EPUB version, and subjects |
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

The screenshot script accepts several environment variables for pointing at a non-default server or using different credentials:

| Variable | Default | Description |
|---|---|---|
| `BASE_URL` | `http://localhost:5173` | URL of the running Vite dev/serve frontend |
| `DEMO_NAME` | `Demo` | Display name for the admin account the script signs up |
| `DEMO_EMAIL` | `demo@veverka.net` | Email for the admin account |
| `DEMO_PASSWORD` | `password123` | Password for the admin account |
| `NONADMIN_NAME` | `Regular User` | Display name for the secondary non-admin account |
| `NONADMIN_EMAIL` | `nonadmin@veverka.net` | Email for the non-admin account |
| `NONADMIN_PASSWORD` | `password123` | Password for the non-admin account |
| `SCREENSHOT_TIMEOUT_MS` | `5000` | Default Playwright action timeout (ms) |
| `SCREENSHOT_NAVIGATION_TIMEOUT_MS` | `8000` | Playwright navigation timeout (ms) |

#### `Procfile.screen`

`Procfile.screen` defines the server pair used **only** during screenshot capture. It starts a production-mode server (the compiled Go binary is not required — it uses `go run`) alongside `pnpm run dev --host`:

```
web: PORT=8080 go run cmd/server/main.go -mode=server
frontend: cd frontend && pnpm run dev --host
```

This is distinct from `Procfile.dev`, which uses [air](https://github.com/cosmtrek/air) for hot-reload during normal development.

### IDE and Editor Support

#### VS Code

The repository includes a `.vscode/launch.json` with ready-to-use **Run and Debug** configurations (`Ctrl+Shift+D` / `⇧⌘D`):

| Configuration | Binary | What it does |
|---|---|---|
| **Run CLI (Folder)** | `cmd/cli/main.go` | Runs `scan-directory books/` — scans a local `books/` directory and imports all supported files |
| **Run CLI (AZW3)** | `cmd/cli/main.go` | Runs `process-file` against `books/theprince.azw3` |
| **Run CLI (MOBI)** | `cmd/cli/main.go` | Runs `process-file` against `books/theprince.mobi` |
| **Run CLI (EPUB)** | `cmd/cli/main.go` | Runs `process-file` against `books/alice.epub` (EPUB 2) |
| **Run CLI (EPUB3)** | `cmd/cli/main.go` | Runs `process-file` against `books/epub30-spec.epub` (EPUB 3) |
| **Run CLI (Goodreads Search)** | `cmd/cli/main.go` | Runs `goodreads-search "Project Hail Mary"` |
| **Run CLI (Goodreads Search by ISBN)** | `cmd/cli/main.go` | Runs `goodreads-search-isbn 9780593135204` |
| **Run CLI (Goodreads Fetch by ASIN)** | `cmd/cli/main.go` | Runs `goodreads-get-by-asin 0593135202` |
| **Run CLI (Goodreads Fetch by ID)** | `cmd/cli/main.go` | Runs `goodreads-get-by-id` with a Goodreads KCA ID |
| **Run CLI (Goodreads Fetch by Legacy ID)** | `cmd/cli/main.go` | Runs `goodreads-get-by-legacy-id 54493401` |
| **Run Server** | `cmd/server/main.go` | Starts the Biblioteka HTTP server |

All CLI configurations load environment variables from `.env` in the workspace root. Copy `.env.sample` to `.env` and fill in the required values before launching:

```bash
cp .env.sample .env
# Edit .env with your local values (DATABASE_URL, REDIS_URL, JWT_SECRET, …)
```

> **Sample books for CLI launch configs**: The "Run CLI" launch configs for individual file formats expect book files in a local `books/` directory (e.g. `books/theprince.azw3`). These files are **not** committed to the repository. Create the `books/` directory and add your own copies of the relevant files before using those launch configs. The **Run CLI (Goodreads)** and **Run Server** configs require no local book files.

The repository also includes a `.vscode/settings.json` with workspace-wide editor settings for the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go):

| Setting | Value | Effect |
|---------|-------|--------|
| `go.lintTool` | `golangci-lint` | Uses golangci-lint instead of the default `staticcheck` |
| `go.lintFlags` | `["--fast"]` | Runs a faster subset of linters on save; the full suite (`make lint`) runs in CI |
| `go.lintOnSave` | `workspace` | Lints all packages in the workspace on every save |
| `editor.formatOnSave` | `true` | Auto-formats Go files on save (equivalent to `go fmt`) |

This means that when you save a `.go` file, VS Code will automatically format it and run a fast lint pass. Run `make lint` (or `golangci-lint run ./...`) for the complete linter output before opening a pull request.

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
- **Database error handling**: Use `handleDBErr(ctx, w, err, resource)` from `internal/handlers/helpers.go` after a DB lookup. It returns `true` and writes the appropriate HTTP error when the error is non-nil (404 for `sql.ErrNoRows`, 500 otherwise), so callers can simply `return`:
  ```go
  book, err := h.DB.GetBook(r.Context(), id)
  if handleDBErr(r.Context(), w, err, "book") {
      return
  }
  ```
- **Pagination**: For list endpoints that support paging, use `parseLimitOffset(r, defaultPageLimit, maxPageLimit)` from `internal/handlers/pagination.go`. It reads `limit` and `offset` query parameters, silently falls back to safe defaults for invalid or missing values, and caps `limit` at `maxPageLimit`:
  ```go
  limit, offset := parseLimitOffset(r, defaultPageLimit, maxPageLimit)
  ```
  The package-level constants `defaultPageLimit = 50` and `maxPageLimit = 200` are the standard values for most list endpoints.
- **Admin-only endpoints**: Use the package-level `requireAdmin(h.DB, w, r) bool` function from `internal/handlers/helpers.go` to protect admin endpoints. Return early if it returns `false` — the function already writes the error response:
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
- **Deleting a resource**: For DELETE handlers on global resources, use `deleteResource` from `internal/handlers/helpers.go` instead of hand-rolling the fetch-delete-audit pattern. It fetches the entity, deletes it, writes an audit log entry, and responds with `204 No Content`. Always `return` immediately after the call:
  ```go
  deleteResource(h.DB, w, r, id, "author", otelkeys.AuthorID,
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

The workflow runs five jobs. All three leaf jobs start in parallel at the beginning of every run:

```
frontend-checks ──► frontend-all (gate)

go-lint ──┐
          └──► go-all (gate)
go-test ──┘
```

| Job | Depends on | What it does |
|---|---|---|
| `frontend-checks` | — | Installs pnpm deps (cached), builds frontend (`pnpm run build`), runs TypeScript check (`pnpm run check`), Prettier format check, ESLint (`pnpm run lint`), and frontend unit tests |
| `frontend-all` | `frontend-checks` | Gate job — fails the run if the frontend job failed |
| `go-lint` | — | Runs golangci-lint and Go format check (`gofmt`) |
| `go-test` | — | Installs `exiftool` (apt package cached), runs `go test -v ./...` |
| `go-all` | `go-lint` + `go-test` | Gate job — fails the run if either Go job failed |

All five jobs run fully in parallel (the two gate jobs wait for their dependencies). Total CI time is roughly `max(frontend-checks, go-lint, go-test)`.

The frontend job uses pnpm's built-in cache via `actions/setup-node` (`cache: 'pnpm'`, keyed on `frontend/pnpm-lock.yaml`) to avoid re-downloading the dependency tree on every run. Both Go jobs use the Go module cache via `actions/setup-go` (`cache: true`, keyed on `go.sum`). The `go-test` job additionally caches the `libimage-exiftool-perl` apt package via `actions/cache` (keyed on the test workflow file hash). On a cache hit the job skips `apt-get update` entirely and uses `--no-download` to install directly from the cache, cutting CI overhead. On a cache miss the full `apt-get update` + install runs and the downloaded package is saved for subsequent runs.

> **Note:** Pull requests that only touch documentation files (e.g. `README.md`, `CONTRIBUTING.md`, `docs/`) will not trigger the test workflow. If you need CI to run on a docs-only PR, trigger it manually via **Actions → Test → Run workflow**.

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
| **Daily Doc Updater** | Daily at 06:00 UTC | Draft pull requests correcting and expanding documentation |
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

#### Daily Repo Chronicle

The `daily-repo-chronicle` workflow runs on weekdays at 16:00 UTC and can also be triggered on demand. It collects the day's repository activity — commits, pull requests, issues, and discussions — and writes a newspaper-style narrative summary with exactly two trend charts. The report is published as a GitHub Discussion in the **announcements** category with a `📰` title prefix. The previous day's discussion from this workflow is closed automatically when a new one is created.

These discussions give contributors a quick narrative view of what changed each day without reading the raw commit log.

#### Weekly Repo Map

The `weekly-repo-map` workflow runs every Monday at approximately 15:00 UTC and can also be triggered on demand. It generates an ASCII file-tree visualization of the repository's structure with size distribution, then creates a GitHub issue labeled `documentation` with the title prefix `[repo-map]`. The previous repo-map issue is closed automatically when a new one is created.

Use these issues to quickly understand which directories have grown and whether the project layout remains navigable.

#### Greptile Labeler

The `greptile-labeler` workflow fires whenever a pull request is opened, updated (new commit pushed), or receives a new comment. It scans PR review threads and comments for activity from the **Greptile** bot. If Greptile has commented, the workflow adds the `greptile-changes` label to the PR. If Greptile has not commented (or its comments were removed), the label is removed. You may see this label appear or disappear automatically as you push commits or as Greptile completes its review.

#### Update Docs

The `update-docs` workflow runs on every push to `main`. It examines the diff, identifies new or changed APIs, functions, configuration, and other user-visible behaviour, and opens a draft pull request with the corresponding documentation updates. If documentation is already up to date, it does nothing. Merge or close these PRs as you would any human-authored documentation PR.

#### Portfolio Analyst

The `portfolio-analyst` workflow runs every Monday at approximately 09:00 UTC. It downloads up to 30 days of agentic workflow logs and analyzes them for cost reduction opportunities (targeting 20%+ savings) and reliability improvements. Results are published as a GitHub Discussion in the **audits** category. Previous discussions from this workflow are closed automatically when a new one is created. You do not need to act on these discussions unless you want to adjust a workflow configuration.

#### Static Analysis Report

The `static-analysis-report` workflow runs daily. It scans agentic workflow files for security vulnerabilities using [zizmor](https://github.com/woodruffw/zizmor), [poutine](https://github.com/boostsecurityio/poutine), and [actionlint](https://github.com/rhysd/actionlint). Findings are posted as a GitHub Discussion in the **security** category. The workflow closes its previous discussion when it opens a new one to keep the list tidy.

#### Artifacts Summary

The `artifacts-summary` workflow runs every Sunday at approximately 06:00 UTC. It generates a report of GitHub Actions artifact usage (names, sizes, retention policies) across all workflows in the repository. The report is published as a GitHub Discussion in the **artifacts** category. Use it to identify large or stale artifacts that could be cleaned up to reduce storage costs.

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

#### Metrics Collector

The `metrics-collector` workflow runs daily and gathers performance metrics for the entire agentic workflow ecosystem. It reads workflow run history, job durations, success/failure rates, and token usage, then writes structured JSON data to a dedicated `memory/meta-orchestrators` branch under `metrics/`. This data is intended for meta-orchestrator workflows for trend analysis, cost tracking, and health monitoring. You do not need to interact with these files directly.

#### Schema Consistency Checker

The `schema-consistency-checker` workflow runs daily and detects inconsistencies across three sources of truth:

- The agentic workflow JSON schema (as defined in this repository's schema definitions)
- The parser and compiler implementation (the Go packages that load, validate, and execute workflow definitions)
- The documentation (Markdown docs) and workflow definition files (for example, `.github/workflows/*.yml`)

Findings are published as a GitHub Discussion in the **audits** category with a `[Schema Consistency]` title prefix. Previous discussions are closed automatically when a new one is created. If a finding affects Biblioteka's workflow definitions or documentation, address it as you would any other documentation inconsistency.

#### Claude Code User Docs Review

The `claude-code-user-docs-review` workflow runs every day at 08:00 UTC. It adopts the perspective of a developer who uses Claude Code but does not have a GitHub Copilot subscription. It reads `README.md` and the `docs/` directory, then evaluates the documentation for clarity, completeness, and accessibility to non-Copilot users. Findings are published as a GitHub Discussion in the **audits** category. The previous discussion from this workflow is closed automatically when a new one is created.

Review these discussions to identify documentation gaps that may block contributors who don't use Copilot.

#### Daily Assign Issue to User

The `daily-assign-issue-to-user` workflow runs every day on a schedule. It finds one open, unassigned issue and assigns it to an active contributor. After assigning, it adds a comment to the issue notifying the contributor. This ensures issues don't remain unassigned indefinitely without requiring manual triage.

#### Daily Code Metrics

The `daily-code-metrics` workflow runs every day. It measures code-health indicators (lines of code, test coverage, complexity, churn, and similar metrics), generates trend charts over a 30-day window, and publishes the results as a GitHub Discussion in the **audits** category. Historical data is stored in the `memory/` branch so that trend comparisons remain accurate between runs. The previous discussion from this workflow is closed automatically.

#### Daily Copilot Token Report

The `daily-copilot-token-report` workflow runs on weekdays at 11:00 UTC. It downloads logs from all agentic workflows over the previous 30 days, aggregates token consumption by workflow and engine, calculates approximate costs, and identifies usage trends. Results are published as a GitHub Discussion in the **audits** category. The previous discussion from this workflow is closed automatically.

Review these reports to monitor AI spending across workflows and identify expensive or runaway workflows before costs accumulate.

#### Daily Doc Updater

The `daily-doc-updater` workflow runs every day at 06:00 UTC. It reviews recent code changes and the existing documentation for gaps, inaccuracies, and outdated content. When it identifies documentation that needs updating, it opens a draft pull request with the title prefix `[docs]`. Draft PRs expire after one day if not merged. Review and merge these pull requests to keep documentation in sync with the codebase.

This is a complement to the event-driven `update-docs` workflow (which fires on every push to `main`): `update-docs` documents specific code changes immediately, while `daily-doc-updater` sweeps for broader documentation drift on a schedule.

#### Daily Issues Report

The `daily-issues-report` workflow runs every day. It retrieves the most recent 1,000 issues, clusters them by theme, computes key metrics (open rate, resolution time, label distribution), and generates trend charts. The report is published as a GitHub Discussion in the **audits** category. The previous discussion from this workflow is closed automatically.

Use these reports to spot patterns in user-reported bugs or feature requests, or to identify recurring problems that may warrant broader fixes.

#### Daily Multi-Device Docs Tester

The `daily-multi-device-docs-tester` workflow runs every day. It opens the documentation site using Playwright at three device widths — mobile (375 px), tablet (768 px), and desktop (1280 px) — and checks for layout breakage, unreadable text, inaccessible interactive elements, and missing responsive behaviour. Any failures are reported as GitHub issues. Full test results are uploaded as Actions artifacts and retained for 2 days.

When assigned a docs-tester issue, inspect the attached artifact for screenshots and check the affected page at the reported viewport width.

#### Daily Observability Report

The `daily-observability-report` workflow runs every day. It analyzes the last 7 days of workflow run logs to assess logging and telemetry coverage across all agentic workflows. It specifically looks at the AWF firewall and MCP Gateway layers, checks structured-logging completeness, and flags workflows with poor observability. Findings are published as a GitHub Discussion in the **audits** category. The previous discussion from this workflow is closed automatically.

#### Daily Performance Summary

The `daily-performance-summary` workflow runs every day. It queries GitHub for repository activity over a 90-day window — pull request cycle time, issue resolution time, workflow success rates, and contributor velocity — then generates trend charts. The report is published as a GitHub Discussion in the **audits** category. The previous discussion from this workflow is closed automatically.

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
| `/q` | Issue or PR comment | Answers questions about the codebase, analyzes agentic workflow performance, and can open pull requests with workflow optimizations. Also triggered by a 🚀 reaction on a comment. |

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
