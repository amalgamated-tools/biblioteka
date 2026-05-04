# Test Improver Memory — biblioteka

## Last Updated
2026-05-04

## Build/Test/Coverage Commands

### Go Tests
```bash
GOTOOLCHAIN=local go test ./...          # requires go1.26.1 to be installed
go test -v ./...                         # verbose
go test -coverprofile=/tmp/coverage.out -coverpkg=./... -count=1 -timeout=20m ./...  # with coverage
go tool cover -func=/tmp/coverage.out    # function-level report
```

### Coverage Pipeline
- A `weekly-coverage-summary.md` CI workflow generates weekly coverage trends in discussions.
- The CI installs `exiftool` (`sudo apt-get install -y libimage-exiftool-perl`) before running tests - required by metadata extraction tests.
- Frontend: `cd frontend && npm install && node_modules/.bin/vitest run`
- npm install required if node_modules/.bin/vitest missing

### Formatting/Lint
```bash
make fmt       # go fmt ./...
make hardfmt   # strict formatting (gofumpt)
make lint      # golangci-lint run ./...
cd frontend && node_modules/.bin/eslint src/
cd frontend && node_modules/.bin/prettier --write .
```

## Agent Environment Note
Go 1.26.2 is available (GOTOOLCHAIN=auto auto-downloads it). Tests run successfully.
go.mod requires go >= 1.26.2; use GOTOOLCHAIN=auto.
pnpm not available; use npm install instead.

## Testing Notes
- Uses testify/require for all assertions (not t.Fatal/t.Fatalf)
- Test helper: `internal/testutils` (MakeTestEPUB, MakeTestPDF)
- DB tests use real SQLite with WAL mode + foreign_keys=ON
- Test DB helper: `internal/db/testhelper_test.go`
- Package tests use `package <pkg>` (internal), enabling access to unexported functions
- Go 1.24+ `t.Context()` is used throughout (not `context.Background()`)
- handlers package has `testsetup_test.go` with `newTestDB`, `newTestJWT`, `withUserID`
- errorResponse struct is `{Error string "json:\"error\""}` in handlers/response.go
- CreateOIDCUser(ctx, name, email, oidcSubject) creates user with empty PasswordHash
- httptest.NewRequest sets r.Host = "example.com" by default when no Host header present
- Sub-resource handler tests call h.HandleBookRoutes(w, r) with the full path
- readingListDTO is in reading_lists.go; AddBookToReadingList(ctx, listID, userID, bookID)
- authstore package has own `newTestDB` helper in adapter_test.go
- Frontend tests: use vitest + vi.stubGlobal("fetch", fetchMock) pattern
- Frontend testUtils.ts has mockFetchResponse and mockNoContentResponse helpers
- Frontend tests import from "../api" (the barrel file) not individual modules
- pathparser uses `package pathparser` (not _test), so unexported helpers are accessible
- Frontend vi.mock("./api", ...) pattern works for mocking API module functions in lib/ tests
- autofocusFirstButton action is mocked as: `() => ({ destroy: () => {} })` in tests
- types.ts split into types/ directory (annotation, audit, auth, book, calibre, config, library, metadata, reading) — tests still import from "../../types" via index.ts barrel
- Svelte 5 $effect fires async on mount; use `await tick(); await tick()` to drain effects in tests
- UsersTab mock pattern: module-level vi.mock with all API functions; use vi.mocked(fn).mockResolvedValue per test in beforeEach

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
DB package has 46 test files for 40 source files — extremely thorough.
Frontend: 1103+ tests pass across 92+ test files.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/storage    (pure interface, no implementation)
- internal/timeutil   (one-liner NowRFC3339, trivial)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

Frontend API modules: all covered
Frontend stores: all covered
Frontend components: all covered
Handler files without own test files (covered by shared test files): book_crud.go, book_dto.go, crud.go, dberrors.go, doc.go, group_lists.go, group_members.go, group_progress.go, groups_crud.go, metadata_dto.go, metadata_goodreads.go, metadata_sse.go, request.go, tokens_compat.go, validate.go

## Task Round-Robin Status
- 2026-05-01: Tasks 1, 2, 7 (validate commands, scan opportunities, monthly issue)
- 2026-05-03: Tasks 4, 3, 7 (no open PRs; created registration config API test PR)
- 2026-05-04: Tasks 2, 3, 4, 7 (scanned May additions, UsersTab toggle tests, monthly issue)
- Next run: Tasks 1, 5, 6, 7

## Testing Backlog (prioritized)

1. ~~**pathparser internal helpers**~~ — PR #2439 merged ✅
2. ~~**library handler 409 conflict**~~ — PR #2464 merged ✅
3. ~~**authRequiredErrors + authFeatureFlags**~~ — PR #2519 merged ✅
4. ~~**enrich_ai.go error paths**~~ — PR #2546 merged ✅
5. ~~**DeleteConfirmation component**~~ — PR #2567 merged ✅
6. ~~**kobo metadata branch coverage**~~ — PR #2592 merged ✅
7. ~~**WithTx and deferRollback**~~ — PR #2654 merged ✅
8. ~~**kobo sync pagination**~~ — PR #2690 merged ✅
9. ~~**registration config API frontend tests**~~ — PR #2797 open (pending merge)
10. ~~**UsersTab registration toggle tests**~~ — PR open (pending merge)
11. **organize path-escape defense-in-depth** — `filepath.Rel` escape guard has no dedicated test. Unreachable in normal operation. Low value.
12. **Scan May additions** — scan each run for newly added features lacking test coverage

## Maintainer Priorities
- All previous monthly issues closed by veverkap as "completed"
- Signals strong positive reception; maintainer is actively merging Test Improver PRs
- Merged: #1689, #1771, #1792, #1845, #1943, #2021, #2143, #2221, #2349, #2403, #2439, #2464, #2519, #2546, #2567, #2592, #2654, #2690

## Completed Work

### 2026-05-04
- Tasks: 2 (scanned May additions), 3 (UsersTab registration toggle tests), 4 (Go tests verified), 7 (created new May issue)
- Created PR branch test-assist/users-tab-registration-tests: 8 new tests for registration config toggle in UsersTab
- UsersTab.test.ts: 4 → 12 tests; covers toggleRegistration, API calls, success/error feedback
- Gap: #2731 added registration toggle UI to UsersTab but tests only mocked the API without exercising it

### 2026-05-03
- Tasks: 4 (no open PRs), 3 (frontend registration config API tests), 7 (created new May issue)
- Created PR branch test-assist/registration-config-api-tests: 2 tests for getRegistrationConfig/setRegistrationConfig
- Config.test.ts: 10 → 12 tests
- Gap discovered: #2731 added registration config API functions but didn't update config.test.ts

### 2026-05-01
- Tasks: 1 (validated commands, all pass), 2 (scanned recent refactors, no high-value gaps), 7 (closed April #2404, created May issue #2721)
- Frontend: 1103 tests pass (92 test files)
- Go: organize, ssrf, auth packages all pass
- Scanned: types.ts split, bcrypt consolidation, getPendingAIEnrichmentOrErr, validateSSRFURL extraction — all well-covered

### 2026-04-30
- Tasks: 4, 5, 6, 3, 7; PR #2690 merged: 3 new tests for Kobo sync pagination

### Earlier (2026-04-11 to 2026-04-29)
- PRs #1689, #1771, #1792, #1845, #1943, #2021, #2143, #2221, #2349, #2403, #2439, #2464, #2519, #2546, #2567, #2592, #2654, #2690 — all merged ✅
