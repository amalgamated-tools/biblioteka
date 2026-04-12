# Test Improver Memory — biblioteka

## Last Updated
2026-04-12

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
make hardfmt   # strict formatting
make lint      # golangci-lint run ./...
cd frontend && pnpm run lint && pnpm run check
```

## Agent Environment Note
Go 1.24.13 is present but go.mod requires ≥ 1.26.1. Cannot run Go tests directly.
Use `GOTOOLCHAIN=local go version` to confirm. CI has the correct Go version.

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

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

Key untested internal helpers (tested only via high-level integration):
- pathparser: isLikelyPersonName, normalizeName, namesEqual, stripTrailingAuthor
  (these are low priority - all covered indirectly through ParseBookPath tests)
- smtp/send.go: newClientWithContext, Send (require TCP - hard to unit test)

## Task Round-Robin Status
- 2026-04-11: Tasks 1, 2, 3, 7 (discovery + sanitizeDirName PR + monthly issue)
- 2026-04-12 run 2: Tasks 4, 3, 7 (no open PRs to maintain; OIDC login regression PR; monthly issue update)
- 2026-04-12 run 3: Tasks 2, 3, 7 (auth_origin CSRF helpers PR; new monthly issue)
- Next run: Task 4 (maintain PRs), Task 5 (comment on testing issues), Task 7

## Testing Backlog (prioritized)

1. **[IN PROGRESS] auth_origin CSRF helpers** — PR on branch test-assist/auth-origin-csrf-tests
   - Tests sameOrigin, matchRequestOrigin, parseHostPort, normalizeHost, defaultPort
   - 263 lines: 14 table-driven sameOrigin cases + 3 logout integration tests
2. **organize path-escape defense-in-depth test** — the `filepath.Rel` escape guard in organize.go has no test. Likely unreachable in practice. Low value.
3. **smtp/send.go** — no tests for newClientWithContext or Send. These require real TCP connections, making unit tests difficult. Could use a test server.
4. **pathparser internal helpers** — isLikelyPersonName, stripTrailingAuthor have no direct tests. Covered via ParseBookPath table tests. Medium value if direct tests would catch regressions in heuristics.

## Maintainer Priorities
- Previous monthly issue #1690 was closed by veverkap on 2026-04-12 as "completed"
- This signals positive reception; maintainer is actively merging Test Improver PRs

## Completed Work

### 2026-04-12 Run 3
- Created new monthly activity issue for April 2026
- Created PR on branch `test-assist/auth-origin-csrf-tests`: unit tests for CSRF origin-checking helpers (sameOrigin, matchRequestOrigin, parseHostPort, normalizeHost, defaultPort)

### 2026-04-12 Run 2
- Confirmed PR #1689 (sanitizeDirName tests) was merged
- Created PR on branch `test-assist/oidc-login-enumeration-regression`: regression test for OIDC account enumeration fix (#1713) — merged as #1771 ✅
- Updated monthly activity issue #1690

### 2026-04-11 Run 1
- Created PR #1689: `test(organize): add direct unit tests for sanitizeDirName` (merged ✅)
- Discovered commands and testing landscape
- Created monthly activity issue #1690 for April 2026
