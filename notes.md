# Test Improver Memory — biblioteka

## Last Updated
2026-04-17

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
- Frontend: `cd frontend && pnpm run test`

### Formatting/Lint
```bash
make fmt       # go fmt ./...
make hardfmt   # strict formatting (gofumpt)
make lint      # golangci-lint run ./...
cd frontend && pnpm run lint && pnpm run check
```

## Agent Environment Note
Go 1.26.1 is available (GOTOOLCHAIN=auto auto-downloads it). Tests run successfully.

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

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

## Task Round-Robin Status
- 2026-04-11: Tasks 1, 2, 3, 7 (discovery + sanitizeDirName PR + monthly issue)
- 2026-04-12 run 2: Tasks 4, 3, 7 (no open PRs to maintain; OIDC login regression PR; monthly issue update)
- 2026-04-12 run 3: Tasks 2, 3, 7 (auth_origin CSRF helpers PR; new monthly issue)
- 2026-04-13: Tasks 3, 7 (OIDC password change guard test PR; new monthly issue #1793 had been closed)
- 2026-04-14: Tasks 4, 6, 7 (no open Test Improver PRs; books reading-lists handler PR; new monthly issue)
- 2026-04-15: Tasks 2, 3, 7 (BuildEnrichPrompt tests PR; new monthly issue #1944 closed by maintainer)
- 2026-04-17: Tasks 3, 7 (PasskeyAdapter tests PR; new monthly issue created)
- Next run: Tasks 4, 5, 2, 7 (maintain PRs, comment on testing issues, identify opportunities)

## Testing Backlog (prioritized)

1. **db/ai_enrichments.go** — No direct DB-level test for ApplyAIEnrichment (complex transactional logic). Covered indirectly via handler tests. Medium value.
2. **db/reading_group_lists.go** — ShareListWithGroup, UnshareListFromGroup, ListGroupReadingLists have no direct DB-level tests. Low-medium value.
3. **smtp/send.go** — tests exist now (PR merged).
4. **pathparser internal helpers** — isLikelyPersonName, stripTrailingAuthor have no direct tests. Covered via ParseBookPath table tests. Low-medium value.
5. **organize path-escape defense-in-depth test** — the filepath.Rel escape guard in organize.go has no test. Likely unreachable in practice. Low value.

## Maintainer Priorities
- All previous monthly issues closed by veverkap as "completed"
- Signals strong positive reception; maintainer is actively merging Test Improver PRs
- Merged: #1689, #1771, #1792, #1845, #1943, #2021

## Completed Work

### 2026-04-17
- New monthly activity issue created
- Submitted PR on branch `test-assist/passkey-adapter-tests`: 15 tests for PasskeyAdapter (all 9 store methods: CreateChallenge, GetAndDeleteChallenge, DeleteExpiredChallenges, CreateCredential, ListCredentialsByUser, FindCredentialByCredentialID, FindCredentialByIDAndUser, UpdateCredentialData, DeleteCredential)

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
