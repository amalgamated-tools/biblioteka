# Shared Alerts
**Updated:** 2026-04-27T23:46Z by agent-performance-analyzer

## Active Alerts

### CRITICAL: repo-assist NOT COMPILED + FAILED (Day 9)
- #2600: Another failure issue today — workflow lock file stale, requires recompile
- No action for 9 consecutive days
- **Escalation**: Longest-running critical defect in the ecosystem

### CRITICAL: Status-as-Issue Pattern Day 8 — NOW 5 WORKFLOWS
- daily-repo-status: #2618 (today)
- daily-team-status: #2613 (today)
- daily-plan: #2610 (today)
- daily-repo-chronicle: #2607 (today) + [aw] failure #2494
- weekly-repo-map: #2606 (today — NEWLY AFFECTED, Day 1)
- Pattern is SPREADING to more workflows without any fix applied

### CRITICAL: code-simplifier Groups.go — Day 5 FIVE-WAY DUPLICATION
- Issues: #2486 (Apr 23), #2520 (Apr 24), #2593 (Apr 27 — NEW TODAY)
- PRs: #2487 (Apr 23), #2521 (Apr 24), #2594 (Apr 27 — NEW TODAY WIP)
- URGENT: Add skip-if-match: `is:issue is:open groups.go` guard immediately

### HIGH: SSRF 3-Way Duplication (Day 4 — UNCHANGED)
- Issues: #2503, #2557, #2579
- PRs: #2504, #2558, #2580 — all still open with 3 competing approaches
- Requires dedup guard: `is:pr is:open SSRF` and immediate closure of 2 competing PRs

### HIGH: Reading Groups Docs — 4 Competing PRs (Day 6)
- #2418 (Apr 20), #2480 (Apr 23), #2481 (Apr 23), #2611 (Apr 27 — NEW TODAY)
- daily-doc-updater added FOURTH reading-groups PR today
- URGENT: Add skip-if-match: `is:pr is:open "reading-groups"` immediately

### HIGH: 10 Active [aw] Failures (3 new today)
- #2615: contribution-guidelines-checker (NEW Apr 27)
- #2614: agentic-triage (NEW Apr 27)
- #2600: repo-assist (Apr 27)
- #2583: daily-accessibility-review (NEW Apr 27)
- #2494: daily-repo-chronicle (Apr 23)
- #2442: dependabot-bundler (Apr 21)
- #2441: sergo (Apr 21 — may be stale)
- #2405: issue-arborist (Apr 20)
- #2390: daily-accessibility-review (Apr 20 — may be stale now #2583 is new)
- #2389: old repo-assist failure

### MEDIUM: Dependabot Bundle Issues Stacking (7 open)
- #2602 (Apr 27 NEW), #2570, #2549, #2524, #2490, #2467, #2411
- dependabot-pr-bundler should close previous bundle issues on new creation

### MEDIUM: Auth.svelte Stale Items (Day 6)
- #2462 (issue), #2463 (PR): still open despite #2438 merged Apr 22

## Resolved Since Apr 26
- Nothing resolved today

## For Campaign Manager
- 21 open PRs (stable)
- discussion-task-miner: 5 quality tasks today (#2591-#2587)
- SSRF: 3 competing PRs (#2504, #2558, #2580) — two should be closed immediately
- Groups.go: NOW 5-way dup — needs human decision + skip-if-match guard

## For Workflow Health Manager
- URGENT: repo-assist NOT COMPILED (Day 9) — critical escalation
- URGENT: Status-as-issue bug Day 8, now 5 workflows — spreading WITHOUT fix
- URGENT: code-simplifier: Groups.go Day 5, 5-way dup — must add dedup guard NOW
- URGENT: daily-doc-updater: 4th reading-groups PR — add skip-if-match NOW
- URGENT: 3 new [aw] failures today (contribution-guidelines-checker, agentic-triage, daily-accessibility-review)
- MEDIUM: dependabot-pr-bundler: 7 stacking bundle issues — fix close-previous logic
