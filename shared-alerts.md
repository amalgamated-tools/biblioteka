# Shared Alerts
**Updated:** 2026-04-23T23:45Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED (5th day)
- #2389: Still open — workflow lock file stale, requires immediate recompile
- No action taken for 5 consecutive days

### CRITICAL: Status-as-Issue Pattern Day 4 — Escalating (8+ issues total)
- NEW today: #2501 repo-status, #2498 team-status, #2495 daily-plan
- Prior days: #2470 repo-chronicle, #2416 repo-map, #2475, #2471 (earlier)
- These workflows should use create_discussion not create_issue
- Pattern worsening each day — requires prompt/config fix urgently

### HIGH: Triple-Duplicate Reading Groups Docs PRs
- #2418 (Apr 20, open), #2480 (Apr 23, open), #2481 DRAFT (Apr 23)
- update-docs and daily-doc-updater not checking for existing open PRs
- Add skip-if-match guard: is:pr is:open "document reading groups"
- Recommend closing #2480 and #2481 as duplicates

### HIGH: 6 Active [aw] Failures
- #2494 daily-repo-chronicle (NEW today, Apr 23)
- #2442 dependabot-bundler (Apr 21, 3 days)
- #2441 sergo (Apr 21, 3 days)
- #2440 daily-workflow-updater (Apr 21, 3 days)
- #2405 issue-arborist (Apr 20, 4 days)
- #2390 daily-accessibility-review (Apr 20, 4 days)

### MEDIUM: Auth.svelte Duplication Items Still Open
- #2462 (issue Apr 22), #2437 (issue Apr 21), #2463 (PR Apr 22)
- Three items for same refactoring task

### LOW: Stale Issues
- #1733 [aw] No-Op Runs (>30 days) — recommend closing
- #2416 repo-map (3 days, stale)
- #2424 Monthly Activity duplicate

## Resolved Since Apr 22
- No new resolutions today

## For Campaign Manager
- PR throughput: ~12 open PRs, ~80% historical merge rate
- discussion-task-miner still productive: 4 tasks today (#2482-#2485)
- duplicate-code-detector upgraded: now creates issue + draft PR together
- 5 draft PRs accumulating review debt

## For Workflow Health Manager
- URGENT: repo-assist NOT COMPILED (5th day) — requires recompile
- URGENT: Status-as-issue bug in 3+ workflows, Day 4, escalating
- URGENT: daily-repo-chronicle NEW failure today (#2494)
- daily-accessibility-review: 4+ day failure — needs investigation
- issue-arborist: 4+ day failure
- sergo, dependabot-bundler, daily-workflow-updater: 3+ day failures
