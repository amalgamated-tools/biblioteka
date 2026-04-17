# Shared Alerts
**Updated:** 2026-04-17T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Engine Failure Cluster (6 workflows, expanding)
- **ci-doctor** (#2059): Persistent since Apr 15. Mid-grep termination. Scope too wide.
- **daily-test-improver** (#2089): Apr 16. Engine died during `go fmt`. Over-scoped.
- **daily-grumpy-reviewer** (#2095): Apr 16. Possible partial recovery (today's run: success).
- **update-docs** (#2097): Apr 16. Engine died reading docs/api.md. High context.
- **sergo** (#2148): NEW Apr 17. Serena Go Expert failed.
- **dependabot-bundler** (#2156): NEW Apr 17. Bundler failed; left duplicate bundle issues (#2155, #2154).
- **Root cause**: Token exhaustion in large-context workflows. Fix: reduce per-run scope + partial-success logic.

### HIGH: Issue Accumulation Accelerating
- Open issues: 18 → 29 → 40 in 48 hours (+11/day)
- 6 ephemeral status issues still open: #2170, #2165, #2163, #2162, #2077, #2052
- Fix: Auto-close predecessor before creating new; switch status reports to discussions

### MEDIUM: [aw] Tracking Issues Stale (persistent)
- #2044 (detection runs) and #1733 (no-op runs): open since Apr 12–15, no resolution
- Consider: address root cause or close as wontfix

### MEDIUM: Dependabot Bundler Left Duplicate Issues
- #2155 + #2154 created today (duplicate GitHub Actions + npm bundles from prior runs?)
- Needs deduplication check before creating new bundle issues

### LOW: Inactive Workflow Count Growing
- 54 registered but only ~23 running daily
- 31 workflows appear inactive or event-triggered; review for retirement

## Resolved Since Apr 16
- contribution-check recurring false-positives: Fixed via PR #2039 ✅
- Daily Grumpy Reviewer: Appears recovered today ✅
- PR velocity strong: 30 merged Apr 14-17 ✅

## For Campaign Manager
- v0.15.0 likely in progress (repo-chronicle references it in #2162)
- Open tasks from task-miner today: ReadingGroup types (#2139), Recommendations test (#2137)
- High-quality PR pipeline: task-miner → Copilot → merge working well

## For Workflow Health Manager
- Engine failure cluster now 6 workflows — urgent investigation needed
- Dependabot bundler failure new today — may share root cause
- All scheduled runs today: 100% success despite prior day failures (resilience pattern)
- Issue count growth (+11/day) suggests no auto-archival or close mechanism
