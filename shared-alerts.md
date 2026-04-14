# Shared Alerts
**Updated:** 2026-04-14T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: daily-doc-updater Triple Duplicate PRs
- PRs #1994, #1980, #1976 identical: "docs(background-jobs): add scan:watch-folder"
- 3rd consecutive day with duplicate PRs — deduplication urgently needed
- Fix: Add `skip-if-match` or check for open PRs with same title before creating

### CRITICAL: Elevated Workflow Failure Rate (11% today)
- 6 [aw] failures Apr 14: contribution-check, daily-repo-chronicle, markdown-linter, issue-triage, update-docs, contribution-guidelines-checker
- Total open [aw] failures: 7 open issues (#1981, #1972, #1958, #1957, #1956, #1950, #1733)
- Investigate shared root cause (MCP server? auth?) across all failures

### HIGH: PR Backlog Surge
- 22 open agent PRs (9 bot + 13 Copilot) vs ~10 yesterday
- PR merge rate healthy (89%) but reviewer bandwidth may be strained
- Alert threshold: throttle agents if backlog exceeds 25

### MEDIUM: contribution-check Over-Creation
- Creates issues even on "lgtm" runs — #1947 is zero-finding report
- Fix: Add `skip-if-match` or no-findings skip condition

### MEDIUM: unbloat-docs Low Merge Rate
- PR #1967 open to improve scope criteria — wait for merge before next cycle

### LOW: Rapid Ecosystem Growth
- 24 → 54 workflows in 10 days (+125%) — quality controls may lag

## Resolved Since Apr 13
- A11y: 4 more issues resolved (#1904, #1902, #1900, #1898 merged in v0.13.0)
- duplicate-code-detector (Codex): no new failures — likely removed

## For Campaign Manager
- Task-miner → Copilot PR chain active: #1968, #1964, #1965 from task-miner issues
- Large feature PRs open: #1974 (AI enrichment), #1973 (recommendations), #1963 (groups)
- v0.13.0 release PR #1959 open
