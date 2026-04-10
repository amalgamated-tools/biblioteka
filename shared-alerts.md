# Shared Alerts
**Updated:** 2026-04-10T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Noop Omission Epidemic (Apr 10)
- 4 agents: daily-nitpick-reviewer, dead-code-remover, daily-file-diet, code-simplifier
- All ran successfully but omitted noop call; generated failure issues #1629 #1615 #1614 #1613
- Fix PRs #1635 #1636 awaiting merge; need to extend fix to all 4 agents

### CRITICAL: Duplicate Code Detector Missing API Key (Apr 10)
- CODEX_API_KEY / OPENAI_API_KEY secret not configured
- Run 24267037507 — hard fail every day until secret added

### MEDIUM: Daily Accessibility Review Crash (Apr 10)
- Run 51 failed; orphan process termination; was OK Apr 6-9 (runs 47-50)
- Investigate crash/OOM

### PENDING MERGE: daily-team-evolution-insights Fix PR #1635 (Apr 9)
### PENDING MERGE: code-simplifier Noop Fix PR #1636 (Apr 10)

## Resolved
- update-docs duplicate PR race condition: RESOLVED Apr 10
- All workflows compiled: RESOLVED Apr 7
- daily-doc-updater PR volume escalation: RESOLVED Apr 6
- Copilot PR queue backlog: RESOLVED Apr 8
