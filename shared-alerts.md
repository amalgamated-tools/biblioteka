# Shared Alerts
**Updated:** 2026-04-16T23:45Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Engine Failure Cluster (4 workflows)
- **ci-doctor** (#2059): Persistent since Apr 15. Mid-grep termination. Scope too wide.
- **daily-test-improver** (#2089): New Apr 16. Engine died during `go fmt`. Over-scoped.
- **daily-grumpy-reviewer** (#2095): New Apr 16. Engine died writing JSON cache. High context.
- **update-docs** (#2097): New Apr 16. Engine died reading docs/api.md. High context.
- **Root cause**: Likely token exhaustion in large-context workflows. Fix: reduce per-run scope.

### HIGH: Status Issue Accumulation (Persistent)
- 6 open ephemeral status issues: repo-status #2111, team-status #2104, daily-plan #2103,
  repo-chronicle #2100, repo-assist monthly #2077, perf-improver monthly #2052
- Fix: Switch to discussions or auto-close predecessor before creating new

### MEDIUM: [aw] Tracking Issues Stale
- #2044 (detection runs) and #1733 (no-op runs) unresolved since Apr 12–15
- These indicate systemic workflow behavior issues not yet addressed

### LOW: Inactive Workflow Question
- 54 registered workflows; April metrics showed only 13/24 "active"
- Many workflows show "Unknown" status — GitHub API limitation or true inactivity?

## Resolved Since Apr 15
- contribution-check recurring failure: Closed via PR #2039 (false-positive fix) ✅
- PR backlog managed: 26 merged today ✅
- daily-doc-updater triple-dup: No recurrence ✅
- dependabot API block: bundler ran successfully today ✅

## For Campaign Manager
- v0.14.0 shipped Apr 16 (goauth migration, perf improvements, a11y fixes)
- Strong PR velocity: 26 merged Apr 16 (strongest day this week)
- Open features: #1971 (multi-tenant), #1832 (browser ext), #1531 (S3)
- task-miner → PR chain highly effective: drives #2115 (db refactor), #2109 (otelkeys fix)

## For Workflow Health Manager
- Engine termination cluster needs urgent investigation (4 affected workflows)
- All likely share root cause: large-context scan → token exhaustion before safe-output
- Recommend: add per-run file limits and partial-success logic to scan workflows
