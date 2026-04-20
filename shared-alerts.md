# Shared Alerts
**Updated:** 2026-04-20T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED
- #2389: [aw] Repo Assist failed — workflow not compiled (lock file stale)
- Immediate action required: recompile repo-assist workflow

### HIGH: [aw] Failure Cluster — 5 active (↑ from 3 Apr 19)
- NEW: **issue-arborist** (#2405): Failed today
- NEW: **repo-assist** (#2389): Failed today + NOT COMPILED
- PERSISTENT: **daily-accessibility-review** (#2390): Still failing
- STALE: **Detection Runs** (#2328): Persistent
- VERY STALE: **No-Op Runs** (#1733): Candidate for closure (>30 days stale)

### HIGH: Status Content as Issues (5 agents, ↑ from 1 yesterday)
- #2428 repo-status, #2420 team-status, #2419 daily-plan, #2417 repo-chronicle, #2416 repo-map
- Pattern spreading: 1 agent yesterday → 5 today
- These agents should use create_discussion, not create_issue

### MEDIUM: 3 Draft PRs Awaiting Review
- #2418 docs(reading-groups + AI enrichment) — DRAFT
- #2423 perf: composite sort indexes — DRAFT
- #2425 docs(unbloat): administration cleanup — DRAFT

### MEDIUM: Monthly Activity Duplication
- #2424 "perf: Monthly Activity 2026-04" + #2404 "test: Monthly Activity 2026-04"
- Two agents both creating monthly activity issues for same period

### LOW: Nitpick Reviewer Issues (#2253)
- Long-running open issue, may be stale

## Resolved Since Apr 19
- daily-doc-updater duplicate PR pattern: 3 dupes Apr 19 → 0 today ✅
- PR backlog cleared: ~19 open → 6 open ✅
- veverkananobot issues resolved via Copilot PRs ✅
- PR merge rate improved: 87% → 90% ✅

## For Campaign Manager
- PR throughput: 18/20 recent merged (90% rate) — strong execution
- Open feature PRs: 3 drafts need review attention
- Task miner pipeline healthy: 5 new actionable issues created

## For Workflow Health Manager
- repo-assist: NOT COMPILED — requires immediate recompile
- issue-arborist: New failure — check logs
- daily-accessibility-review: Persistent failure — needs investigation
- Status-as-issue problem spreading across 5 agents — needs output type audit
- #1733 No-Op Runs very stale — recommend closing
