# Test Improver Memory — biblioteka

## Last Updated
2026-04-29

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
- Frontend: `cd frontend && node_modules/.bin/vitest run`
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

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/storage    (pure interface, no implementation)
- internal/timeutil   (one-liner NowRFC3339, trivial)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

Frontend API modules: all covered
Frontend stores: all covered
Frontend components: all covered as of 2026-04-27

## Task Round-Robin Status
- 2026-04-11: Tasks 1, 2, 3, 7 (discovery + sanitizeDirName PR + monthly issue)
- 2026-04-12 run 2: Tasks 4, 3, 7 (no open PRs to maintain; OIDC login regression PR; monthly issue update)
- 2026-04-12 run 3: Tasks 2, 3, 7 (auth_origin CSRF helpers PR; new monthly issue)
- 2026-04-13: Tasks 3, 7 (OIDC password change guard test PR; new monthly issue #1793 had been closed)
- 2026-04-14: Tasks 4, 6, 7 (no open Test Improver PRs; books reading-lists handler PR; new monthly issue)
- 2026-04-15: Tasks 2, 3, 7 (BuildEnrichPrompt tests PR; new monthly issue #1944 closed by maintainer)
- 2026-04-17: Tasks 3, 7 (PasskeyAdapter tests PR; new monthly issue created)
- 2026-04-18: Tasks 4, 2, 3, 7 (no open test-assist PRs to maintain; reading_group_lists DB tests PR; new monthly issue)
- 2026-04-19: Tasks 5, 2, 3, 7 (no testing-label issues; tags-in-bookDTO regression tests PR; new monthly issue)
- 2026-04-20: Tasks 6, 4, 2, 3, 7 (no open PRs; calibre+recommendations frontend API tests PR; new monthly issue)
- 2026-04-21: Tasks 3, 7 (pathparser helper tests PR #2439; monthly issue updated)
- 2026-04-22: Tasks 4, 3, 7 (updated PR #2439 addressing 4 review comments; new library handler 409 conflict tests PR; monthly issue updated)
- 2026-04-25: Tasks 2, 3, 4, 7 (no open PR issues; enrich_ai error-path tests PR; monthly issue updated)
- 2026-04-26: Tasks 2, 3, 7 (DeleteConfirmation component direct tests PR; monthly issue updated)
- 2026-04-27: Tasks 1, 4, 5, 6, 7 (no open PR issues/conflicts; kobo metadata branch coverage PR; monthly issue updated)
- 2026-04-29: Tasks 2, 3, 7 (WithTx/deferRollback DB tests PR; monthly issue updated)
- Next run: Tasks 4, 5, 6, 7

## Testing Backlog (prioritized)

1. ~~**pathparser internal helpers**~~ — PR #2439 merged 2026-04-23 ✅
2. ~~**library handler 409 conflict**~~ — PR #2464 merged 2026-04-23 ✅
3. **authRequiredErrors + authFeatureFlags** — PR #2519 open
4. **enrich_ai.go error paths** — PR #2546 open
5. **DeleteConfirmation component** — PR #2567 open
6. **kobo metadata branch coverage** — PR #2592 open
7. **WithTx and deferRollback** — PR submitted 2026-04-29 (branch test-assist/db-with-tx-tests)
8. **organize path-escape defense-in-depth test** — the filepath.Rel escape guard in organize.go has no test. Likely unreachable in practice. Low value.
9. **SSRF dialer Class B (172.16.x.x)** — ollama/client_test.go tests Class A, C, loopback, AWS metadata but not Class B. Very low value.

## Maintainer Priorities
- All previous monthly issues closed by veverkap as "completed"
- Signals strong positive reception; maintainer is actively merging Test Improver PRs
- Merged: #1689, #1771, #1792, #1845, #1943, #2021, #2143, #2221, #2349, #2403, #2439, #2464

## Completed Work

### 2026-04-29
- Tasks: 2 (scanned for opportunities; found WithTx/deferRollback at 0% direct coverage), 3 (new PR), 7 (monthly issue updated)
- Created PR on branch `test-assist/db-with-tx-tests`:
  - 6 new tests: WithTx commit path, rollback path, error propagation, no-op fn, deferRollback ErrTxDone, deferRollback double rollback
  - All DB tests pass (19.9s)
- Tasks: 1 (commands validated, no changes needed), 4 (all 3 open PRs #2519, #2546, #2567 have no conflicts or review comments), 5 (no open testing issues), 6 (kobo metadata PR), 7 (monthly issue updated)
- Created PR on branch `test-assist/kobo-metadata-coverage`:
  - 12 new tests: ReadingStateResponse (location, partial-location guard, timestamps), BookMetadata PubDate (RFC3339/YYYY-MM-DD/year/unparseable/empty), SeriesNumber (nil/fractional/whole)
  - All 26 kobo tests pass (14 existing + 12 new)

### 2026-04-26
- Tasks: 2 (scanned for new opportunities; found DeleteConfirmation component), 3 (new PR), 7 (monthly issue updated)
- Created PR #2567 on branch `test-assist/delete-confirmation-tests`:
  - 7 tests: renders item name, role=group, aria-labelledby, onConfirm, onCancel, no cross-fire
  - All 1085 frontend tests pass (90 test files)
- Found: GroupDetail.test.ts already covers GroupEditHeader/GroupMembers/GroupSharedLists indirectly
- Found: Auth.test.ts covers LoginForm/SignupForm thoroughly through parent
- All frontend API modules, stores, and components now have tests

### 2026-04-25
- Tasks: 2 (identified enrich_ai error paths), 3 (new PR), 4 (verified PR #2519 still open, no action needed), 7 (monthly issue updated)
- Created PR on branch `test-assist/enrich-ai-error-paths` (became PR #2546):
  - 4 new tests: InvalidPayload, BookNotFound, ParseError, EmptyReadingLevelAndDescription
  - All 7 `TestEnrichAI_*` tests pass (3 existing + 4 new)
- Pre-existing failures confirmed in `TestProcessBookFile_*` on `main` (unrelated)

### 2026-04-24
- Tasks: 1 (validated commands still work), 5 (no open testing issues found), 6 (new PR), 7 (monthly issue updated)
- Created PR on branch `test-assist/auth-util-tests` (became PR #2519):
  - `authRequiredErrors.test.ts`: 12 tests covering all 8 branch combos for login/signup required-field messages
  - `authFeatureFlags.test.ts`: 7 tests for Promise.allSettled fallback logic
  - Total: 19 new tests, 1097 total passing
- Discovered new packages without tests: `internal/storage` (pure interface), `internal/timeutil` (trivial)
- Notable: vi.mock("./api", ...) pattern works for lib/ test files

### 2026-04-22
- Updated PR #2439 (pathparser helper tests): addressed all 4 review comments
- Created PR #2464: 2 handler-level 409 Conflict tests for CreateLibrary/UpdateLibrary

### 2026-04-21
- Created PR #2439: 44 test cases across 6 new test functions for pathparser helpers

### 2026-04-20
- Created PR #2403 (merged): 13 tests for calibre.ts + recommendations.ts

### 2026-04-19
- Created PR #2349 (merged): 2 regression guard tests for tags in bookDTO

### 2026-04-18
- PR #2221 merged: 11 DB-level tests for reading_group_lists

### 2026-04-17
- PR #2143 merged: 15 tests for PasskeyAdapter

### 2026-04-15
- PR #2021 merged: 7 unit tests for BuildEnrichPrompt

### 2026-04-14
- PR merged: tests for getBookReadingLists handler

### 2026-04-13 (PR #1845 merged)
- PR #1845: OIDC password change guard test

### 2026-04-12 Run 3 (PR #1792 merged)
- PR #1792: auth origin CSRF helpers tests

### 2026-04-12 Run 2 (PR #1771 merged)
- PR #1771: OIDC login enumeration regression tests

### 2026-04-11 Run 1 (PR #1689 merged)
- PR #1689: sanitizeDirName unit tests
