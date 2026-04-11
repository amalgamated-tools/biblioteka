# Test Improver Memory — biblioteka

## Last Updated
2026-04-11

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

## Testing Landscape
Codebase is very well tested overall. Most packages have 1:1 test file ratio.
Packages with no test files (intentional):
- internal/otel       (bootstrap/init code)
- internal/otelkeys   (pure constants)
- internal/testutils  (test helpers, used only in tests)
- internal/errorfcheck/testdata, internal/slogcheck/testdata (test data dirs)

## Testing Backlog (prioritized)

1. **[DONE in PR] sanitizeDirName unit tests** — direct tests for security-relevant char filtering in organize package. Indirectly tested before; now explicitly documented. Low maintenance burden.
2. **sanitizeDirName path traversal test** — the `filepath.Rel` escape check in organize.go has no test. The check may be unreachable in practice (sanitizer removes / and \, TrimLeft removes leading dots), but a test would verify the defense-in-depth behavior. Could use a crafted stat+path workaround.
3. **smtp/send.go** — no tests for newClientWithContext or Send. These require real TCP connections, making unit tests difficult. Could use a test server.
4. **handlers/middleware** — 4 source files, 1 test file; but the 1 test file already covers all 4 source files comprehensively.

## Maintainer Priorities
Not yet observed. Will monitor PR feedback.

## Completed Work

### 2026-04-11 Run 1
- Created PR: `test(organize): add direct unit tests for sanitizeDirName`
- Discovered commands and testing landscape
- Created monthly activity issue for April 2026
