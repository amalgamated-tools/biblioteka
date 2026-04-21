# Shared Alerts
**Updated:** 2026-04-21T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: Copilot Engine Crash Cluster — 3 NEW failures (Apr 21 AM UTC)
- #2440 daily-workflow-updater: crashed during gh-aw-actions update (v0.68.7→v0.69.0)
- #2441 sergo: crashed after writing cache (was completing successfully)
- #2442 dependabot-bundler: crashed during git commit
- **Pattern**: All "⚠️ Engine Failure: The copilot engine terminated unexpectedly"
- **Suspected**: Copilot API instability / context window limits on Apr 21 AM

### CRITICAL: repo-assist NOT COMPILED + FAILED (3rd day)
- #2389: Still open — workflow lock file stale, requires recompile

### HIGH: [aw] Failure Total = 8 Active (↑ from 5 Apr 20)
- NEW: #2440 daily-workflow-updater, #2441 sergo, #2442 dependabot-bundler (engine crashes)
- PERSISTENT: #2405 issue-arborist, #2390 daily-accessibility-review
- STALE: #2389 repo-assist, #2328 Detection Runs
- VERY STALE: #1733 No-Op Runs (>30 days — recommend closing)

### HIGH: Status-as-Issue Pattern Persists (Day 2, 5 agents)
- #2453 repo-status, #2450 team-status, #2447 daily-plan, #2446 repo-chronicle, #2416 repo-map
- These agents should use create_discussion not create_issue
- No improvement from yesterday

### MEDIUM: 4 Draft PRs Awaiting Review
- #2451 perf benchmarks, #2425 unbloat docs, #2423 composite sort indexes, #2418 reading groups
- Drafts accumulating without review action

### MEDIUM: Monthly Activity Duplication (2 issues, unresolved)
- #2424 "perf: Monthly Activity 2026-04" + #2404 "test: Monthly Activity 2026-04"

### LOW: Nitpick Reviewer Issues (#2253) — long-running stale

## Resolved Since Apr 20
- daily-doc-updater duplicate PR pattern: 0 duplicates for 2nd day ✅
- sergo partially recovered (created 3 tasks before crashing) — partial ✅

## For Campaign Manager
- PR throughput: 10 open (4 draft, 5 ready, 1 release)
- discussion-task-miner pipeline productive: 4 new actionable refactor tasks (#2435-#2434)
- Engine crash cluster may indicate API reliability issue for scheduling

## For Workflow Health Manager
- **URGENT**: 3 new engine crash failures (copilot terminated unexpectedly) — systemic pattern
- repo-assist: NOT COMPILED (3rd day) — requires immediate recompile
- daily-accessibility-review: Persistent failure (3+ days) — needs investigation
- issue-arborist: 2+ days failure
- Status-as-issue output type bug: 5 agents, day 2, no improvement
- #1733 No-Op Runs very stale — recommend closing
