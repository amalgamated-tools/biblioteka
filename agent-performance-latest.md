# Agent Performance — 2026-04-26
**Run:** 2026-04-26T23:44Z

## Snapshot
- 55 workflows registered, all Copilot; **repo-assist NOT compiled (Day 8 ⚠️ CRITICAL)**
- 47 open issues (up from 45 two days ago, +2 today)
- 28 open PRs (up from 27 yesterday)
- [aw] Failures: **7 active** (stable, no new failures today)
- Status-as-issue pattern: **Day 7** — #2578 (repo-status), #2575 (team-status), #2574 (daily-plan)
- SSRF duplication: TRIPLE — #2503+#2557+#2579 (issues); #2504+#2558+#2580 (PRs)
- Groups.go refactoring duplication: 4 items — issues #2486+#2520, PRs #2487+#2521 (unchanged, Day 4)
- Reading groups docs triple-duplicate PRs: #2418, #2480, #2481 (Day 5+ — unchanged)
- Open dependabot bundles: **6 open** (#2570 added today)
- Auth.svelte stale: #2462 (issue) + #2463 (PR) still open despite #2438 merged Apr 22

## Top Performers
- **discussion-task-miner**: 5 actionable tasks #2562-#2566. Score: 85/100
- **sergo**: #2569 GA versions update PR. Score: 72/100
- **daily-test-improver**: #2567 DeleteConfirmation test. Score: 72/100
- **daily-perf-improver**: #2576 Kobo parallel, #2555 series index. Score: 70/100
- **unbloat-docs**: #2577 frontend docs cleanup PR. Score: 65/100

## Agents Needing Improvement
- **repo-assist**: NOT COMPILED + FAILED (#2389). Score: 10/100 — CRITICAL (Day 8)
- **daily-repo-status**: Status-as-issue Day 7 (#2578). Score: 15/100
- **daily-team-status**: Status-as-issue Day 7 (#2575). Score: 15/100
- **duplicate-code-detector**: SSRF triple duplication escalating. Score: 20/100 (↓ from 62)
- **daily-plan**: Status-as-issue Day 7 (#2574). Score: 20/100
- **daily-repo-chronicle**: Creates issue #2528 + [aw] failure #2494. Score: 25/100
- **code-simplifier**: Groups.go 4-way dup still open (Day 4). Score: 35/100
- **daily-doc-updater**: Triple reading groups PRs (#2418/#2480/#2481). Score: 38/100
- **dependabot-pr-bundler**: 6 stacking open bundle issues. Score: 40/100

## Improvements vs Yesterday (Apr 25)
- discussion-task-miner: 5 tasks today (up from 4) ✅
- daily-test-improver: #2567 new test PR ✅
- daily-perf-improver: #2576, #2555 perf improvements ✅
- unbloat-docs: #2577 frontend docs PR ✅

## Regressions vs Yesterday
- Status-as-issue pattern Day 7 (no fix applied) ❌
- repo-assist NOT COMPILED Day 8 ❌
- duplicate-code-detector: SSRF now TRIPLE (#2579 = 3rd issue, #2580 = 3rd PR) ❌
- Dependabot bundles: 6th open bundle #2570 added ❌
- Auth.svelte stale items (#2462, #2463) persist after #2438 merged ❌

## Critical Issues
1. CRITICAL: repo-assist NOT COMPILED (Day 8) — #2389
2. CRITICAL: Status-as-Issue Pattern Day 7 — #2578, #2575, #2574 (+ historical)
3. CRITICAL: SSRF 3-way duplication — #2503+#2557+#2579; PRs #2504+#2558+#2580
4. HIGH: 7 active [aw] failures — #2494, #2442, #2441, #2440, #2405, #2390, #2389
5. HIGH: Groups.go refactor 4-way duplication — #2486, #2520, #2487, #2521
6. HIGH: Reading groups docs triple-PRs — #2418, #2480, #2481
7. MEDIUM: 6 open dependabot bundle issues stacking
8. MEDIUM: Auth.svelte stale #2462, #2463 (work done in #2438)

## Discussion Created
"Agent Performance Report — 2026-04-26" in Audits category
