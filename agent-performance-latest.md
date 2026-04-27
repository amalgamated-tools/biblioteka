# Agent Performance — 2026-04-27
**Run:** 2026-04-27T23:46Z

## Snapshot
- 55 workflows registered, all Copilot; **repo-assist NOT compiled (Day 9 ⚠️ CRITICAL)**
- ~40 open issues (21 created today)
- 21 open PRs (6 created today)
- [aw] Failures: **10 active** (+3 new today: contribution-guidelines-checker #2615, agentic-triage #2614, daily-accessibility-review #2583)
- Status-as-issue pattern: **Day 8** — #2618 (repo-status), #2613 (team-status), #2610 (daily-plan), #2607 (daily-repo-chronicle)
- **NEW**: weekly-repo-map creates status-as-issue (#2606) — 5th workflow with this pattern
- SSRF duplication: TRIPLE still open — #2503+#2557+#2579; PRs #2504+#2558+#2580
- Groups.go refactoring: NOW 5 ITEMS — issues #2486+#2520+#2593, PRs #2487+#2521+#2594 (Day 5)
- Reading groups docs: NOW 4 PRs — #2418, #2480, #2481, #2611 (daily-doc-updater added 4th today)
- Open dependabot bundles: **7 open** (#2602 added today)
- Auth.svelte stale: #2462/#2463 persist (Day 6)

## Top Performers
- **discussion-task-miner**: 5 tasks #2591-#2587. Score: 90/100 ✅
- **repository-quality-improver**: #2620 dialect consistency. Score: 80/100 ✅
- **daily-perf-improver**: #2616 reading list benchmarks. Score: 75/100 ✅
- **daily-test-improver**: #2592 kobo metadata tests. Score: 70/100 ✅
- **schema-consistency-checker**: clean run, no spurious output. Score: 75/100 ✅
- **glossary-maintainer**: clean run. Score: 75/100 ✅

## Agents Needing Improvement
- **repo-assist**: NOT COMPILED + FAILED (#2600). Score: 10/100 — CRITICAL (Day 9)
- **daily-repo-status**: Status-as-issue Day 8 (#2618). Score: 15/100
- **daily-team-status**: Status-as-issue Day 8 (#2613). Score: 15/100
- **daily-repo-chronicle**: Status-as-issue Day 8 (#2607) + [aw] #2494. Score: 15/100
- **daily-plan**: Status-as-issue Day 8 (#2610). Score: 20/100
- **weekly-repo-map**: New status-as-issue Day 1 (#2606). Score: 20/100 ⬆ NEWLY CRITICAL
- **code-simplifier**: Groups.go 5-way dup — Day 5 (#2593 issue, #2594 WIP PR). Score: 20/100 (↓ from 35)
- **contribution-guidelines-checker**: new failure (#2615). Score: 50/100
- **agentic-triage**: new failure (#2614). Score: 55/100
- **daily-accessibility-review**: failure (#2583). Score: 35/100
- **daily-doc-updater**: 4th reading-groups PR (#2611) today. Score: 40/100
- **dependabot-pr-bundler**: 7 stacking open bundle issues. Score: 30/100

## Regressions vs Yesterday (Apr 26)
- status-as-issue: weekly-repo-map NOW affected (5th workflow) ❌
- code-simplifier: Groups.go Day 5 — FIFTH ISSUE (#2593) + WIP PR (#2594) ❌
- daily-doc-updater: 4th reading-groups PR (#2611) ❌
- [aw] failures: 3 new today (total now 10, up from 7) ❌
- dependabot bundles: 7th open #2602 ❌

## Critical Issues
1. CRITICAL: repo-assist NOT COMPILED (Day 9) — #2600
2. CRITICAL: Status-as-Issue Pattern Day 8 — 5 workflows affected
3. CRITICAL: weekly-repo-map newly joins status-as-issue club (#2606)
4. HIGH: 10 active [aw] failures
5. HIGH: Groups.go 5-way duplication (Day 5) — #2486, #2520, #2593 + PRs #2487, #2521, #2594
6. HIGH: SSRF 3-way duplication — #2503+#2557+#2579; PRs #2504+#2558+#2580
7. HIGH: Reading groups 4 PRs — #2418, #2480, #2481, #2611
8. MEDIUM: 7 stacking dependabot bundle issues

## Discussion Created
"Agent Performance Report — 2026-04-27" in Audits category
