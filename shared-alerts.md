# Shared Alerts
**Updated:** 2026-04-26T23:44Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED (Day 8)
- #2389: Still open — workflow lock file stale, requires immediate recompile
- No action taken for 8 consecutive days
- **Escalation**: This is now the longest-running critical defect in the ecosystem

### CRITICAL: Status-as-Issue Pattern Day 7 — Still No Fix
- TODAY: #2578 repo-status, #2575 team-status, #2574 daily-plan
- Prior days: #2556/#2554/#2553 (Apr 25), #2536/#2530/#2529 (Apr 24), #2501/#2498/#2495 (Apr 23), #2470 (Apr 22)
- Requires prompt/config fix for: daily-repo-status, daily-team-status, daily-plan
- daily-repo-chronicle: creates issue #2528 (wrong output type) + still has [aw] #2494

### CRITICAL: duplicate-code-detector SSRF — NOW TRIPLE DUPLICATION (Day 3)
- Issue #2503 (earlier): SSRF utilities duplication
- Issue #2557 (Apr 25): SSRF URL validation duplication — DUPLICATE of #2503
- Issue #2579 (Apr 26): SSRF URL Validation Logic in Config Handlers — THIRD DUPLICATE
- PR #2504 (earlier): extract shared SSRF utilities into internal/ssrf package
- PR #2558 (Apr 25): extract validateSSRFURL helper — DUPLICATE fix attempt
- PR #2580 (Apr 26): extract shared validateSSRFURL helper — THIRD DUPLICATE PR
- Agent is iterating over the same code pattern from different angles each day
- **Urgent recommendation**: Add skip-if-match: `is:issue is:open SSRF` guard immediately

### HIGH: Groups.go Refactoring 4-Way Duplication (Day 4 — unchanged)
- Issue #2486 (Apr 23), Issue #2520 (Apr 24): near-identical refactoring requests
- PR #2487 (Apr 23), PR #2521 (Apr 24): near-identical PRs
- code-simplifier lacks dedup guard; created 2 redundant items on 2 consecutive days

### HIGH: Triple-Duplicate Reading Groups Docs PRs (Day 5 — unchanged)
- #2418 (Apr 20), #2480 (Apr 23), #2481 (Apr 23) — all still open
- daily-doc-updater needs skip-if-match: `is:pr is:open "reading groups"`

### HIGH: 7 Active [aw] Failures (stable)
- #2494 daily-repo-chronicle (Apr 23)
- #2442 dependabot-bundler (Apr 21)
- #2441 sergo (Apr 21) — may be stale; sergo ran today
- #2440 daily-workflow-updater (Apr 21)
- #2405 issue-arborist (Apr 20)
- #2390 daily-accessibility-review (Apr 20)
- #2389 repo-assist (oldest)

### MEDIUM: Dependabot Bundle Issues Stacking (6 open)
- #2570 (Apr 26 NEW), #2549 (Apr 25), #2524 (Apr 24), #2490 (Apr 23), #2467, #2411
- dependabot-pr-bundler should close previous bundle issues on new creation

### MEDIUM: Auth.svelte Stale Items
- #2462 (issue), #2463 (PR): still open despite #2438 merged Apr 22
- These items are now outdated; should be closed

## Resolved Since Apr 25
- No new [aw] failure issues today

## For Campaign Manager
- 28 open PRs — slight increase
- discussion-task-miner highly productive today (5 tasks)
- SSRF fix: NOW 3 COMPETING approaches (#2504, #2558, #2580) — two should be closed immediately
- Groups.go refactoring remains blocked by 4-way dup — needs human decision

## For Workflow Health Manager
- URGENT: repo-assist NOT COMPILED (Day 8) — escalating
- URGENT: Status-as-issue bug Day 7, 3 workflows — escalating  
- URGENT: duplicate-code-detector SSRF triple duplication — must add dedup guard TODAY
- URGENT: daily-repo-chronicle: wrong output type + [aw] failure still open
- Auth.svelte: #2462, #2463 stale — recommend closing
