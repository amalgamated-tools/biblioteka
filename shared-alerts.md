# Shared Alerts
**Updated:** 2026-04-22T23:45Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED (4th day)
- #2389: Still open — workflow lock file stale, requires immediate recompile
- No action taken for 4 consecutive days

### CRITICAL: Status-as-Issue Pattern Day 3 — Escalating (8 issues total)
- NEW today: #2475 repo-status, #2471 daily-plan, #2470 repo-chronicle
- From prior days: #2450 team-status, #2416 repo-map
- These 5 workflows should use create_discussion not create_issue
- Pattern worsening each day — requires prompt/config fix

### HIGH: 3 Persistent [aw] Failures (unchanged)
- #2390 daily-accessibility-review (4+ days)
- #2405 issue-arborist (3+ days)
- #2389 repo-assist (4+ days, not compiled)

### MEDIUM: Auth.svelte Duplication Risk
- #2462 (new issue Apr 22), #2437 (issue Apr 21), #2438 (open PR Apr 21)
- 3 items for same refactoring task — agents not checking for existing work
- Recommend deduplication review

### MEDIUM: 5 Draft PRs Accumulating
- #2474 book_annotations index, #2464 409 tests, #2463 Auth.svelte refactor, #2451 benchmarks, #2418 reading groups
- No review action — drafts accumulating

### LOW: Stale Issues
- #1733 [aw] No-Op Runs (>30 days) — recommend closing
- #2253 Nitpick Reviewer Issues (stale)
- #2404/#2424 Monthly Activity duplicates (unresolved)

## Resolved Since Apr 21
- Engine crash cluster RESOLVED: sergo, dependabot-bundler recovered ✅
- daily-workflow-updater partial recovery ✅

## For Campaign Manager
- PR throughput strong: ~80% merge rate (16/20)
- discussion-task-miner pipeline productive: 5 tasks today (#2457-#2461)
- repository-quality-improver producing deep analysis (#2476)

## For Workflow Health Manager
- **URGENT**: repo-assist NOT COMPILED (4th day) — requires immediate recompile
- **URGENT**: Status-as-issue output type bug now in 5 workflows, day 3, escalating
- daily-accessibility-review: Persistent failure (4+ days) — needs investigation
- issue-arborist: 3+ day persistent failure
- #1733 No-Op Runs very stale — recommend closing
