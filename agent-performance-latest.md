# Agent Performance — 2026-04-25
**Run:** 2026-04-25T23:44Z

## Snapshot
- 55 workflows registered, all Copilot; **repo-assist NOT compiled (Day 7 ⚠️ CRITICAL)**
- 45 open issues (up from 44 yesterday)
- 27 open PRs (up from 25)
- [aw] Failures: **7 active** (stable, no new failures today)
- Status-as-issue pattern: **Day 6** — #2556 (repo-status), #2554 (team-status), #2553 (daily-plan)
- SSRF duplication: duplicate-code-detector created #2557 (DUPLICATES #2503) + #2558 PR
- Groups.go refactoring duplication: 4 items — issues #2486+#2520, PRs #2487+#2521 (unchanged)
- Reading groups docs triple-duplicate PRs: #2418, #2480, #2481 (Day 4+ — unchanged)
- Open dependabot bundles: now **5 open** (#2549 added today)
- Auth.svelte stale: #2462 (issue) + #2463 (PR) still open despite #2438 merged Apr 22

## Top Performers
- **discussion-task-miner**: 4 actionable tasks #2542-#2545. Score: 82/100
- **sergo**: #2548 GA versions update. Score: 75/100
- **tech-content-editorial-board**: #2532 security editorial (carry-forward). Score: 68/100
- **duplicate-code-detector**: #2538+#2539 valid (group check), but #2557 duplicates #2503. Score: 62/100

## Agents Needing Improvement
- **repo-assist**: NOT COMPILED + FAILED (#2389). Score: 10/100 — CRITICAL (Day 7)
- **daily-repo-status**: Status-as-issue Day 6 (#2556). Score: 15/100
- **daily-team-status**: Status-as-issue Day 6 (#2554). Score: 15/100
- **daily-plan**: Status-as-issue Day 6 (#2553). Score: 20/100
- **daily-repo-chronicle**: Creates issue #2528 + [aw] failure #2494. Score: 25/100
- **code-simplifier**: Groups.go 4-way dup still open. Score: 38/100
- **daily-doc-updater**: Triple reading groups PRs (#2418/#2480/#2481). Score: 40/100
- **dependabot-pr-bundler**: 5 stacking open bundle issues. Score: 45/100

## Improvements vs Yesterday (Apr 24)
- No new [aw] failures today ✅
- discussion-task-miner: 4 productive tasks created ✅
- sergo: #2548 PR (GA versions update) ✅

## Regressions vs Yesterday
- Status-as-issue pattern Day 6 (no fix applied) ❌
- repo-assist NOT COMPILED Day 7 ❌
- duplicate-code-detector: SSRF self-duplication (#2557 duplicates #2503) ❌
- Dependabot bundles: 5th open bundle #2549 added ❌
- Auth.svelte stale items (#2462, #2463) persist after #2438 merged ❌

## Critical Issues
1. CRITICAL: repo-assist NOT COMPILED (Day 7) — #2389
2. CRITICAL: Status-as-Issue Pattern Day 6 — #2556, #2554, #2553 (+ historical)
3. HIGH: 7 active [aw] failures — #2494, #2442, #2441, #2440, #2405, #2390, #2389
4. HIGH: Groups.go refactor 4-way duplication — #2486, #2520, #2487, #2521
5. HIGH: Reading groups docs triple-PRs — #2418, #2480, #2481
6. HIGH: SSRF duplicate issue + PR — #2557 dup of #2503; #2558 dup of #2504
7. MEDIUM: 5 open dependabot bundle issues stacking
8. MEDIUM: Auth.svelte stale #2462, #2463 (work done in #2438)

## Discussion Created
"Agent Performance Report — 2026-04-25" in Audits category
