---
# Go Source Code Analysis Base
# Bundles Serena Go LSP analysis + standard bash permissions for Go source navigation.
#
# Usage:
#   imports:
#     - shared/go-source-analysis.md

imports:
  - shared/mcp/serena-go.md
  - shared/reporting.md

tools:
  bash:
    - "find internal -name '*.go' ! -name '*_test.go' -type f"
    - "find internal -type f -name '*.go' ! -name '*_test.go'"
    - "find internal/ -maxdepth 1 -ls"
    - "find internal/workflow/ -maxdepth 1 -ls"
    - "wc -l internal/**/*.go"
    - "head -n * internal/**/*.go"
    - "grep -r 'func ' internal --include='*.go'"
    - "cat internal/**/*.go"
---

## Go Source Code Analysis Setup

Serena Go LSP analysis is configured for this workspace. Standard bash tools for Go source navigation are available.

### Bash Navigation Tools

Use these bash tools to supplement Serena's semantic analysis:

- `find internal -name '*.go' ! -name '*_test.go' -type f` — list all non-test Go source files
- `find internal/ -maxdepth 1 -ls` / `find internal/workflow/ -maxdepth 1 -ls` — explore directory structure
- `wc -l internal/**/*.go` — measure file sizes
- `head -n * internal/**/*.go` / `cat internal/**/*.go` — read file contents
- `grep -r 'func ' internal --include='*.go'` — find all function definitions
