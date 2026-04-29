---
description: Detects inconsistencies between JSON schema, implementation code, and documentation
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  discussions: read
  issues: read
  pull-requests: read
<<<<<<< current (local changes)
engine: copilot
||||||| base (original)
engine: claude
=======
engine:
  id: claude
  max-turns: 60
>>>>>>> new (upstream)
tools:
  edit:
  bash: ["*"]
  github:
    mode: remote
    toolsets: [default, discussions]
  cache-memory:
    key: schema-consistency-cache-${{ github.workflow }}
timeout-minutes: 30
checkout:
  - fetch-depth: 1
    current: true
imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[Schema Consistency] "
      expires: 1d
pre-agent-steps:
  - name: Pre-compute schema analysis data
    run: |
      set -e
      mkdir -p /tmp/gh-aw/agent

      echo "=== Extracting schema fields ==="

      # 1. All top-level fields in the main JSON schema
      SCHEMA_FIELDS=$(jq -r '.properties | keys[]' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null | sort -u || echo "")

      # 2. yaml-tagged struct fields in pkg/parser/*.go
      PARSER_YAML_FIELDS=$(grep -rh 'yaml:"' pkg/parser/*.go 2>/dev/null \
        | grep -o 'yaml:"[^"]*"' \
        | sed 's/yaml:"//;s/"//' \
        | sed 's/,omitempty//' \
        | sed 's/,.*$//' \
        | grep -v '^-$' \
        | grep -v '^$' \
        | sort -u || echo "")

      # 3. yaml-tagged struct fields in pkg/workflow/*.go
      WORKFLOW_YAML_FIELDS=$(grep -rh 'yaml:"' pkg/workflow/*.go 2>/dev/null \
        | grep -o 'yaml:"[^"]*"' \
        | sed 's/yaml:"//;s/"//' \
        | sed 's/,omitempty//' \
        | sed 's/,.*$//' \
        | grep -v '^-$' \
        | grep -v '^$' \
        | sort -u || echo "")

      # 4. Top-level frontmatter keys actually used in workflow .md files
      USED_FIELDS=$(grep -rh '^[a-z][a-z0-9_-]*:' .github/workflows/*.md 2>/dev/null \
        | sed 's/:.*//' \
        | grep -v '^#' \
        | sort -u || echo "")

      # 5. Schema field types for all top-level fields
      FIELD_TYPES=$(jq -r '.properties | to_entries[] |
        "\(.key): \(.value.type // (.value.anyOf // .value.oneOf // [] | map(.type // "complex") | unique | join("|")) // "complex")"' \
        pkg/parser/schemas/main_workflow_schema.json 2>/dev/null | sort || echo "")

      # 6. Fields in schema but absent as yaml tags in parser structs
      IN_SCHEMA_NOT_PARSER=$(comm -23 \
        <(echo "$SCHEMA_FIELDS") \
        <(echo "$PARSER_YAML_FIELDS" | sort -u) 2>/dev/null || echo "")

      # 7. yaml tags in parser structs absent from schema
      IN_PARSER_NOT_SCHEMA=$(comm -23 \
        <(echo "$PARSER_YAML_FIELDS" | sort -u) \
        <(echo "$SCHEMA_FIELDS") 2>/dev/null || echo "")

      # 8. Fields in schema but absent from workflow compiler structs
      IN_SCHEMA_NOT_WORKFLOW=$(comm -23 \
        <(echo "$SCHEMA_FIELDS") \
        <(echo "$WORKFLOW_YAML_FIELDS" | sort -u) 2>/dev/null || echo "")

      # 9. Fields used in actual workflow .md files but not in schema
      IN_USED_NOT_SCHEMA=$(comm -23 \
        <(echo "$USED_FIELDS" | sort -u) \
        <(echo "$SCHEMA_FIELDS") 2>/dev/null || echo "")

      # Write JSON output
      jq -n \
        --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg schema_fields "$SCHEMA_FIELDS" \
        --arg parser_yaml_fields "$PARSER_YAML_FIELDS" \
        --arg workflow_yaml_fields "$WORKFLOW_YAML_FIELDS" \
        --arg used_in_workflows "$USED_FIELDS" \
        --arg field_types "$FIELD_TYPES" \
        --arg in_schema_not_parser "$IN_SCHEMA_NOT_PARSER" \
        --arg in_parser_not_schema "$IN_PARSER_NOT_SCHEMA" \
        --arg in_schema_not_workflow "$IN_SCHEMA_NOT_WORKFLOW" \
        --arg in_used_not_schema "$IN_USED_NOT_SCHEMA" \
        '{
          generated_at: $generated_at,
          schema_fields: ($schema_fields | split("\n") | map(select(. != ""))),
          parser_yaml_fields: ($parser_yaml_fields | split("\n") | map(select(. != ""))),
          workflow_yaml_fields: ($workflow_yaml_fields | split("\n") | map(select(. != ""))),
          used_in_workflows: ($used_in_workflows | split("\n") | map(select(. != ""))),
          field_types: ($field_types | split("\n") | map(select(. != ""))),
          field_gaps: {
            in_schema_not_parser: ($in_schema_not_parser | split("\n") | map(select(. != ""))),
            in_parser_not_schema: ($in_parser_not_schema | split("\n") | map(select(. != ""))),
            in_schema_not_workflow: ($in_schema_not_workflow | split("\n") | map(select(. != ""))),
            in_used_not_schema: ($in_used_not_schema | split("\n") | map(select(. != "")))
          }
        }' > /tmp/gh-aw/agent/schema-diff.json

      echo "✓ Schema diff written to /tmp/gh-aw/agent/schema-diff.json"
      echo "Summary:"
      jq '{
        schema_field_count: (.schema_fields | length),
        parser_yaml_field_count: (.parser_yaml_fields | length),
        workflow_yaml_field_count: (.workflow_yaml_fields | length),
        gaps: {
          in_schema_not_parser: (.field_gaps.in_schema_not_parser | length),
          in_parser_not_schema: (.field_gaps.in_parser_not_schema | length),
          in_schema_not_workflow: (.field_gaps.in_schema_not_workflow | length),
          in_used_not_schema: (.field_gaps.in_used_not_schema | length)
        }
      }' /tmp/gh-aw/agent/schema-diff.json
source: github/gh-aw/.github/workflows/schema-consistency-checker.md@7f977f17bd6948b45209fab4719566b435f8ecc5
---

# Schema Consistency Checker

You are an expert system that detects inconsistencies between:
- Database migrations (`db/migrations/sqlite/*.sql` and `db/migrations/postgres/*.sql`)
- The Go database layer (`internal/db/*.go`) and handler DTOs (`internal/handlers/*.go`)
- The TypeScript frontend types (`frontend/src/types.ts`) and API client (`frontend/src/lib/api.ts`)
- The workflows in the project (`.github/workflows/*.md`)
- The API routes (`internal/server/routes.go`)

## Mission

Analyze the Biblioteka repository to find inconsistencies across these key areas and create a discussion report with actionable findings.

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

**Early exit**: If **none** of the five layers have any changed files, call the `noop` safe-output tool and stop immediately. Do not proceed to analysis.

```json
{"noop": {"message": "✅ No schema-related files changed in the last 24 hours. Schema Consistency Checker has nothing to analyze today."}}
```

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

<<<<<<< current (local changes)
### Step 0: Detect Changed Schema Files (Scope Detection)

Run the scope detection from the [Scope Detection](#scope-detection) section above:
1. Get the list of files changed in the last 24 hours
2. Map them to the five schema layer flags
3. Exit gracefully if no schema layers changed
4. Record which layers are active for this run — only active layers are analyzed in Steps 3–4

||||||| base (original)
=======
### Step 0: Read Pre-Computed Data (Start Here)

Before doing anything else, read the schema diff that was computed before your session began:

```bash
cat /tmp/gh-aw/agent/schema-diff.json
```

This file contains:
- `schema_fields`: All top-level field names in the main JSON schema
- `parser_yaml_fields`: All yaml-tagged struct fields in `pkg/parser/*.go`
- `workflow_yaml_fields`: All yaml-tagged struct fields in `pkg/workflow/*.go`
- `used_in_workflows`: All top-level frontmatter keys used in `.github/workflows/*.md`
- `field_types`: Schema field types for all top-level fields
- `field_gaps.in_schema_not_parser`: Fields in schema absent from parser yaml tags
- `field_gaps.in_parser_not_schema`: Fields as parser yaml tags absent from schema
- `field_gaps.in_schema_not_workflow`: Fields in schema absent from workflow compiler yaml tags
- `field_gaps.in_used_not_schema`: Fields used in workflow files but not in schema

**Use this pre-computed data as your primary starting point.** Do NOT re-run the field enumeration commands from scratch — instead, refine and supplement the pre-computed data with targeted follow-up queries (e.g., checking a specific file for a specific field).

>>>>>>> new (upstream)
### Step 1: Load Previous Strategies
```bash
# Check if strategies file exists
if [ -f /tmp/gh-aw/cache-memory/strategies.json ]; then
  cat /tmp/gh-aw/cache-memory/strategies.json
fi
```

### Step 2: Choose Analysis Focus

<<<<<<< current (local changes)
### Step 3: Execute Analysis
||||||| base (original)
### Step 3: Execute Analysis
Use chosen strategy to find inconsistencies. Examples:
=======
Using the pre-computed `field_gaps` from Step 0 plus the strategy cache from Step 1:
- If `field_gaps` show promising leads, start there (they are likely high-signal)
- If cache has strategies, use a proven strategy 70% of the time; try a new approach 30% of the time
>>>>>>> new (upstream)

<<<<<<< current (local changes)
**Only analyze the layers flagged as active in Step 0.** Skip any analysis area whose layer flag is `false`.

Use the chosen strategy to find inconsistencies within the active layers. Example for DTO field enumeration (run only when `LAYER_HANDLERS` or `LAYER_FRONTEND` is active):

||||||| base (original)
**Example: Field enumeration**
=======
>>>>>>> new (upstream)
```bash
<<<<<<< current (local changes)
# Step 1: List all Go DTO types and their json fields in handlers
echo "=== Go Handler DTOs ==="
grep -rn 'json:"' internal/handlers/*.go | grep -v '_test.go' | grep 'type\|json:' | head -60
||||||| base (original)
# Extract schema fields using jq for robust JSON parsing
jq -r '.properties | keys[]' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null | sort -u
=======
# Determine selection mode (0-6 = proven strategy, 7-9 = new approach)
day_mod=$(( $(date +%j) % 10 ))
if [ "$day_mod" -le 6 ]; then
  echo "Use proven strategy from cache"
else
  echo "Try new approach"
fi
```
>>>>>>> new (upstream)

<<<<<<< current (local changes)
# Step 2: List TypeScript types
echo "=== TypeScript Types ==="
cat frontend/src/types.ts
||||||| base (original)
# Extract parser fields from pkg/parser (look for yaml tags)
grep -r "yaml:\"" pkg/parser/*.go | grep -o 'yaml:"[^"]*"' | sort -u
=======
### Step 3: Execute Targeted Analysis
>>>>>>> new (upstream)

<<<<<<< current (local changes)
# Step 3: Find DB entity structs
echo "=== DB Entity Structs ==="
grep -A 30 'type Book struct' internal/db/books.go
||||||| base (original)
# Extract workflow compiler fields from pkg/workflow (look for yaml tags and frontmatter access)
grep -r "yaml:\"" pkg/workflow/*.go | grep -o 'yaml:"[^"]*"' | sort -u
grep -r 'frontmatter\["[^"]*"\]' pkg/workflow/*.go | grep -o '\["[^"]*"\]' | sort -u
=======
Use the pre-computed data as context and run **targeted** follow-up commands only when
deeper inspection is needed (e.g., checking how a specific field is actually processed in code).
>>>>>>> new (upstream)

<<<<<<< current (local changes)
# Step 4: Find toBookDTO mapping
echo "=== toBookDTO mapping ==="
grep -A 30 'func toBookDTO' internal/handlers/books.go
||||||| base (original)
# Extract documented fields
grep -r "^###\? " docs/src/content/docs/reference/frontmatter.md
=======
**Example: Verify a gap from pre-computed data**
```bash
# Verify a specific field gap by searching implementation files
grep -r "fieldName" pkg/parser/ pkg/workflow/ 2>/dev/null | grep -v "_test.go"
>>>>>>> new (upstream)
```

<<<<<<< current (local changes)
When `LAYER_MIGRATIONS` or `LAYER_DB` is active, also run migration drift checks. When `LAYER_ROUTES` is active, also run route coverage analysis. Skip commands for inactive layers to keep the run fast.
||||||| base (original)
**Example: Type checking**
```bash
# Find schema field types (handles different JSON Schema patterns)
jq -r '
  (.properties // {}) | to_entries[] |
  "\(.key): \(.value.type // .value.oneOf // .value.anyOf // .value.allOf // "complex")"
' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null || echo "Failed to parse schema"
```
=======
**Example: Type checking for a specific field**
```bash
# Find schema field types (handles different JSON Schema patterns)
jq -r '
  (.properties // {}) | to_entries[] |
  "\(.key): \(.value.type // .value.oneOf // .value.anyOf // .value.allOf // "complex")"
' pkg/parser/schemas/main_workflow_schema.json 2>/dev/null || echo "Failed to parse schema"
```
>>>>>>> new (upstream)

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

### Step 5b: Cross-Agent Awareness Check

Before creating the discussion report, query for any **open** pull requests from sibling documentation automation agents. This provides maintainers with full context: inconsistencies reported here may already be addressed (or in progress) via a concurrent documentation PR.

Search for open sibling agent PRs:
```
repo:${{ github.repository }} is:pr is:open label:documentation label:automation
```

For each open PR found, note:
- The PR number and title
- Which documentation files it modifies (check the PR body for filenames like `docs/api-reference.md`)
- Whether any of this checker's findings are likely covered by that PR

Include the results as a dedicated **"Open Sibling Agent PRs"** section in the discussion report (see report format below). If no open sibling PRs exist, record `None`.

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

## Open Sibling Agent PRs

[List any open PRs from sibling automation agents (update-docs, daily-doc-updater) with label:documentation label:automation. For each PR, note the PR number, title, and which docs files it touches. Mention if any of this report's findings appear to be already addressed by a sibling PR. If none: "None."]

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
<<<<<<< current (local changes)
- **Run scope detection first** (Step 0) — exit early when no schema files changed
- Analyze only active layers; skip inactive layers entirely
- Use bash tools efficiently (grep, find, etc.)
||||||| base (original)
- Use bash tools efficiently (grep, jq, etc.)
=======
- **Always start from `/tmp/gh-aw/agent/schema-diff.json`** — this pre-computed diff eliminates the need to re-read all source files
- Use targeted bash commands to verify specific leads from the pre-computed data
>>>>>>> new (upstream)
- Cache results when re-analyzing same data
- Don't re-check things found in previous runs (check cache first)
<<<<<<< current (local changes)
- Focus on high-impact areas (books, authors, series are core entities)
||||||| base (original)
- Focus on high-impact areas
=======
- Focus on high-impact areas (field gaps with parser mismatches are usually most critical)
>>>>>>> new (upstream)

### Strategy Evolution
- Try genuinely different approaches when not using cached strategies
- Document why a strategy worked or failed
- Update success metrics in cache
- Consider combining successful strategies

## Tools Available

You have access to:
- **bash**: Any command (use grep, jq, find, cat, etc.)
- **edit**: Create/modify files in cache memory
- **github**: Read repository data, discussions

## Success Criteria

A successful run:
- ✅ Runs scope detection (Step 0) before any analysis
- ✅ Calls `noop` and exits gracefully when no schema-related files changed in the last 24 hours
- ✅ Analyzes only the layers affected by recent changes (skips unchanged layers)
- ✅ Uses or creates an effective detection strategy
- ✅ Updates cache with strategy results
- ✅ Finds at least one category of inconsistencies OR confirms consistency
- ✅ Creates a detailed discussion report with actionable recommendations, OR calls `noop` if no inconsistencies are found

**Important**: You **MUST** call exactly one terminal safe-output (`noop` or `create-discussion`) before finishing. Ancillary calls like `upload-asset` or `close-discussion` are allowed alongside the terminal output. If no schema-related files changed in the last 24 hours, or if analysis finds zero inconsistencies, call `noop` with a descriptive status message. Otherwise, create a discussion report.

<<<<<<< current (local changes)
Example noop output:

```json
{"noop": {"message": "✅ Schema consistency check complete: no inconsistencies found across analyzed layers."}}
```

Begin your analysis now. Check the cache, choose a strategy, execute it, and either create a discussion report with your findings or call `noop` with a descriptive status message when there are no relevant changes or no inconsistencies.
||||||| base (original)
**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
=======
{{#import shared/noop-reminder.md}}
>>>>>>> new (upstream)
