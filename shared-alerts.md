# Shared Alerts
**Updated:** 2026-04-12T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Duplicate Code Detector Missing API Key
- CODEX_API_KEY/OPENAI_API_KEY not configured — hard fail every run (36 failures total)
- Run ID: 24317892622 (latest failure)
- Action needed: Configure CODEX_API_KEY secret in repository settings

### MEDIUM: 5 Open [aw] Failure Issues
- issue-arborist #1753, agentic-triage #1737, agent-perf-analyzer #1735, code-metrics #1730, markdown-linter #1702
- All appear to be noop/safe-output related

### MEDIUM: contribution-check Over-Creation
- Running 4x/day, creating report issues even with 0 or 'lgtm' results
- Issue #1807 created today with 'lgtm' label — low signal value

### MEDIUM: unbloat-docs Low Merge Rate
- PR #1808 open, 0% merge rate for recent PRs (consolidation PRs need review)

### LOW: discussion-task-miner Dedup
- Previously flagged repeat issue, monitor ongoing

## Resolved
- a11y issues: RESOLVED (all merged in v0.10.0)
- update-docs duplicate PRs: RESOLVED Apr 10
- Copilot PR queue backlog: RESOLVED Apr 8
- v0.10.0 shipped Apr 12 with features from agentic discussions
