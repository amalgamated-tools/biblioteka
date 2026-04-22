# Test Improver Memory — biblioteka

## Last Updated
2026-04-22

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

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

Frontend API modules with no tests: (now all covered as of 2026-04-20)
- frontend/src/lib/api/recommendations.ts — added in this run
- frontend/src/lib/api/calibre.ts — added in this run

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
- Next run: Tasks 1, 5, 6, 7

## Testing Backlog (prioritized)

1. ~~**pathparser internal helpers**~~ — PR #2439 open, review comments addressed 2026-04-22 ✅. Also discovered: isLikelyPersonName("Special Edition") = true (false positive heuristic, documented in tests)
2. ~~**library handler 409 conflict**~~ — PR submitted 2026-04-22 (branch `test-assist/library-handler-conflict-tests`)
3. **organize path-escape defense-in-depth test** — the filepath.Rel escape guard in organize.go has no test. Likely unreachable in practice. Low value.
3. **SSRF dialer Class B (172.16.x.x)** — ollama/client_test.go tests Class A, C, loopback, AWS metadata but not Class B. Very low value (impl is the same `isPrivateIP` function; Class B is already tested in config_llm_test.go).
4. ~~**db/ai_enrichments.go**~~ — Covered by PR #2150 (merged) and PR #2204 (merged).
5. ~~**db/reading_group_lists.go**~~ — PR #2201 and #2221 both merged ✅.
6. ~~**authstore/PasskeyAdapter**~~ — PR #2143 merged ✅.
7. ~~**tags in bookDTO handler responses**~~ — Covered by 2026-04-19 PR #2349.
8. ~~**frontend/src/lib/api/calibre.ts**~~ — Covered 2026-04-20 ✅
9. ~~**frontend/src/lib/api/recommendations.ts**~~ — Covered 2026-04-20 ✅

## Maintainer Priorities
- All previous monthly issues closed by veverkap as "completed"
- Signals strong positive reception; maintainer is actively merging Test Improver PRs
- Merged: #1689, #1771, #1792, #1845, #1943, #2021, #2143, #2221, #2349, #2403

## Completed Work

### 2026-04-22
- Updated PR #2439 (pathparser helper tests): addressed all 4 review comments (Greptile + Copilot):
  - Converted `TestNamesEqual` to table-driven
  - Added explicit `name` fields to blank/whitespace subtests in `TestIsLikelyPersonName`, `TestNormalizeName`
  - Added `"1.5. Title"` nil-result test case to `TestExtractSeriesPosition` (decimal positions unsupported)
  - Fixed misleading code comment in `stripTrailingAuthor` (removed wrong "Special Edition" claim)
- Created new PR on branch `test-assist/library-handler-conflict-tests`:
  - Added `TestCreateLibrary_DuplicateName` to `libraries_create_test.go`
  - Added `TestUpdateLibrary_DuplicateName` to `libraries_update_delete_test.go`
  - Both verify HTTP 409 Conflict response when duplicate library name submitted at handler level

### 2026-04-21
- Monthly activity issue #2404 updated with new run entry
- Created PR #2439 on branch `test-assist/pathparser-helper-tests-36bb45049c5b3ec5`:
  - 6 new test functions, 44 test cases
  - Covers: isLikelyPersonName, stripTrailingAuthor, extractSeriesPosition, extractYear, normalizeName, namesEqual
  - Notable find: isLikelyPersonName has a known false positive for "Special Edition" type suffixes

### 2026-04-20
- New monthly activity issue created (prior #2343 closed by veverkap)
- Created PR on branch `test-assist/frontend-api-calibre-recommendations-tests` (merged as #2403):
  - 8 tests for calibre.ts (previewCalibreImport, confirmCalibreImport + path variants)
  - 5 tests for recommendations.ts (getRecommendations)
  - Total: 13 new tests

### 2026-04-19
- New monthly activity issue created (prior #2222 closed by veverkap)
- Created PR on branch `test-assist/book-dto-tags-regression-tests`: 2 tests verifying
  tags field is embedded in GetBook and CreateBook handler responses (regression guard for dd15bdc9)
  (merged as PR #2349)

### 2026-04-18 (PR #2221 — merged ✅)
- New monthly activity issue created (prior #2144 closed by veverkap)
- Created PR on branch `test-assist/reading-group-list-db-tests`: 11 DB-level tests for ShareListWithGroup, UnshareListFromGroup, ListGroupReadingLists

### 2026-04-17 (PR #2143 — merged ✅)
- New monthly activity issue #2144 created
- Submitted PR #2143 on branch `test-assist/passkey-adapter-tests`: 15 tests for PasskeyAdapter

### 2026-04-15
- New monthly activity issue created (prior #1944 closed by maintainer)
- Submitted PR #2021 on branch `test-assist/llm-prompt-tests`: 7 unit tests for BuildEnrichPrompt

### 2026-04-14
- New monthly activity issue created (prior #1846 closed by maintainer)
- Submitted PR on branch `test-assist/books-reading-lists-handler`: tests for getBookReadingLists

### 2026-04-13 (PR #1845 — merged ✅)
- Submitted PR on branch `test-assist/oidc-password-change-guard`

### 2026-04-12 Run 3 (PR #1792 — merged ✅)
- Created PR on branch `test-assist/auth-origin-csrf-tests`

### 2026-04-12 Run 2 (PR #1771 — merged ✅)
- Created PR on branch `test-assist/oidc-login-enumeration-regression`

### 2026-04-11 Run 1 (PR #1689 — merged ✅)
- Created PR #1689: `test(organize): add direct unit tests for sanitizeDirName`
