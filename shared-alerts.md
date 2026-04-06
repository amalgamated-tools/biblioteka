# Shared Alerts — Cross-Orchestrator Coordination
**Last updated:** 2026-04-06T23:44Z by agent-performance-analyzer

## Active Alerts

### 🔴 CRITICAL: GH_AW_AGENT_TOKEN Permission Failure (NEW - Apr 6)
- **Issue**: `duplicate-code-detector` failed to assign Copilot to issues #1449 and #1451
- **Error**: ERR_PERMISSION: copilot coding agent is not available for this repository
- **Tracking**: Issue #1452 (open)
- **Impact**: All workflows using Copilot assignment via GH_AW_AGENT_TOKEN are affected
- **Action needed**: Verify GH_AW_AGENT_TOKEN has `issues: write` permission + active Copilot subscription
- **Raised by**: agent-performance-analyzer

### 🟠 HIGH: 6 Uncompiled Workflows
- **Workflows**: artifacts-summary, commit-changes-analyzer, duplicate-code-detector, grumpy-reviewer, pr-nitpick-reviewer, weekly-repo-map
- **Impact**: These workflows will not execute until compiled
- **Action needed**: Run agenticworkflows-compile on each
- **Raised by**: agent-performance-analyzer (Apr 4, still open)

### 🟡 MEDIUM: ci-coach Perpetual No-Op Loop (7+ consecutive runs)
- **Pattern**: Runs #31–#37+ — all no action needed
- **Context**: All CI optimization opportunities exhausted
- **Recommendation**: Reduce schedule from daily to weekly
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: daily-workflow-updater Permanent Failure
- **Issue**: gh CLI not installed/authenticated in agent environment
- **Impact**: Workflow reports noop but never executes; wasted compute
- **Recommendation**: Disable or fix environment; remove from daily schedule
- **Raised by**: agent-performance-analyzer

## Resolved Alerts
### ✅ RESOLVED: daily-doc-updater PR Volume Escalation (Apr 4-5)
- Was CRITICAL for 2 days; issue #1381 closed by veverkap on Apr 6 as "completed"
- Today output: 2 PRs (down from 18+ on Apr 5) — 89% reduction
- Resolved: 2026-04-06

### ✅ RESOLVED: daily-team-evolution-insights NOW COMPILED
- Was critical for 2 days (Apr 3, Apr 4); compiled successfully 2026-04-05
