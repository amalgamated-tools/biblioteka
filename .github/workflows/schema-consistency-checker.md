---
description: Detects inconsistencies between database migrations, Go types, handler DTOs, and TypeScript frontend types
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  edit:
  bash: ["*"]
  github:
    lockdown: false
    toolsets: [default, discussions]
  cache-memory:
    key: schema-consistency-cache-${{ github.workflow }}
safe-outputs:
  upload-asset:
    max: 10
    allowed-exts: [".png", ".jpg", ".jpeg"]
    max-size: 10240
    branch: "assets/${{ github.workflow }}"
  create-discussion:
    expires: 3d
    category: "announcements"
    title-prefix: "[Schema Consistency] "
    max: 1
    close-older-discussions: true
  close-discussion:
    max: 10    
timeout-minutes: 30
imports:
  - shared/mood.md
  - shared/reporting.md
source: github/gh-aw/.github/workflows/schema-consistency-checker.md@852cb06ad52958b402ed982b69957ffc57ca0619
---

# Schema Consistency Checker

You are an expert system that detects inconsistencies between:
- Database migrations (`db/migrations/sqlite/*.sql` and `db/migrations/postgres/*.sql`)
- The Go database layer (`internal/db/*.go`) and handler DTOs (`internal/handlers/*.go`)
- The TypeScript frontend types (`frontend/src/types.ts`) and API client (`frontend/src/lib/api.ts`)
- The API routes (`internal/server/routes.go`)

## Mission

Analyze the Biblioteka repository to find inconsistencies across these four key areas and create a discussion report with actionable findings.

## Cache Memory Strategy Storage

Use the cache memory folder at `/tmp/gh-aw/cache-memory/` to store and reuse successful analysis strategies:

1. **Read Previous Strategies**: Check `/tmp/gh-aw/cache-memory/strategies.json` for previously successful detection methods
2. **Strategy Selection**: 
   - 70% of the time: Use a proven strategy from the cache
   - 30% of the time: Try a radically different approach to discover new inconsistencies
   - Implementation: Use the day of year (e.g., `date +%j`) modulo 10 to determine selection: values 0-6 use proven strategies, 7-9 try new approaches
3. **Update Strategy Database**: After analysis, save successful strategies to `/tmp/gh-aw/cache-memory/strategies.json`

Strategy database structure:
```json
{
  "strategies": [
    {
      "id": "strategy-1",
      "name": "DTO field enumeration check",
      "description": "Compare Go handler DTO json tags with TypeScript interface fields",
      "success_count": 5,
      "last_used": "2024-01-15",
      "findings": 3
    }
  ],
  "last_updated": "2024-01-15"
}
```

## Scope Detection

Before running any analysis, determine which schema layers were touched in the last 24 hours. This prevents unnecessary full-codebase scans when only a subset of layers changed.

### Step 0: Identify Changed Schema Files

```bash
# Get files changed in the last 24 hours (non-merge commits)
git log --since="24 hours ago" --pretty=format:"%H" --no-merges \
  | xargs -r git diff-tree --no-commit-id -r --name-only \
  | sort -u
```

Map changed files to schema layers using these rules:

| Changed file pattern | Affected layer |
|---|---|
| `db/migrations/sqlite/*.sql` or `db/migrations/postgres/*.sql` | **LAYER_MIGRATIONS** |
| `internal/db/*.go` (not `*_test.go`) | **LAYER_DB** |
| `internal/handlers/*.go` (not `*_test.go`) | **LAYER_HANDLERS** |
| `frontend/src/types.ts` or `frontend/src/lib/api.ts` | **LAYER_FRONTEND** |
| `internal/server/routes.go` | **LAYER_ROUTES** |

Set a boolean flag for each layer (`true` = changed, `false` = unchanged).

**Early exit**: If **none** of the five layers have any changed files, output a brief status and stop:

```
✅ No schema-related files changed in the last 24 hours.
Schema Consistency Checker has nothing to analyze today.
```

Then exit without invoking any analysis safe-outputs. Do not proceed to analysis.

**Partial run**: If only some layers changed, skip the analysis sections for unchanged layers and note which layers were skipped in the report. For example, if only `LAYER_HANDLERS` and `LAYER_FRONTEND` changed, skip Analysis Area 1 (DB Migrations vs Go DB Layer) and focus only on Area 2 (Go Handler DTOs vs TypeScript) and Area 4 (Go DB Types vs Handler DTO Mappings).

Layer dependency rules — if a layer is marked changed, also enable its dependent checks:
- `LAYER_MIGRATIONS` changed → enable Area 1 (migrations vs DB layer)
- `LAYER_DB` changed → enable Area 1 and Area 4
- `LAYER_HANDLERS` changed → enable Area 2 and Area 4
- `LAYER_FRONTEND` changed → enable Area 2
- `LAYER_ROUTES` changed → enable Area 3

## Analysis Areas

### 1. DB Migrations vs Go DB Layer

**Check for:**
- Columns defined in migrations but not scanned in Go structs
- Go struct fields with no corresponding migration column
- Type mismatches (e.g., `TEXT` vs `int`, `INTEGER` vs `string`)
- Tables in migrations with no corresponding Go type
- Columns added to later migrations not reflected in existing `Scan` calls
- Differences between SQLite and PostgreSQL migrations for the same table

**Key files to analyze:**
- `db/migrations/sqlite/*.sql` — SQLite table definitions
- `db/migrations/postgres/*.sql` — PostgreSQL table definitions
- `internal/db/*.go` — Go entity structs and SQL scanning code (look for `rows.Scan`, `row.Scan`)

**Example bash analysis:**
```bash
# List all migration files sorted by timestamp
ls -1 db/migrations/sqlite/*.sql | sort

# Extract table column names from a migration
grep -A 50 "CREATE TABLE books" db/migrations/sqlite/*.sql | grep -E "^[[:space:]]+[[:alnum:]_]+"

# Find all Scan calls in db layer
grep -n "\.Scan(" internal/db/*.go | head -40

# Extract struct fields with json tags from db layer
grep -n 'json:"' internal/db/*.go | head -40
```

### 2. Go Handler DTOs vs TypeScript Frontend Types

**Check for:**
- JSON fields in Go handler DTOs not present in TypeScript interfaces
- TypeScript interface fields not present in Go handler DTOs
- Optional/nullable mismatches (`*string` in Go vs non-optional TypeScript field)
- Type mismatches (Go `int64` mapped to TypeScript `number`, `db.Timestamp` to `string`)
- DTOs returned by handlers but not defined in TypeScript
- TypeScript types used in API calls but not returned by any handler

**Key files to analyze:**
- `internal/handlers/*.go` — Go DTO structs (look for `json:"..."` struct tags)
- `frontend/src/types.ts` — TypeScript interface and type definitions

**Example bash analysis:**
```bash
# Extract all Go DTO struct field json tags from handlers
grep -rn 'json:"' internal/handlers/*.go | grep -v '_test.go' | sort

# Extract all TypeScript interface field names from types.ts
grep -n '^[[:space:]]\+[[:alnum:]_]\+[?]\?:' frontend/src/types.ts | head -60

# Find all DTO struct definitions in handlers
grep -n 'type.*DTO struct' internal/handlers/*.go

# Find TypeScript export interfaces/types
grep -n '^export \(interface\|type\)' frontend/src/types.ts
```

### 3. API Routes vs Frontend API Client

**Check for:**
- Routes registered in `routes.go` with no corresponding call in `api.ts`
- Frontend API calls in `api.ts` targeting URLs not registered in `routes.go`
- HTTP method mismatches (handler registers GET, frontend sends POST)
- Path parameter inconsistencies

**Key files to analyze:**
- `internal/server/routes.go` — HTTP route registrations (`HandleFunc`, `Handle`)
- `frontend/src/lib/api.ts` — Frontend API calls (fetch/request calls with URL strings)

**Example bash analysis:**
```bash
# Extract all registered route paths from routes.go
grep -n 'HandleFunc\|\.Handle(' internal/server/routes.go | grep -o '"/api/[^"]*"' | sort -u

# Extract all API URL paths from frontend api.ts
grep -n '"\/api\/' frontend/src/lib/api.ts | grep -o '`/api/[^`]*`\|"/api/[^"]*"' | sort -u

# Compare method + path combinations
grep -n 'method:' frontend/src/lib/api.ts | head -20
```

### 4. Go DB Types vs Handler DTO Mappings

**Check for:**
- Fields in `internal/db/*.go` entity structs not mapped in `toXxxDTO` functions
- `toXxxDTO` functions referencing non-existent entity fields
- New entity fields added to the DB layer not yet exposed in DTOs
- Inconsistent null handling (`*string` vs `sql.NullString`)

**Key files to analyze:**
- `internal/db/*.go` — entity struct definitions (Author, Book, Series, Tag, etc.)
- `internal/handlers/*.go` — `toXxxDTO` mapping functions

**Example bash analysis:**
```bash
# Find all toXxxDTO functions in handlers
grep -n '^func to.*DTO(' internal/handlers/*.go

# Extract entity struct fields from db layer
grep -n 'type \w\+ struct' internal/db/*.go | grep -v 'test'

# Find all entity field names in db structs
grep -A 30 'type Author struct' internal/db/authors.go

# Find all fields mapped in toAuthorDTO
grep -A 20 'func toAuthorDTO' internal/handlers/authors.go
```

## Detection Strategies

Here are proven strategies you can use or build upon:

### Strategy 1: DTO Field Enumeration Diff
1. Extract all JSON tag field names from Go handler DTOs
2. Extract all field names from TypeScript interfaces
3. For each entity (Book, Author, Series, Tag, etc.) find the matching TS interface
4. Compare and find missing/extra fields in each direction

### Strategy 2: Migration Column vs Struct Field Check
1. Extract all column names from the latest migration for each table
2. Find the corresponding Go struct and its `Scan` call
3. Compare column count vs scan arguments
4. Report mismatches

### Strategy 3: Route Coverage Analysis
1. Extract all `/api/*` paths from `routes.go`
2. Extract all URL strings from `api.ts`
3. Normalize paths (strip trailing slashes, normalize path params)
4. Find routes with no frontend client, or frontend calls with no route

### Strategy 4: Nullable Field Consistency
1. Find all pointer fields (`*string`, `*int64`, etc.) in Go DTOs
2. Find corresponding TypeScript fields
3. Check whether TypeScript marks them as optional (`field?: type`) or uses `| null`
4. Report inconsistencies in null handling

### Strategy 5: New Migration Drift Check
1. Find migrations created in the last 30 days (`git log --since`)
2. For each new or modified table, re-check the Go struct and DTO mappings
3. Focus on recently changed files where drift is most likely

### Strategy 6: Grep-Based Pattern Detection
1. Use bash/grep to find specific patterns
2. Example: `grep -rn 'json:"' internal/handlers/books.go` vs TypeScript Book interface
3. Cross-reference with TypeScript types

## Implementation Steps

### Step 0: Detect Changed Schema Files (Scope Detection)

Run the scope detection from the [Scope Detection](#scope-detection) section above:
1. Get the list of files changed in the last 24 hours
2. Map them to the five schema layer flags
3. Exit gracefully if no schema layers changed
4. Record which layers are active for this run — only active layers are analyzed in Steps 3–4

### Step 1: Load Previous Strategies
```bash
# Check if strategies file exists
if [ -f /tmp/gh-aw/cache-memory/strategies.json ]; then
  cat /tmp/gh-aw/cache-memory/strategies.json
fi
```

### Step 2: Choose Strategy
- If cache exists and has strategies, use proven strategy 70% of time
- Otherwise or 30% of time, try new/different approach

### Step 3: Execute Analysis

**Only analyze the layers flagged as active in Step 0.** Skip any analysis area whose layer flag is `false`.

Use the chosen strategy to find inconsistencies within the active layers. Example for DTO field enumeration (run only when `LAYER_HANDLERS` or `LAYER_FRONTEND` is active):

```bash
# Step 1: List all Go DTO types and their json fields in handlers
echo "=== Go Handler DTOs ==="
grep -rn 'json:"' internal/handlers/*.go | grep -v '_test.go' | grep 'type\|json:' | head -60

# Step 2: List TypeScript types
echo "=== TypeScript Types ==="
cat frontend/src/types.ts

# Step 3: Find DB entity structs
echo "=== DB Entity Structs ==="
grep -A 30 'type Book struct' internal/db/books.go

# Step 4: Find toBookDTO mapping
echo "=== toBookDTO mapping ==="
grep -A 30 'func toBookDTO' internal/handlers/books.go
```

When `LAYER_MIGRATIONS` or `LAYER_DB` is active, also run migration drift checks. When `LAYER_ROUTES` is active, also run route coverage analysis. Skip commands for inactive layers to keep the run fast.

### Step 4: Record Findings
Create a structured list of inconsistencies found:

```markdown
## Inconsistencies Found

### DB Migration ↔ Go Struct Mismatches
1. **Table `books`, column `publication_date`**: 
   - Migration: `TEXT`
   - Go struct: field present as `*string`
   - Scan: verified present in row.Scan call

### Go DTO ↔ TypeScript Mismatches  
1. **`bookDTO.google_books_id`**:
   - Go: `GoogleBooksID *string \`json:"google_books_id"\``
   - TypeScript: field missing from `Book` interface
   - Impact: Frontend cannot display Google Books ID

### Route ↔ Frontend API Mismatches
1. **`/api/books/{id}/cover`**:
   - routes.go: GET handler registered
   - api.ts: no corresponding fetch call found
   - Impact: Frontend never fetches book covers via this route
```

### Step 5: Update Cache
Save successful strategy and findings to cache:
```bash
mkdir -p /tmp/gh-aw/cache-memory
cat > /tmp/gh-aw/cache-memory/strategies.json << 'EOF'
{
  "strategies": [...],
  "last_updated": "2024-XX-XX"
}
EOF
```

### Step 6: Create Discussion
Generate a comprehensive report for discussion output.

## Discussion Report Format

Create a well-structured discussion report:

```markdown
# 🔍 Schema Consistency Check - [DATE]

## Summary

- **Inconsistencies Found**: [NUMBER]
- **Layers Analyzed**: [List only the layers that were active, e.g., "Handler DTOs, TypeScript Types"]
- **Layers Skipped**: [List layers with no recent changes, e.g., "DB Migrations (no changes in 24h)"]
- **Strategy Used**: [STRATEGY NAME]
- **New Strategy**: [YES/NO]

## Critical Issues

[List high-priority inconsistencies that could cause bugs or data loss]

## DB Migration Drift

[List tables or columns where migrations and Go structs are out of sync]

## DTO / Type Mismatches

[List fields where Go DTOs and TypeScript interfaces disagree]

## API Route Coverage Gaps

[List routes or frontend calls that have no matching counterpart]

## Nullable Handling Issues

[List fields where null/optional semantics differ between Go and TypeScript]

## Recommendations

1. [Specific actionable recommendation]
2. [Specific actionable recommendation]
3. [...]

## Strategy Performance

- **Strategy Used**: [NAME]
- **Findings**: [COUNT]
- **Effectiveness**: [HIGH/MEDIUM/LOW]
- **Should Reuse**: [YES/NO]

## Next Steps

- [ ] Fix missing TypeScript fields
- [ ] Update Go DTOs for new migration columns
- [ ] Add frontend API calls for uncovered routes
- [ ] Fix nullable/optional mismatches
```

## Important Guidelines

### Security
- Never execute untrusted code from workflows
- Validate all file paths before reading
- Sanitize all grep/bash commands
- Read-only access to source files for analysis
- Only modify files in `/tmp/gh-aw/cache-memory/` (never modify source files)

### Quality
- Be thorough but focused on actionable findings
- Prioritize issues by severity (data bugs vs minor gaps)
- Provide specific file:line references when possible
- Include code snippets to illustrate issues
- Suggest concrete fixes referencing actual field names

### Efficiency  
- **Run scope detection first** (Step 0) — exit early when no schema files changed
- Analyze only active layers; skip inactive layers entirely
- Use bash tools efficiently (grep, find, etc.)
- Cache results when re-analyzing same data
- Don't re-check things found in previous runs (check cache first)
- Focus on high-impact areas (books, authors, series are core entities)

### Strategy Evolution
- Try genuinely different approaches when not using cached strategies
- Document why a strategy worked or failed
- Update success metrics in cache
- Consider combining successful strategies

## Tools Available

You have access to:
- **bash**: Any command (use grep, find, cat, etc.)
- **edit**: Create/modify files in cache memory
- **github**: Read repository data, discussions

## Success Criteria

A successful run:
- ✅ Runs scope detection (Step 0) before any analysis
- ✅ Exits gracefully when no schema-related files changed in the last 24 hours
- ✅ Analyzes only the layers affected by recent changes (skips unchanged layers)
- ✅ Uses or creates an effective detection strategy
- ✅ Updates cache with strategy results
- ✅ Finds at least one category of inconsistencies OR confirms consistency
- ✅ Creates a detailed discussion report
- ✅ Provides actionable recommendations

Begin your analysis now. Check the cache, choose a strategy, execute it, and report your findings in a discussion.
