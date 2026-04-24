# Shared Alerts
**Updated:** 2026-04-24T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED (6th day)
- #2389: Still open — workflow lock file stale, requires immediate recompile
- No action taken for 6 consecutive days

### CRITICAL: Status-as-Issue Pattern Day 5 — Escalating (12+ issues total)
- NEW today: #2536 repo-status, #2530 team-status, #2529 daily-plan, #2528 chronicle
- Prior days: #2501, #2498, #2495 (Apr 23), #2470 (Apr 22), #2416 (earlier)
- daily-repo-chronicle "fixed" failure but now also creates issues instead of discussions
- Requires prompt/config fix for: daily-repo-status, daily-team-status, daily-plan, daily-repo-chronicle

### HIGH: Groups.go Refactoring 4-Way Duplication
- Issue #2486 (Apr 23): "split groups.go into focused sub-modules"
- Issue #2520 (Apr 24): "split groups.go into focused sub-files" (DUPLICATE)
- PR #2487 (Apr 23): "split group route handlers into focused modules"
- PR #2521 (Apr 24): "split groups.go into focused sub-files" (DUPLICATE)
- Need to close #2520 and #2521 as duplicates; code-simplifier lacks dedup check

### HIGH: Triple-Duplicate Reading Groups Docs PRs (Day 3+)
- #2418 (Apr 20, open), #2480 (Apr 23, open), #2481 (Apr 23, open)
- Add skip-if-match guard: is:pr is:open "reading groups"
- Recommend closing #2480 and #2481 as duplicates

### HIGH: 7 Active [aw] Failures (stable since Apr 23)
- #2494 daily-repo-chronicle (Apr 23)
- #2442 dependabot-bundler (Apr 21, 4 days)
- #2441 sergo (Apr 21, 4 days) — but sergo ran today (#2523)
- #2440 daily-workflow-updater (Apr 21, 4 days)
- #2405 issue-arborist (Apr 20, 5 days)
- #2390 daily-accessibility-review (Apr 20, 5 days)
- #2389 repo-assist (older, 6+ days)

### MEDIUM: Dependabot Bundle Issues Stacking (4 open)
- #2524 (Apr 24), #2490 (Apr 23), #2467, #2411
- dependabot-bundler should close previous bundle issues after creating a new one

### MEDIUM: Auth.svelte Duplication Items
- #2462 (issue Apr 22), #2463 (PR Apr 22) - but #2438 was MERGED, so these may be stale
- Check if #2462, #2463 are still valid after #2438 merge

## Resolved Since Apr 23
- No new [aw] failure issues today (good sign)
- daily-repo-chronicle: no new failure [aw] issue (run succeeded, though output type wrong)
- sergo appears to have run today (#2523)

## For Campaign Manager
- 25 open PRs, strong merge cadence (10 PRs merged Apr 23-24)
- duplicate-code-detector highly productive: issues + PRs together
- Groups.go refactoring blocked by 4-way duplication — human needs to decide which PR to use
- Auth.svelte split is DONE (#2438 merged) — related issues #2462, #2463 may be stale

## For Workflow Health Manager
- URGENT: repo-assist NOT COMPILED (6th day) — requires recompile
- URGENT: Status-as-issue bug in 4 workflows, Day 5, getting worse
- URGENT: daily-repo-chronicle still outputting wrong type (issue vs discussion)
- daily-accessibility-review: 5+ day failure
- issue-arborist: 5+ day failure
- sergo, dependabot-bundler, daily-workflow-updater: 4+ day failures
