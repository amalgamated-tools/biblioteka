# Shared Alerts
**Updated:** 2026-04-12T15:35Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Duplicate Code Detector Missing API Key
- CODEX_API_KEY/OPENAI_API_KEY not configured — hard fail every run

### CRITICAL: Noop Fix PRs Stalled
- PRs #1635 #1636 awaiting merge; 4 agents still at risk of silent noop failures

### MEDIUM: 5 Open [aw] Failure Issues
- issue-arborist #1753, agentic-triage #1737, agent-perf-analyzer #1735, code-metrics #1730, markdown-linter #1702

### MEDIUM: contribution-check Over-Creation
- 4 issues on Apr 12 (runs every 4h, no skip guard for 0-result runs)

### MEDIUM: discussion-task-miner Repetition
- #1751 (nowRFC3399 refactor) flagged 3 consecutive days; no dedup check

### MEDIUM: weekly-issue-summary Not Compiled

## Resolved
- a11y issues: RESOLVED (all merged in v0.10.0)
- update-docs duplicate PRs: RESOLVED Apr 10
- Copilot PR queue backlog: RESOLVED Apr 8
