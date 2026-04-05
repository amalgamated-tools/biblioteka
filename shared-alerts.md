# Shared Alerts — Cross-Orchestrator Coordination
**Last updated:** 2026-04-05T02:15Z by agent-performance-analyzer

## Active Alerts

### 🔴 CRITICAL: daily-team-evolution-insights NOT COMPILED (Day 2)
- **Status**: Persistent — was flagged 2026-04-04, still unresolved 2026-04-05
- **Impact**: Workflow will not execute at all until compiled
- **Action needed**: Run `agenticworkflows-compile` on this workflow
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: daily-doc-updater PR Volume Spike
- **Pattern**: 0 (Mar 31) → 8 PRs (Apr 3) → 16 PRs (Apr 4, doubled)
- **Risk**: PR fatigue for human reviewers; potential over-creation
- **Monitor**: If Apr 5 exceeds 16 PRs, escalate to critical
- **Raised by**: agent-performance-analyzer

### 🟡 MEDIUM: ci-coach Perpetual No-Op Loop
- **Pattern**: Runs #31, #32, #33, #34 — all no action needed (4 consecutive)
- **Context**: All 5 prior optimization PRs (#802, #820, #915, #981, #1007) fully applied
- **Recommendation**: Reduce schedule from daily to weekly
- **Raised by**: agent-performance-analyzer

### 🟢 LOW: issue-triage Double-Fire on Same Issue
- **Instance**: Issue #1248 triaged twice in ~30 min on Apr 4
- **Cause**: Likely edited-issue event firing multiple times
- **Impact**: Minimal (noop both times, no duplicates created)
- **Watch**: Look for pattern recurrence
- **Raised by**: agent-performance-analyzer

## Resolved Alerts
- (none yet — all active alerts from today)
