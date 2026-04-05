# Shared Alerts — Cross-Orchestrator Coordination
**Last updated:** 2026-04-05T23:44Z by agent-performance-analyzer

## Active Alerts

### 🔴 CRITICAL: daily-doc-updater PR Volume Escalation + Duplicates (Day 3)
- **Pattern**: 0 (Mar 31) → 8 (Apr 3) → 16 (Apr 4) → 18+ (Apr 5, +3 confirmed duplicates)
- **Duplicates today**: #1376/#1378 (same title), #1308/#1352 (icon aria-hidden), #1308/#1311 (api.test.ts)
- **Action needed**: Add deduplication check + cap PRs per run (suggest max 8)
- **Raised by**: agent-performance-analyzer (escalated from Medium Apr 4)

### 🟠 HIGH: 6 Uncompiled Workflows
- **Workflows**: artifacts-summary, commit-changes-analyzer, duplicate-code-detector, grumpy-reviewer, pr-nitpick-reviewer, weekly-repo-map
- **Impact**: These workflows will not execute until compiled
- **Action needed**: Run agenticworkflows-compile on each
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: ci-coach Perpetual No-Op Loop (6 consecutive runs)
- **Pattern**: Runs #31–#36 — all no action needed
- **Context**: All CI optimization opportunities exhausted
- **Recommendation**: Reduce schedule from daily to weekly
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: daily-workflow-updater Permanent Failure
- **Issue**: gh CLI not installed/authenticated in agent environment
- **Impact**: Workflow reports noop but never actually runs; wasted compute
- **Recommendation**: Disable or fix environment; remove from daily schedule
- **Raised by**: agent-performance-analyzer

## Resolved Alerts
### ✅ RESOLVED: daily-team-evolution-insights NOW COMPILED
- Was critical for 2 days (Apr 3, Apr 4); compiled successfully 2026-04-05
- Resolved by: agent-performance-analyzer (compiled during this run)
