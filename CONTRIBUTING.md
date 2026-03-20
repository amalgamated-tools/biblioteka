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
  handlers/          # HTTP request handlers
  jobs/              # Background job definitions
  metadata/          # EPUB/MOBI/AZW3/PDF metadata extraction via ExifTool
  server/            # HTTP server setup, routing, embedded frontend
  worker/            # asynq-based background job processing
  otel/              # Logging and tracing setup
  otelkeys/          # Shared log/telemetry field-name constants
  telemetry/         # Anonymous usage telemetry (opt-in)
  testutils/         # Test helpers: MakeTestEPUB, MakeTestPDF (used in _test.go files only)
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

# Run in headed mode to watch the browser
pnpm exec playwright test --headed
```

> **Note:** When `CI=true` (set automatically in GitHub Actions), Playwright always starts a fresh server. Locally, it reuses a running server on port `3847` if one is already available.

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

### IDE and Editor Support

#### VS Code

The repository includes a `.vscode/launch.json` with three ready-to-use **Run and Debug** configurations (`Ctrl+Shift+D` / `⇧⌘D`):

| Configuration | Binary | What it does |
|---|---|---|
| **Run CLI (Folder)** | `cmd/cli/main.go` | Runs `scan-directory books/` — scans the sample `books/` directory and imports all supported files |
| **Run CLI (File)** | `cmd/cli/main.go` | Runs `process-file` against `books/Alice's Adventures in Wonderland by Lewis Carroll.epub` |
| **Run Server** | `cmd/server/main.go` | Starts the Biblioteka HTTP server |

All three configurations load environment variables from `.env` in the workspace root. Copy `.env.sample` to `.env` and fill in the required values before launching:

```bash
cp .env.sample .env
# Edit .env with your local values (DATABASE_URL, REDIS_URL, JWT_SECRET, …)
```

> **Sample books**: Two EPUB files in `books/` serve as realistic test data for the CLI configurations. They are version-controlled so the "Run CLI" launch configs work out of the box without any setup.

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

The workflow runs six jobs. All four leaf jobs start in parallel at the beginning of every run:

```
frontend-build ──┐
                 └──► frontend-all (gate)
frontend-checks ─┘

go-lint ──┐
          └──► go-all (gate)
go-test ──┘
```

| Job | Depends on | What it does |
|---|---|---|
| `frontend-build` | — | Installs pnpm deps (cached), runs `pnpm run build` |
| `frontend-checks` | — | Installs pnpm deps (cached), runs TypeScript check (`pnpm run check`), Prettier format check, ESLint (`pnpm run lint`), and frontend unit tests |
| `frontend-all` | `frontend-build` + `frontend-checks` | Gate job — fails the run if either frontend job failed |
| `go-lint` | — | Runs golangci-lint and Go format check (`gofmt`) |
| `go-test` | — | Installs `exiftool`, runs `go test -v ./...` |
| `go-all` | `go-lint` + `go-test` | Gate job — fails the run if either Go job failed |

All six jobs run fully in parallel (the two gate jobs wait for their pair). Total CI time is roughly `max(frontend-build, frontend-checks, go-lint, go-test)`.

Both frontend jobs use pnpm's built-in cache via `actions/setup-node` (`cache: 'pnpm'`, keyed on `frontend/pnpm-lock.yaml`) to avoid re-downloading the dependency tree on every run.

> **Note:** Pull requests that only touch documentation files (e.g. `README.md`, `CONTRIBUTING.md`, `docs/`) will not trigger the test workflow. If you need CI to run on a docs-only PR, trigger it manually via **Actions → Test → Run workflow**.

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
| `e2e` | — | Installs pnpm deps (cached), builds the frontend, compiles the Go binary, installs Playwright + Chromium (cached), runs `pnpm test` with `CI=true` (Playwright starts a fresh server for the run) |

On completion, the `playwright-report/` artifact is uploaded and retained for **7 days**, giving you screenshots and traces for any failures.

> **Note:** Pull requests that only touch documentation files (e.g. `README.md`, `CONTRIBUTING.md`, `docs/`) will not trigger the E2E workflow. If you need E2E tests to run on a docs-only PR, trigger the workflow manually via **Actions → E2E Tests → Run workflow**.

### Automated agentic workflows

Biblioteka uses a set of AI-powered workflows (GitHub Agentic Workflows) that run on a schedule or in response to events. They analyze the codebase, find issues, and open pull requests or GitHub issues automatically. You do not need to trigger them manually.

| Workflow | Trigger | Output |
|---|---|---|
| **Daily Accessibility Review** | Every 3 hours | GitHub issues labeled `a11y`, `automated-analysis` |
| **Code Simplifier** | Daily | Pull requests simplifying recently changed code |
| **Issue Triage** | On every new issue | Applies type/priority labels, flags duplicates, asks clarifying questions |
| **Daily File Diet** | On demand | GitHub issues for source files that exceed healthy size thresholds |
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
