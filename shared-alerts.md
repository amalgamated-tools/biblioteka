# Shared Alerts
**Updated:** 2026-04-13T23:45Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Duplicate Code Detector Missing API Key
- CODEX_API_KEY/OPENAI_API_KEY not configured — hard fail every run (36+ failures)
- Action needed: Configure CODEX_API_KEY secret in repository settings
- Impact: Zero output from only non-Copilot engine in ecosystem

### MEDIUM: contribution-check Over-Creation (Persists)
- Running 4x/day, creating report issues for "lgtm" / zero-finding scenarios
- Issue #1875 created 2026-04-13 with 0 actionable items
- Recommendation: Add skip condition when all checks pass with no findings

### MEDIUM: daily-doc-updater Duplicate PRs
- PRs #1865 and #1870 both address "docs(stats): fix clamping description" — near-duplicate
- 10 agent PRs currently open; review bandwidth may be bottleneck
- Recommendation: Add deduplication check before creating docs PRs

### MEDIUM: unbloat-docs Low Merge Rate
- PR #1878 open (docs/administration unbloat)
- Historical: 0% merge rate for recent batch
- Recommendation: Review PR quality or adjust scope of changes

### MEDIUM: PR Backlog Growing
- 10 open agent PRs (up from ~5 on Apr 12)
- If maintainer bandwidth is limited, some agents may need throttling
- Monitor: if backlog > 15 open agent PRs, recommend rate-limiting

### LOW: 5 Open [aw] Failure Issues (Stale)
- #1753, #1737, #1735, #1730, #1702 — open since Apr 12, no resolution
- These are noop/safe-output related failures from various workflows
- May need manual triage by maintainer

### LOW: daily-plan Issue Staleness
- #1804 Daily Plan from 2026-04-12 still open as of 2026-04-13
- Daily plan issues should be closed/resolved within 24h
- Recommendation: Add auto-close after 24h to daily-plan workflow

## Resolved
- a11y issues: RESOLVED (all merged in v0.10.0)
- update-docs duplicate PRs: RESOLVED Apr 10
- Copilot PR queue backlog: RESOLVED Apr 8
- v0.10.0 shipped Apr 12 with features from agentic discussions
- v0.11.0 release PR #1781 in progress
