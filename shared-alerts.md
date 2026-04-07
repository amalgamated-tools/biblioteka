# Shared Alerts — Cross-Orchestrator Coordination
**Last updated:** 2026-04-07T23:44Z by agent-performance-analyzer

## Active Alerts

### ⚠️ NEW: Copilot PR Queue Backlog (Apr 7)
- **Issue**: 14 open Copilot PRs from today's agentic workflows
- **PRs**: #1483-#1494 (refactoring, auth, docs, CI)
- **Risk**: Review burden, merge conflicts if not addressed timely
- **Action needed**: Review and merge batch — no single blocker; quality appears high
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: GH_AW_AGENT_TOKEN Permission Failure (from Apr 6)
- **Issue**: `duplicate-code-detector` failed to assign Copilot to issues
- **Error**: ERR_PERMISSION: copilot coding agent is not available
- **Tracking**: Issue #1452 (open)
- **Impact**: Workflows using Copilot assignment may be silently skipped
- **Action needed**: Verify GH_AW_AGENT_TOKEN has Copilot assignment permission
- **Raised by**: agent-performance-analyzer (Apr 6, still open)

### 🟡 LOW: daily-workflow-updater Monitoring
- **Issue**: Previously had persistent gh CLI failure; today ran 4.2 min
- **Status**: Possible improvement — needs 3 more successful runs to clear
- **Raised by**: agent-performance-analyzer

## Resolved Alerts

### ✅ RESOLVED: All Workflows Now Compiled (Apr 7)
- Was: 🟠 6 uncompiled workflows (Apr 3-7)
- Now: All 31 workflows compiled: Yes
- Resolved: 2026-04-07

### ✅ RESOLVED: daily-doc-updater PR Volume Escalation (Apr 4-6)
- Was CRITICAL for 2 days; issue #1381 closed Apr 6
- Today: 1 PR (sustained fix confirmed)
- Resolved: 2026-04-06

### ✅ RESOLVED: daily-team-evolution-insights NOW COMPILED
- Compiled successfully 2026-04-05
