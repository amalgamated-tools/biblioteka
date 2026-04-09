# Shared Alerts — Cross-Orchestrator Coordination
**Last updated:** 2026-04-09T23:44Z by agent-performance-analyzer

## Active Alerts

### ⚠️ CRITICAL: update-docs Duplicate PR Creation (Apr 8, Day 2)
- **Issue**: 3 identical PRs opened in ~2 hours (#1538, #1539, #1547)
- **Title**: "docs: document TimeoutState base class, ThemePreference type, and TRUSTED_PROXIES rate limiter"
- **Root cause**: TOCTOU race — multiple Test workflow completions trigger concurrent update-docs runs; dedup check passes before sibling opens its PR
- **Action needed**: Implement lock-based or post-create dedup check in update-docs workflow
- **Raised by**: agent-performance-analyzer (Apr 8); escalated Apr 9

### ⚠️ MEDIUM: GH_AW_AGENT_TOKEN Permission Failure (from Apr 6, Day 4)
- **Issue**: Workflows fail to assign Copilot to issues
- **Error**: ERR_PERMISSION: copilot coding agent is not available
- **Tracking**: Issues #1452 (old), #1551 (Apr 8)
- **Impact**: Auto-fix pipeline degraded; issues sit unassigned
- **Action needed**: Verify GH_AW_AGENT_TOKEN has Copilot assignment scope + org plan check
- **Raised by**: agent-performance-analyzer

### ⚠️ PENDING MERGE: merge-conflict-resolver Config Bug (Apr 8)
- **Issue**: push_to_pull_request_branch fails when triggered via workflow_dispatch
- **Tracking**: Issue #1541 (open), Fix PR #1544 (open, Copilot-authored)
- **Action needed**: Review and merge PR #1544
- **Raised by**: agent-performance-analyzer

### ⚠️ PENDING MERGE: daily-malicious-code-scan Compilation (Apr 8)
- **Issue**: Compiled: No despite successful runs
- **Fix PR #1545**: Open, Copilot-authored, awaiting merge
- **Raised by**: agent-performance-analyzer

### 🟡 LOW: daily-workflow-updater Phantom Failure (Apr 8)
- **Issue**: Run 24134169785 shows success but created issue #1526
- **Status**: Assigned to Copilot; expires Apr 15
- **Raised by**: agent-performance-analyzer

### 🟡 NEW: daily-repo-chronicle Zero Runs (Apr 9)
- **Issue**: No runs observed Apr 3-4; scheduled daily but appears inactive
- **Status**: Under observation; no issue created yet
- **Action needed**: Check if workflow is disabled, check triggers
- **Raised by**: agent-performance-analyzer

## Resolved Alerts

### ✅ RESOLVED: All Workflows Now Compiled (Apr 7)
- Was: 🟠 6 uncompiled workflows (Apr 3-7)
- Now: 32/33 compiled (1 fix PR still open)
- Resolved: 2026-04-07

### ✅ RESOLVED: daily-doc-updater PR Volume Escalation (Apr 4-6)
- Was CRITICAL; issue #1381 closed Apr 6
- Daily doc updater now creates 1 targeted PR
- Resolved: 2026-04-06

### ✅ RESOLVED: Copilot PR Queue Backlog (Apr 7)
- Was: 14 open Copilot PRs
- Now: ~6 open PRs (significant reduction)
- Resolved: 2026-04-08
